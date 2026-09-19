use crate::config::ConfigManager;
use crate::inventory::{InventoryDb, NewTaskRecord, TaskRecord, TaskStatus, TaskUpdate};
use crate::tasks::download::DownloadTask;
use crate::tasks::types::{TaskKind, TaskPayload, TaskProgress};
use crate::tasks::upload::UploadTask;
use anyhow::{Context, Result, anyhow};
use cloudreve_api::Client;
use dashmap::DashMap;
use serde_json::Value;
use std::io::ErrorKind;
use std::path::{Path, PathBuf};
use std::sync::Arc;
use std::sync::atomic::{AtomicBool, AtomicUsize, Ordering};
use std::time::{Duration, SystemTime};
use tokio::sync::{
    Mutex, Notify, Semaphore,
    mpsc::{self, UnboundedReceiver, UnboundedSender},
};
use tokio::task::JoinHandle;
use tokio::time::{Instant, sleep};
use tracing::{debug, error, info, warn};
use uuid::Uuid;

#[derive(Debug, Clone)]
pub struct TaskQueueConfig {
    pub max_concurrent: usize,
}

impl Default for TaskQueueConfig {
    fn default() -> Self {
        Self { max_concurrent: 2 }
    }
}

pub struct TaskQueue {
    pub drive_id: String,
    pub cr_client: Arc<Client>,
    pub inventory: Arc<InventoryDb>,
    pub sync_path: PathBuf,
    pub remote_base: String,
    config: TaskQueueConfig,
    semaphore: Arc<Semaphore>,
    command_tx: UnboundedSender<QueueCommand>,
    dispatcher_handle: Mutex<Option<JoinHandle<()>>>,
    inflight: AtomicUsize,
    idle_notify: Notify,
    shutting_down: AtomicBool,
    cancel_requested: AtomicBool,
    progress: Arc<DashMap<String, TaskProgress>>,
    task_handles: DashMap<String, JoinHandle<()>>,
    /// Maps task_id to local_path for running tasks, used for path-based cancellation
    task_paths: DashMap<String, String>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
struct FileChangeSignature {
    size: u64,
    modified: SystemTime,
}

impl TaskQueue {
    pub async fn new(
        drive_id: impl Into<String>,
        cr_client: Arc<Client>,
        inventory: Arc<InventoryDb>,
        config: TaskQueueConfig,
        sync_path: PathBuf,
        remote_base: String,
    ) -> Arc<Self> {
        let drive_id = drive_id.into();
        let max_concurrent = config.max_concurrent.max(1);
        let sanitized_config = TaskQueueConfig { max_concurrent };

        let (command_tx, command_rx) = mpsc::unbounded_channel();
        let queue = Arc::new(Self {
            drive_id,
            inventory,
            cr_client,
            sync_path,
            remote_base,
            config: sanitized_config,
            semaphore: Arc::new(Semaphore::new(max_concurrent)),
            command_tx,
            dispatcher_handle: Mutex::new(None),
            inflight: AtomicUsize::new(0),
            idle_notify: Notify::new(),
            shutting_down: AtomicBool::new(false),
            cancel_requested: AtomicBool::new(false),
            progress: Arc::new(DashMap::new()),
            task_handles: DashMap::new(),
            task_paths: DashMap::new(),
        });

        queue.spawn_dispatcher(command_rx).await;
        if let Err(err) = queue.resume_incomplete_tasks().await {
            warn!(
                target: "tasks::queue",
                drive = %queue.drive_id,
                error = %err,
                "Failed to resume pending tasks from inventory"
            );
        }
        queue
    }

    pub fn max_concurrent(&self) -> usize {
        self.config.max_concurrent
    }

    pub fn drive_id(&self) -> &str {
        &self.drive_id
    }

