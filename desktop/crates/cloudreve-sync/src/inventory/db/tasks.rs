use super::InventoryDb;
use crate::inventory::{NewTaskRecord, TaskRecord, TaskStatus, TaskUpdate};
use anyhow::{Context, Result, anyhow};
use chrono::Utc;
use diesel::prelude::*;

use crate::inventory::schema::task_queue::{self, dsl as task_queue_dsl};

impl InventoryDb {
    /// Insert a task queue record if no pending/running task with the same type and path exists.
    /// Returns `true` if the task was inserted, `false` if a duplicate was found.
    pub fn insert_task_if_not_exist(&self, task: &NewTaskRecord) -> Result<bool> {
        let mut conn = self.connection()?;

        // Check if a pending or running task with the same type and path already exists
        let active_statuses = vec![
            TaskStatus::Pending.as_str().to_string(),
            TaskStatus::Running.as_str().to_string(),
        ];

        let existing: Option<String> = task_queue_dsl::task_queue
            .filter(task_queue_dsl::drive_id.eq(&task.drive_id))
            .filter(task_queue_dsl::task_type.eq(&task.task_type))
            .filter(task_queue_dsl::local_path.eq(&task.local_path))
            .filter(task_queue_dsl::status.eq_any(&active_statuses))
            .select(task_queue_dsl::id)
            .first(&mut conn)
            .optional()
            .context("Failed to check for existing task")?;

        if existing.is_some() {
            return Ok(false);
        }

        let row = NewTaskRow::try_from(task)?;
        diesel::insert_into(task_queue::table)
            .values(&row)
            .execute(&mut conn)
            .context("Failed to insert task queue record")?;
        Ok(true)
    }

    /// Update task queue record
    pub fn update_task(&self, task_id: &str, update: TaskUpdate) -> Result<()> {
        if update.is_empty() {
            return Ok(());
        }

        let mut conn = self.connection()?;
        let changeset = TaskChangeset::try_from(update)?;
        diesel::update(task_queue_dsl::task_queue.filter(task_queue_dsl::id.eq(task_id)))
            .set(changeset)
            .execute(&mut conn)?;
        Ok(())
    }

    /// List task queue records with optional filters
    pub fn list_tasks(
        &self,
        drive_id: Option<&str>,
        statuses: Option<&[TaskStatus]>,
    ) -> Result<Vec<TaskRecord>> {
        let mut conn = self.connection()?;
        let mut query = task_queue_dsl::task_queue.into_boxed();

        if let Some(drive) = drive_id {
            query = query.filter(task_queue_dsl::drive_id.eq(drive));
        }

        if let Some(status_filter) = statuses {
            let values: Vec<String> = status_filter
                .iter()
                .map(|status| status.as_str().to_string())
                .collect();
            query = query.filter(task_queue_dsl::status.eq_any(values));
        }

        let rows = query
            .order(task_queue_dsl::created_at.asc())
            .load::<TaskRow>(&mut conn)
            .context("Failed to query task queue records")?;

        rows.into_iter()
            .map(TaskRecord::try_from)
            .collect::<Result<Vec<_>>>()
    }

    /// Delete a completed/failed task entry
    pub fn delete_task(&self, task_id: &str) -> Result<()> {
        let mut conn = self.connection()?;
        diesel::delete(task_queue_dsl::task_queue.filter(task_queue_dsl::id.eq(task_id)))
            .execute(&mut conn)
            .context("Failed to delete task queue record")?;
        Ok(())
    }

