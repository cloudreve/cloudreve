use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::path::PathBuf;

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
pub enum TaskKind {
    Upload,
    Download,
}

impl TaskKind {
    pub fn as_str(&self) -> &'static str {
        match self {
            TaskKind::Upload => "upload",
            TaskKind::Download => "download",
        }
    }

    pub fn from_str(value: &str) -> Option<Self> {
        match value {
            "upload" => Some(TaskKind::Upload),
            "download" => Some(TaskKind::Download),
            _ => None,
        }
    }
}

#[derive(Debug, Clone)]
pub struct TaskPayload {
    pub task_id: Option<String>,
    pub kind: TaskKind,
    pub local_path: PathBuf,
    pub priority: i32,
    pub total_bytes: Option<i64>,
    pub processed_bytes: Option<i64>,
    pub custom_state: Option<Value>,

    // Upload
    pub force_override: bool,
}

impl TaskPayload {
    pub fn new(kind: TaskKind, local_path: impl Into<PathBuf>) -> Self {
        Self {
            task_id: None,
            kind,
            local_path: local_path.into(),
            priority: 0,
            total_bytes: None,
            processed_bytes: None,
            custom_state: None,
            force_override: false,
        }
    }

    pub fn upload(local_path: impl Into<PathBuf>) -> Self {
        Self::new(TaskKind::Upload, local_path)
    }

    pub fn download(local_path: impl Into<PathBuf>) -> Self {
        Self::new(TaskKind::Download, local_path)
    }

    pub fn with_task_id(mut self, id: impl Into<String>) -> Self {
        self.task_id = Some(id.into());
        self
    }

    pub fn with_priority(mut self, priority: i32) -> Self {
        self.priority = priority;
        self
    }

    pub fn with_totals(mut self, processed: i64, total: i64) -> Self {
        self.processed_bytes = Some(processed);
        self.total_bytes = Some(total);
        self
    }

    pub fn with_custom_state(mut self, state: Value) -> Self {
        self.custom_state = Some(state);
        self
    }

    pub fn with_force_override(mut self, force: bool) -> Self {
        self.force_override = force;
        self
    }

    pub fn local_path_display(&self) -> String {
        self.local_path.as_path().to_string_lossy().into_owned()
    }

    pub fn custom_state(&self) -> Option<&Value> {
        self.custom_state.as_ref()
    }
}

#[derive(Debug, Clone, Serialize)]
pub struct TaskProgress {
    pub task_id: String,
    pub kind: TaskKind,
    pub local_path: String,
    /// Progress percentage (0.0 - 1.0)
    pub progress: f64,
    /// Bytes processed so far
    pub processed_bytes: Option<i64>,
    /// Total bytes to process
    pub total_bytes: Option<i64>,
    /// Upload/download speed in bytes per second
    pub speed_bytes_per_sec: u64,
    /// Estimated time remaining in seconds
    pub eta_seconds: Option<u64>,
    /// Custom state for task-specific data
    pub custom_state: Option<Value>,
}

impl TaskProgress {
    pub fn from_payload(task_id: impl Into<String>, payload: &TaskPayload) -> Self {
        Self {
            task_id: task_id.into(),
            kind: payload.kind,
            local_path: payload.local_path_display(),
            progress: 0.0,
            processed_bytes: payload.processed_bytes,
            total_bytes: payload.total_bytes,
            speed_bytes_per_sec: 0,
            eta_seconds: None,
            custom_state: payload.custom_state.clone(),
        }
    }

    /// Update progress with speed and ETA information
    pub fn update_with_speed(
        &mut self,
        progress: f64,
        processed_bytes: i64,
        total_bytes: i64,
        speed_bytes_per_sec: u64,
        eta_seconds: Option<u64>,
    ) {
        self.progress = progress;
        self.processed_bytes = Some(processed_bytes);
        self.total_bytes = Some(total_bytes);
        self.speed_bytes_per_sec = speed_bytes_per_sec;
        self.eta_seconds = eta_seconds;
    }

    /// Update progress from ProgressUpdate
    pub fn update_from_progress(&mut self, update: &crate::uploader::ProgressUpdate) {
        self.progress = update.progress;
        self.processed_bytes = Some(update.uploaded as i64);
        self.total_bytes = Some(update.total_size as i64);
        self.speed_bytes_per_sec = update.speed_bytes_per_sec;
        self.eta_seconds = update.eta_seconds;
        tracing::debug!(update=?update, current = ?self, "Updated progress from ProgressUpdate");
    }

    pub fn update(
        &mut self,
        progress: f64,
        processed_bytes: Option<i64>,
        total_bytes: Option<i64>,
        custom_state: Option<Value>,
    ) {
        self.progress = progress;
        if let Some(bytes) = processed_bytes {
            self.processed_bytes = Some(bytes);
        }
        if let Some(total) = total_bytes {
            self.total_bytes = Some(total);
        }
        if let Some(state) = custom_state {
            self.custom_state = Some(state);
        }
    }
}