    pub async fn enqueue(&self, payload: TaskPayload) -> Result<String> {
        if self.shutting_down.load(Ordering::SeqCst) {
            return Err(anyhow!("task queue is shutting down"));
        }

        let task_id = payload
            .task_id
            .clone()
            .unwrap_or_else(|| Uuid::new_v4().to_string());

        let mut record = NewTaskRecord::new(
            task_id.clone(),
            self.drive_id.clone(),
            payload.kind.as_str().to_string(),
            payload.local_path_display(),
        )
        .with_priority(payload.priority);

        match (payload.total_bytes, payload.processed_bytes) {
            (Some(total), Some(processed)) => {
                record = record.with_totals(total, processed);
            }
            (Some(total), None) => {
                record = record.with_totals(total, 0);
            }
            (None, Some(processed)) => {
                record = record.with_totals(0, processed);
            }
            _ => {}
        }

        if let Some(state) = payload.custom_state.clone() {
            record = record.with_custom_state(state);
        }

        let inserted = self
            .inventory
            .insert_task_if_not_exist(&record)
            .with_context(|| format!("Failed to persist task {}", task_id))?;

        if !inserted {
            return Err(anyhow!(
                "Task already exists for {} with type {}",
                payload.local_path_display(),
                payload.kind.as_str()
            ));
        }

        let payload = payload.with_task_id(task_id.clone());
        self.dispatch_task(task_id.clone(), payload)?;
        Ok(task_id)
    }

    pub fn list_active_tasks(&self) -> Result<Vec<TaskRecord>> {
        self.inventory.list_tasks(
            Some(&self.drive_id),
            Some(&[TaskStatus::Pending, TaskStatus::Running]),
        )
    }

    pub async fn ongoing_progress(&self) -> Vec<TaskProgress> {
        self.progress
            .iter()
            .map(|entry| entry.value().clone())
            .collect()
    }

    fn dispatch_task(&self, task_id: String, payload: TaskPayload) -> Result<()> {
        let command = QueueCommand::Enqueue(QueuedTask { task_id, payload });
        self.command_tx
            .send(command)
            .context("Task dispatcher closed")?;
        Ok(())
    }

    pub async fn persist_progress(
        &self,
        task_id: &str,
        progress: f64,
        processed_bytes: Option<i64>,
        total_bytes: Option<i64>,
        custom_state: Option<Value>,
    ) -> Result<()> {
        let clamped = progress.clamp(0.0, 1.0);
        if let Some(mut entry) = self.progress.get_mut(task_id) {
            entry.update(clamped, processed_bytes, total_bytes, custom_state);
            Ok(())
        } else {
            Err(anyhow!("No progress entry for task {}", task_id))
        }
    }

    pub async fn shutdown(&self) {
        if self.shutting_down.swap(true, Ordering::SeqCst) {
            return;
        }

        self.cancel_requested.store(true, Ordering::SeqCst);

        if let Err(err) = self.command_tx.send(QueueCommand::Shutdown) {
            warn!(target: "tasks::queue", error = %err, "Task queue dispatcher already closed");
        }

        if let Some(handle) = self.dispatcher_handle.lock().await.take() {
            handle.abort();
        }

        self.cancel_running_tasks().await;
        self.task_handles.clear();
        self.task_paths.clear();
        self.progress.clear();
    }