    /// Cancel all pending/running tasks matching a path or its descendants.
    /// Returns the list of task IDs that were cancelled.
    pub fn cancel_tasks_by_path(&self, drive_id: &str, path: &str) -> Result<Vec<String>> {
        let mut conn = self.connection()?;

        // Find tasks that match the exact path or are descendants (path starts with "path/")
        let prefix = format!("{}{}", path, std::path::MAIN_SEPARATOR);
        let active_statuses = vec![
            TaskStatus::Pending.as_str().to_string(),
            TaskStatus::Running.as_str().to_string(),
        ];

        let matching_tasks: Vec<TaskRow> = task_queue_dsl::task_queue
            .filter(task_queue_dsl::drive_id.eq(drive_id))
            .filter(task_queue_dsl::status.eq_any(&active_statuses))
            .filter(
                task_queue_dsl::local_path
                    .eq(path)
                    .or(task_queue_dsl::local_path.like(format!("{}%", prefix))),
            )
            .load(&mut conn)
            .context("Failed to query tasks by path")?;

        let task_ids: Vec<String> = matching_tasks.iter().map(|t| t.id.clone()).collect();

        if !task_ids.is_empty() {
            let cancelled_status = TaskStatus::Cancelled.as_str().to_string();
            let now = chrono::Utc::now().timestamp();

            diesel::update(task_queue_dsl::task_queue.filter(task_queue_dsl::id.eq_any(&task_ids)))
                .set((
                    task_queue_dsl::status.eq(&cancelled_status),
                    task_queue_dsl::updated_at.eq(now),
                ))
                .execute(&mut conn)
                .context("Failed to cancel tasks by path")?;
        }

        Ok(task_ids)
    }

    /// Get task status by task ID
    pub fn get_task_status(&self, task_id: &str) -> Result<Option<TaskStatus>> {
        let mut conn = self.connection()?;
        let row: Option<String> = task_queue_dsl::task_queue
            .filter(task_queue_dsl::id.eq(task_id))
            .select(task_queue_dsl::status)
            .first(&mut conn)
            .optional()
            .context("Failed to query task status")?;

        match row {
            Some(status_str) => Ok(TaskStatus::from_str(&status_str)),
            None => Ok(None),
        }
    }

    /// Query recent tasks for status summary.
    /// Returns up to 25 pending/running tasks and up to 25 completed/failed/cancelled tasks,
    /// ordered by updated_at descending.
    pub fn query_recent_tasks(&self, drive_id: Option<&str>) -> Result<RecentTasks> {
        let mut conn = self.connection()?;

        // Query active tasks (pending/running) - limit 25, order by updated_at desc
        let active_statuses = vec![
            TaskStatus::Pending.as_str().to_string(),
            TaskStatus::Running.as_str().to_string(),
        ];

        let mut active_query = task_queue_dsl::task_queue
            .filter(task_queue_dsl::status.eq_any(&active_statuses))
            .into_boxed();

        if let Some(drive) = drive_id {
            active_query = active_query.filter(task_queue_dsl::drive_id.eq(drive));
        }

        let active_rows = active_query
            .order(task_queue_dsl::updated_at.desc())
            .limit(25)
            .load::<TaskRow>(&mut conn)
            .context("Failed to query active tasks")?;

        let active_tasks: Vec<TaskRecord> = active_rows
            .into_iter()
            .map(TaskRecord::try_from)
            .collect::<Result<Vec<_>>>()?;

        // Query finished tasks (completed/failed/cancelled) - limit 25, order by updated_at desc
        let finished_statuses = vec![
            TaskStatus::Completed.as_str().to_string(),
            TaskStatus::Failed.as_str().to_string(),
            TaskStatus::Cancelled.as_str().to_string(),
        ];

        let mut finished_query = task_queue_dsl::task_queue
            .filter(task_queue_dsl::status.eq_any(&finished_statuses))
            .into_boxed();

        if let Some(drive) = drive_id {
            finished_query = finished_query.filter(task_queue_dsl::drive_id.eq(drive));
        }

        let finished_rows = finished_query
            .order(task_queue_dsl::updated_at.desc())
            .limit(25)
            .load::<TaskRow>(&mut conn)
            .context("Failed to query finished tasks")?;

        let finished_tasks: Vec<TaskRecord> = finished_rows
            .into_iter()
            .map(TaskRecord::try_from)
            .collect::<Result<Vec<_>>>()?;

        Ok(RecentTasks {
            active: active_tasks,
            finished: finished_tasks,
        })
    }
}

/// Result of querying recent tasks
#[derive(Debug, Clone)]
pub struct RecentTasks {
    /// Pending and running tasks (up to 25)
    pub active: Vec<TaskRecord>,
    /// Completed, failed, and cancelled tasks (up to 25)
    pub finished: Vec<TaskRecord>,
}

// =========================================================================
// Row Types
// =========================================================================

#[derive(Queryable)]
struct TaskRow {
    id: String,
    drive_id: String,
    task_type: String,
    local_path: String,
    status: String,
    progress: f64,
    total_bytes: i64,
    processed_bytes: i64,
    priority: i32,
    custom_state: Option<String>,
    error: Option<String>,
    created_at: i64,
    updated_at: i64,
}

impl TryFrom<TaskRow> for TaskRecord {
    type Error = anyhow::Error;

    fn try_from(row: TaskRow) -> Result<Self> {
        let status = TaskStatus::from_str(&row.status)
            .ok_or_else(|| anyhow!("Unknown task status value {}", row.status))?;
        let custom_state = match row.custom_state {
            Some(json) => Some(
                serde_json::from_str(&json).context("Failed to deserialize task custom_state")?,
            ),
            None => None,
        };

        Ok(TaskRecord {
            id: row.id,
            drive_id: row.drive_id,
            task_type: row.task_type,
            local_path: row.local_path,
            status,
            progress: row.progress,
            total_bytes: row.total_bytes,
            processed_bytes: row.processed_bytes,
            priority: row.priority,
            custom_state,
            error: row.error,
            created_at: row.created_at,
            updated_at: row.updated_at,
        })
    }
}

#[derive(Insertable)]
#[diesel(table_name = task_queue)]
struct NewTaskRow {
    id: String,
    drive_id: String,
    task_type: String,
    local_path: String,
    status: String,
    progress: f64,
    total_bytes: i64,
    processed_bytes: i64,
    priority: i32,
    custom_state: Option<String>,
    error: Option<String>,
    created_at: i64,
    updated_at: i64,
}

impl TryFrom<&NewTaskRecord> for NewTaskRow {
    type Error = anyhow::Error;

    fn try_from(record: &NewTaskRecord) -> Result<Self> {
        Ok(Self {
            id: record.id.clone(),
            drive_id: record.drive_id.clone(),
            task_type: record.task_type.clone(),
            local_path: record.local_path.clone(),
            status: record.status.as_str().to_string(),
            progress: record.progress,
            total_bytes: record.total_bytes,
            processed_bytes: record.processed_bytes,
            priority: record.priority,
            custom_state: match &record.custom_state {
                Some(value) => Some(
                    serde_json::to_string(value)
                        .context("Failed to serialize task custom_state")?,
                ),
                None => None,
            },
            error: record.error.clone(),
            created_at: record.created_at,
            updated_at: record.updated_at,
        })
    }
}

#[derive(AsChangeset)]
#[diesel(table_name = task_queue)]
struct TaskChangeset {
    status: Option<String>,
    progress: Option<f64>,
    total_bytes: Option<i64>,
    processed_bytes: Option<i64>,
    custom_state: Option<Option<String>>,
    error: Option<Option<String>>,
    updated_at: i64,
}

impl TaskChangeset {
    fn try_from(update: TaskUpdate) -> Result<Self> {
        let custom_state = match update.custom_state {
            Some(Some(value)) => Some(Some(
                serde_json::to_string(&value).context("Failed to serialize task custom_state")?,
            )),
            Some(None) => Some(None),
            None => None,
        };

        let error_state = match update.error {
            Some(Some(err)) => Some(Some(err)),
            Some(None) => Some(None),
            None => None,
        };

        Ok(Self {
            status: update.status.map(|status| status.as_str().to_string()),
            progress: update.progress,
            total_bytes: update.total_bytes,
            processed_bytes: update.processed_bytes,
            custom_state,
            error: error_state,
            updated_at: Utc::now().timestamp(),
        })
    }
}