    /// Cancel all tasks for a given path or its descendants.
    /// This will:
    /// 1. Mark pending tasks in inventory as cancelled
    /// 2. Abort running tasks that match the path
    /// 3. Tasks in the channel queue will check their status upon scheduling and exit early
    ///
    /// Returns the number of tasks that were cancelled.
    pub async fn cancel_by_path(&self, path: impl AsRef<std::path::Path>) -> Result<usize> {
        let path_str = path.as_ref().to_string_lossy().to_string();

        info!(
            target: "tasks::queue",
            drive = %self.drive_id,
            path = %path_str,
            "Cancelling tasks by path"
        );

        // 1. Cancel pending tasks in inventory (this also marks running tasks as cancelled)
        let cancelled_ids = self
            .inventory
            .cancel_tasks_by_path(&self.drive_id, &path_str)
            .context("Failed to cancel tasks in inventory")?;

        let cancelled_count = cancelled_ids.len();

        // 2. Abort running task handles that match the path
        let tasks_to_abort: Vec<String> = self
            .task_paths
            .iter()
            .filter(|entry| {
                let task_path = entry.value();
                task_path == &path_str
                    || task_path.starts_with(&format!("{}{}", path_str, std::path::MAIN_SEPARATOR))
            })
            .map(|entry| entry.key().clone())
            .collect();

        for task_id in tasks_to_abort {
            if let Some((_, handle)) = self.task_handles.remove(&task_id) {
                handle.abort();
                debug!(
                    target: "tasks::queue",
                    drive = %self.drive_id,
                    task_id = %task_id,
                    "Aborted running task"
                );
            }
            self.task_paths.remove(&task_id);
            self.progress.remove(&task_id);
        }

        if cancelled_count > 0 {
            info!(
                target: "tasks::queue",
                drive = %self.drive_id,
                path = %path_str,
                count = cancelled_count,
                "Cancelled tasks by path"
            );
        }

        Ok(cancelled_count)
    }

    async fn spawn_dispatcher(self: &Arc<Self>, command_rx: UnboundedReceiver<QueueCommand>) {
        let queue = Arc::clone(self);
        let handle = tokio::spawn(async move {
            queue.run_dispatch_loop(command_rx).await;
        });
        *self.dispatcher_handle.lock().await = Some(handle);
    }

    async fn run_dispatch_loop(self: Arc<Self>, mut command_rx: UnboundedReceiver<QueueCommand>) {
        info!(
            target: "tasks::queue",
            drive = %self.drive_id,
            concurrency = self.config.max_concurrent,
            "Task queue dispatcher started"
        );

        while let Some(command) = command_rx.recv().await {
            match command {
                QueueCommand::Enqueue(task) => {
                    self.launch_task(task).await;
                }
                QueueCommand::Shutdown => {
                    debug!(
                        target: "tasks::queue",
                        drive = %self.drive_id,
                        "Task queue dispatcher shutting down"
                    );
                    break;
                }
            }
        }

        info!(
            target: "tasks::queue",
            drive = %self.drive_id,
            "Task queue dispatcher stopped"
        );
    }

    async fn launch_task(self: &Arc<Self>, task: QueuedTask) {
        let permit = match self.semaphore.clone().acquire_owned().await {
            Ok(permit) => permit,
            Err(err) => {
                error!(
                    target: "tasks::queue",
                    drive = %self.drive_id,
                    error = %err,
                    "Failed to acquire semaphore permit"
                );
                if let Err(update_err) = self.inventory.update_task(
                    &task.task_id,
                    TaskUpdate {
                        status: Some(TaskStatus::Failed),
                        error: Some(Some("Failed to schedule task".to_string())),
                        ..Default::default()
                    },
                ) {
                    warn!(
                        target: "tasks::queue",
                        drive = %self.drive_id,
                        error = %update_err,
                        "Failed to persist scheduling failure"
                    );
                }
                return;
            }
        };

        self.inflight.fetch_add(1, Ordering::SeqCst);
        let queue_for_execute = Arc::clone(self);
        let queue_for_notify = Arc::clone(self);
        let task_id = task.task_id.clone();
        let handle_task_id = task_id.clone();

        let handle = tokio::spawn(async move {
            queue_for_execute.execute_task(task).await;
            drop(permit);
            queue_for_notify.inflight.fetch_sub(1, Ordering::SeqCst);
            queue_for_notify.idle_notify.notify_waiters();
            queue_for_notify.task_handles.remove(&handle_task_id);
        });

        self.task_handles.insert(task_id, handle);
    }

    async fn execute_task(self: Arc<Self>, task: QueuedTask) {
        // Check if task was cancelled while in the channel queue
        match self.inventory.get_task_status(&task.task_id) {
            Ok(Some(TaskStatus::Cancelled)) => {
                debug!(
                    target: "tasks::queue",
                    drive = %self.drive_id,
                    task_id = %task.task_id,
                    "Task was cancelled before execution, skipping"
                );
                return;
            }
            Ok(Some(status)) if !status.is_active() => {
                debug!(
                    target: "tasks::queue",
                    drive = %self.drive_id,
                    task_id = %task.task_id,
                    status = ?status,
                    "Task is no longer active, skipping"
                );
                return;
            }
            Err(err) => {
                warn!(
                    target: "tasks::queue",
                    drive = %self.drive_id,
                    task_id = %task.task_id,
                    error = %err,
                    "Failed to check task status, proceeding with execution"
                );
            }
            _ => {}
        }

        if let Err(err) = self.inventory.update_task(
            &task.task_id,
            TaskUpdate {
                status: Some(TaskStatus::Running),
                ..Default::default()
            },
        ) {
            error!(
                target: "tasks::queue",
                drive = %self.drive_id,
                task_id = %task.task_id,
                error = %err,
                "Failed to mark task as running"
            );
            return;
        }

        // Register task path for path-based cancellation
        self.task_paths
            .insert(task.task_id.clone(), task.payload.local_path_display());

        self.register_progress_entry(&task).await;

        match self.run_placeholder_task(&task).await {
            Ok(TaskRunState::Completed) => {
                if let Err(err) = self.inventory.update_task(
                    &task.task_id,
                    TaskUpdate {
                        status: Some(TaskStatus::Completed),
                        ..Default::default()
                    },
                ) {
                    warn!(
                        target: "tasks::queue",
                        drive = %self.drive_id,
                        task_id = %task.task_id,
                        error = %err,
                        "Failed to mark task as completed"
                    );
                }
            }
            Ok(TaskRunState::Cancelled) => {
                if let Err(err) = self.inventory.update_task(
                    &task.task_id,
                    TaskUpdate {
                        status: Some(TaskStatus::Cancelled),
                        ..Default::default()
                    },
                ) {
                    warn!(
                        target: "tasks::queue",
                        drive = %self.drive_id,
                        task_id = %task.task_id,
                        error = %err,
                        "Failed to mark task as cancelled"
                    );
                }
                self.cleanup_task_entry(&task.task_id).await;
                return;
            }
            Err(err) => {
                error!(
                    target: "tasks::queue",
                    drive = %self.drive_id,
                    task_id = %task.task_id,
                    error = ?err,
                    "Task execution failed"
                );
                if let Err(update_err) = self.inventory.update_task(
                    &task.task_id,
                    TaskUpdate {
                        status: Some(TaskStatus::Failed),
                        error: Some(Some(format!("{:?}", err))),
                        ..Default::default()
                    },
                ) {
                    warn!(
                        target: "tasks::queue",
                        drive = %self.drive_id,
                        task_id = %task.task_id,
                        error = %update_err,
                        "Failed to persist task failure state"
                    );
                }
                self.cleanup_task_entry(&task.task_id).await;
                return;
            }
        }

        self.cleanup_task_entry(&task.task_id).await;
    }

    async fn run_placeholder_task(&self, task: &QueuedTask) -> Result<TaskRunState> {
        info!(
            target: "tasks::queue",
            drive = %self.drive_id,
            task_id = %task.task_id,
            kind = task.payload.kind.as_str(),
            path = %task.payload.local_path_display(),
            "Executing task"
        );

        match &task.payload.kind {
            TaskKind::Upload => {
                match self.wait_for_upload_stability(task).await? {
                    TaskRunState::Cancelled => return Ok(TaskRunState::Cancelled),
                    TaskRunState::Completed => {}
                }

                let mut task_executor = UploadTask::new(
                    self.inventory.clone(),
                    self.cr_client.clone(),
                    self.drive_id.as_str(),
                    &task,
                    self.sync_path.clone(),
                    self.remote_base.clone(),
                    Arc::clone(&self.progress),
                );

                task_executor.execute().await?;
            }
            TaskKind::Download => {
                let mut task_executor = DownloadTask::new(
                    self.inventory.clone(),
                    self.cr_client.clone(),
                    self.drive_id.as_str(),
                    &task,
                    self.sync_path.clone(),
                    self.remote_base.clone(),
                    Arc::clone(&self.progress),
                );

                task_executor.execute().await?;
            }
        }

        // for step in 0..PLACEHOLDER_STEPS {
        //     if self.cancel_requested.load(Ordering::SeqCst) {
        //         return Ok(TaskRunState::Cancelled);
        //     }
        //     sleep(Duration::from_millis(250)).await;
        //     let progress = (step + 1) as f64 / PLACEHOLDER_STEPS as f64;
        //     let direction = match task.payload.kind {
        //         TaskKind::Upload => "upload",
        //         TaskKind::Download => "download",
        //     };
        //     let state = json!({
        //         "step": step + 1,
        //         "total_steps": PLACEHOLDER_STEPS,
        //         "local_path": task.payload.local_path_display(),
        //         "kind": direction,
        //     });
        //     if let Err(err) = self
        //         .persist_progress(
        //             &task.task_id,
        //             progress,
        //             task.payload.processed_bytes,
        //             task.payload.total_bytes,
        //             Some(state),
        //         )
        //         .await
        //     {
        //         warn!(
        //             target: "tasks::queue",
        //             drive = %self.drive_id,
        //             task_id = %task.task_id,
        //             error = %err,
        //             "Failed to persist placeholder progress"
        //         );
        //     }
        // }

        Ok(TaskRunState::Completed)
    }

    fn is_task_marked_inactive(&self, task_id: &str) -> bool {
        match self.inventory.get_task_status(task_id) {
            Ok(Some(TaskStatus::Cancelled)) => true,
            Ok(Some(status)) => !status.is_active(),
            Err(err) => {
                warn!(
                    target: "tasks::queue",
                    drive = %self.drive_id,
                    task_id = %task_id,
                    error = %err,
                    "Failed to check task status while waiting for delayed upload"
                );
                false
            }
            _ => false,
        }
    }

    fn capture_file_signature(path: &Path) -> Result<Option<FileChangeSignature>> {
        let metadata = match std::fs::metadata(path) {
            Ok(metadata) => metadata,
            Err(err) if err.kind() == ErrorKind::NotFound => return Ok(None),
            Err(err) => {
                return Err(err)
                    .with_context(|| format!("failed to read metadata for {}", path.display()));
            }
        };
        if !metadata.is_file() {
            return Ok(None);
        }

        let modified = metadata
            .modified()
            .with_context(|| format!("failed to read modified time for {}", path.display()))?;

        Ok(Some(FileChangeSignature {
            size: metadata.len(),
            modified,
        }))
    }

    async fn wait_for_upload_stability(&self, task: &QueuedTask) -> Result<TaskRunState> {
        let delay_seconds = ConfigManager::try_get()
            .map(|config| config.sync_delay_seconds())
            .unwrap_or(0);

        if delay_seconds == 0 {
            return Ok(TaskRunState::Completed);
        }

        let local_path = &task.payload.local_path;
        let mut signature = match Self::capture_file_signature(local_path.as_path()) {
            Ok(Some(signature)) => signature,
            Ok(None) => return Ok(TaskRunState::Completed),
            Err(err) => {
                warn!(
                    target: "tasks::queue",
                    drive = %self.drive_id,
                    task_id = %task.task_id,
                    path = %local_path.display(),
                    error = %err,
                    "Failed to capture initial file signature, skipping delayed upload wait"
                );
                return Ok(TaskRunState::Completed);
            }
        };

        info!(
            target: "tasks::queue",
            drive = %self.drive_id,
            task_id = %task.task_id,
            path = %local_path.display(),
            delay_seconds = delay_seconds,
            "Waiting for file to stop changing before upload"
        );

        let required_stable = Duration::from_secs(delay_seconds);
        // A continuously-growing file must not block a worker slot forever:
        // give up waiting after 4x the configured delay (min 10 minutes) and
        // upload whatever is on disk, rather than starving the queue.
        let max_total_wait = required_stable
            .saturating_mul(4)
            .max(Duration::from_secs(600));
        let wait_started = Instant::now();
        let poll_interval = if delay_seconds <= 30 {
            Duration::from_secs(1)
        } else if delay_seconds <= 120 {
            Duration::from_secs(2)
        } else {
            Duration::from_secs(3)
        };
        let status_check_interval = Duration::from_secs(5);
        let mut unchanged_since = Instant::now();
        let mut next_status_check = Instant::now();

        while unchanged_since.elapsed() < required_stable {
            if wait_started.elapsed() >= max_total_wait {
                warn!(
                    target: "tasks::queue",
                    drive = %self.drive_id,
                    task_id = %task.task_id,
                    path = %local_path.display(),
                    "File still changing after maximum wait, proceeding with upload"
                );
                return Ok(TaskRunState::Completed);
            }

            if self.cancel_requested.load(Ordering::SeqCst) {
                debug!(
                    target: "tasks::queue",
                    drive = %self.drive_id,
                    task_id = %task.task_id,
                    path = %local_path.display(),
                    "Task cancelled while waiting for upload delay"
                );
                return Ok(TaskRunState::Cancelled);
            }

            if Instant::now() >= next_status_check {
                if self.is_task_marked_inactive(&task.task_id) {
                    debug!(
                        target: "tasks::queue",
                        drive = %self.drive_id,
                        task_id = %task.task_id,
                        path = %local_path.display(),
                        "Task became inactive while waiting for upload delay"
                    );
                    return Ok(TaskRunState::Cancelled);
                }
                next_status_check = Instant::now() + status_check_interval;
            }

            sleep(poll_interval).await;

            let current_signature = match Self::capture_file_signature(local_path.as_path()) {
                Ok(Some(signature)) => signature,
                Ok(None) => return Ok(TaskRunState::Completed),
                Err(err) => {
                    warn!(
                        target: "tasks::queue",
                        drive = %self.drive_id,
                        task_id = %task.task_id,
                        path = %local_path.display(),
                        error = %err,
                        "Failed to capture file signature while waiting, continuing upload"
                    );
                    return Ok(TaskRunState::Completed);
                }
            };

            if current_signature != signature {
                signature = current_signature;
                unchanged_since = Instant::now();
            }
        }

        debug!(
            target: "tasks::queue",
            drive = %self.drive_id,
            task_id = %task.task_id,
            path = %local_path.display(),
            delay_seconds = delay_seconds,
            "File remained unchanged for configured delay, proceeding with upload"
        );

        Ok(TaskRunState::Completed)
    }

    #[allow(dead_code)]
    async fn wait_for_idle(&self) {
        while self.inflight.load(Ordering::SeqCst) > 0 {
            self.idle_notify.notified().await;
        }
    }

    async fn register_progress_entry(&self, task: &QueuedTask) {
        self.progress.insert(
            task.task_id.clone(),
            TaskProgress::from_payload(&task.task_id, &task.payload),
        );
    }

    #[allow(dead_code)]
    async fn clear_progress_entry(&self, task_id: &str) {
        self.progress.remove(task_id);
    }

    async fn cleanup_task_entry(&self, task_id: &str) {
        self.progress.remove(task_id);
        self.task_paths.remove(task_id);
    }

    async fn resume_incomplete_tasks(self: &Arc<Self>) -> Result<()> {
        let records = self.inventory.list_tasks(
            Some(&self.drive_id),
            Some(&[TaskStatus::Pending, TaskStatus::Running]),
        )?;

        if records.is_empty() {
            return Ok(());
        }

        let mut resumed = 0usize;
        for record in records {
            if record.status == TaskStatus::Running {
                if let Err(err) = self.inventory.update_task(
                    &record.id,
                    TaskUpdate {
                        status: Some(TaskStatus::Pending),
                        ..Default::default()
                    },
                ) {
                    warn!(
                        target: "tasks::queue",
                        drive = %self.drive_id,
                        task_id = %record.id,
                        error = ?err,
                        "Failed to reset task status during resume"
                    );
                    continue;
                }
            }

            let payload = match Self::payload_from_record(&record) {
                Ok(payload) => payload,
                Err(err) => {
                    warn!(
                        target: "tasks::queue",
                        drive = %self.drive_id,
                        task_id = %record.id,
                        error = %err,
                        "Failed to build payload for resumed task"
                    );
                    continue;
                }
            };

            if let Err(err) = self.dispatch_task(record.id.clone(), payload) {
                warn!(
                    target: "tasks::queue",
                    drive = %self.drive_id,
                    task_id = %record.id,
                    error = ?err,
                    "Failed to dispatch resumed task"
                );
                continue;
            }

            resumed += 1;
        }

        if resumed > 0 {
            info!(
                target: "tasks::queue",
                drive = %self.drive_id,
                count = resumed,
                "Resumed pending tasks from inventory"
            );
        }

        Ok(())
    }

    async fn cancel_running_tasks(&self) {
        let running: Vec<String> = self
            .progress
            .iter()
            .map(|entry| entry.key().clone())
            .collect();

        for task_id in running {
            if let Some((_, handle)) = self.task_handles.remove(&task_id) {
                handle.abort();
            }

            self.progress.remove(&task_id);
            self.task_paths.remove(&task_id);
        }
    }

    fn payload_from_record(record: &TaskRecord) -> Result<TaskPayload> {
        let kind = TaskKind::from_str(&record.task_type)
            .ok_or_else(|| anyhow!("Unknown task type {}", record.task_type))?;

        let mut payload = TaskPayload::new(kind, PathBuf::from(&record.local_path))
            .with_priority(record.priority)
            .with_task_id(record.id.clone());

        let total_bytes = record.total_bytes;
        let processed_bytes = record.processed_bytes;
        if total_bytes != 0 || processed_bytes != 0 {
            payload = payload.with_totals(processed_bytes, total_bytes);
        }

        if let Some(state) = &record.custom_state {
            payload = payload.with_custom_state(state.clone());
        }

        Ok(payload)
    }
}

#[allow(dead_code)]
pub enum TaskRunState {
    Completed,
    Cancelled,
}

enum QueueCommand {
    Enqueue(QueuedTask),
    Shutdown,
}

pub struct QueuedTask {
    pub task_id: String,
    pub payload: TaskPayload,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn file_signature_captures_size_and_mtime() {
        let dir = tempfile::tempdir().unwrap();
        let file = dir.path().join("data.bin");
        std::fs::write(&file, b"hello").unwrap();

        let signature = TaskQueue::capture_file_signature(&file)
            .unwrap()
            .expect("file should produce a signature");
        assert_eq!(signature.size, 5);

        std::fs::write(&file, b"hello world").unwrap();
        let updated = TaskQueue::capture_file_signature(&file)
            .unwrap()
            .unwrap();
        assert_ne!(signature, updated);
    }

    #[test]
    fn file_signature_none_for_dir_and_missing() {
        let dir = tempfile::tempdir().unwrap();
        assert!(TaskQueue::capture_file_signature(dir.path())
            .unwrap()
            .is_none());
        assert!(TaskQueue::capture_file_signature(
            &dir.path().join("missing.bin")
        )
        .unwrap()
        .is_none());
    }
}
