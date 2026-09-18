use cloudreve_api::models::explorer::StoragePolicy;
use cloudreve_api::models::user::{Capacity, UserSettings};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use uuid::Uuid;

/// Represents the conflict state of a file with the remote
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
pub enum ConflictState {
    /// File has a conflict, pending user action
    Pending,
    /// User chose to override the remote version
    Override,
}

impl ConflictState {
    pub fn as_str(&self) -> &'static str {
        match self {
            ConflictState::Pending => "pending",
            ConflictState::Override => "override",
        }
    }

    pub fn from_str(value: &str) -> Option<Self> {
        match value {
            "pending" => Some(ConflictState::Pending),
            "override" => Some(ConflictState::Override),
            _ => None,
        }
    }
}

/// Represents a file metadata entry in the inventory
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FileMetadata {
    pub id: i64,
    pub drive_id: Uuid,
    pub is_folder: bool,
    pub local_path: String,
    pub created_at: i64, // Unix timestamp
    pub updated_at: i64, // Unix timestamp
    pub etag: String,
    pub metadata: HashMap<String, String>,
    pub props: Option<serde_json::Value>,
    pub permissions: String,
    pub shared: bool,
    pub size: i64,
    pub conflict_state: Option<ConflictState>,
}

/// Entry for inserting or updating file metadata
#[derive(Debug, Clone)]
pub struct MetadataEntry {
    pub drive_id: Uuid,
    pub is_folder: bool,
    pub created_at: i64, // Unix timestamp
    pub updated_at: i64, // Unix timestamp
    pub local_path: String,
    pub etag: String,
    pub permissions: String,
    pub shared: bool,
    pub size: i64,
    pub metadata: HashMap<String, String>,
    pub props: Option<serde_json::Value>,
    pub conflict_state: Option<ConflictState>,
}

impl MetadataEntry {
    pub fn new(drive_id: Uuid, local_path: impl Into<String>, is_folder: bool) -> Self {
        Self {
            drive_id,
            is_folder,
            local_path: local_path.into(),
            created_at: chrono::Utc::now().timestamp(),
            updated_at: chrono::Utc::now().timestamp(),
            etag: String::new(),
            metadata: HashMap::new(),
            props: None,
            permissions: String::new(),
            shared: false,
            size: 0,
            conflict_state: None,
        }
    }

    pub fn with_permissions(mut self, permissions: impl Into<String>) -> Self {
        self.permissions = permissions.into();
        self
    }

    pub fn with_shared(mut self, shared: bool) -> Self {
        self.shared = shared;
        self
    }

    pub fn with_size(mut self, size: i64) -> Self {
        self.size = size;
        self
    }

    pub fn with_created_at(mut self, created_at: i64) -> Self {
        self.created_at = created_at;
        self
    }

    pub fn with_updated_at(mut self, updated_at: i64) -> Self {
        self.updated_at = updated_at;
        self
    }

    pub fn with_etag(mut self, etag: impl Into<String>) -> Self {
        self.etag = etag.into();
        self
    }

    pub fn with_metadata(mut self, metadata: HashMap<String, String>) -> Self {
        self.metadata = metadata;
        self
    }

    pub fn with_props(mut self, props: serde_json::Value) -> Self {
        self.props = Some(props);
        self
    }
}

impl From<&FileMetadata> for MetadataEntry {
    fn from(file_metadata: &FileMetadata) -> Self {
        Self {
            drive_id: file_metadata.drive_id.clone(),
            is_folder: file_metadata.is_folder,
            created_at: file_metadata.created_at,
            updated_at: file_metadata.updated_at.clone(),
            local_path: file_metadata.local_path.clone(),
            etag: file_metadata.etag.clone(),
            permissions: file_metadata.permissions.clone(),
            shared: file_metadata.shared,
            metadata: file_metadata.metadata.clone(),
            props: file_metadata.props.clone(),
            size: file_metadata.size,
            conflict_state: file_metadata.conflict_state,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TaskRecord {
    pub id: String,
    pub drive_id: String,
    pub task_type: String,
    pub local_path: String,
    pub status: TaskStatus,
    pub progress: f64,
    pub total_bytes: i64,
    pub processed_bytes: i64,
    pub priority: i32,
    pub custom_state: Option<serde_json::Value>,
    pub error: Option<String>,
    pub created_at: i64,
    pub updated_at: i64,
}

#[derive(Debug, Clone)]
pub struct NewTaskRecord {
    pub id: String,
    pub drive_id: String,
    pub task_type: String,
    pub local_path: String,
    pub status: TaskStatus,
    pub progress: f64,
    pub total_bytes: i64,
    pub processed_bytes: i64,
    pub priority: i32,
    pub custom_state: Option<serde_json::Value>,
    pub error: Option<String>,
    pub created_at: i64,
    pub updated_at: i64,
}

impl NewTaskRecord {
    pub fn new(
        id: impl Into<String>,
        drive_id: impl Into<String>,
        task_type: impl Into<String>,
        local_path: impl Into<String>,
    ) -> Self {
        let now = chrono::Utc::now().timestamp();
        Self {
            id: id.into(),
            drive_id: drive_id.into(),
            task_type: task_type.into(),
            local_path: local_path.into(),
            status: TaskStatus::Pending,
            progress: 0.0,
            total_bytes: 0,
            processed_bytes: 0,
            priority: 0,
            custom_state: None,
            error: None,
            created_at: now,
            updated_at: now,
        }
    }

    pub fn with_priority(mut self, priority: i32) -> Self {
        self.priority = priority;
        self
    }

    pub fn with_status(mut self, status: TaskStatus) -> Self {
        self.status = status;
        self
    }

    pub fn with_progress(mut self, progress: f64) -> Self {
        self.progress = progress;
        self
    }

    pub fn with_totals(mut self, total_bytes: i64, processed_bytes: i64) -> Self {
        self.total_bytes = total_bytes;
        self.processed_bytes = processed_bytes;
        self
    }

    pub fn with_custom_state(mut self, state: serde_json::Value) -> Self {
        self.custom_state = Some(state);
        self
    }

    pub fn with_error(mut self, error: impl Into<String>) -> Self {
        self.error = Some(error.into());
        self
    }

    pub fn touch(mut self) -> Self {
        self.updated_at = chrono::Utc::now().timestamp();
        self
    }
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
pub enum TaskStatus {
    Pending,
    Running,
    Completed,
    Failed,
    Cancelled,
}

impl TaskStatus {
    pub fn as_str(&self) -> &'static str {
        match self {
            TaskStatus::Pending => "pending",
            TaskStatus::Running => "running",
            TaskStatus::Completed => "completed",
            TaskStatus::Failed => "failed",
            TaskStatus::Cancelled => "cancelled",
        }
    }

    pub fn from_str(value: &str) -> Option<Self> {
        match value {
            "pending" => Some(TaskStatus::Pending),
            "running" => Some(TaskStatus::Running),
            "completed" => Some(TaskStatus::Completed),
            "failed" => Some(TaskStatus::Failed),
            "cancelled" => Some(TaskStatus::Cancelled),
            _ => None,
        }
    }

    pub fn is_active(&self) -> bool {
        matches!(self, TaskStatus::Pending | TaskStatus::Running)
    }
}

#[derive(Debug, Clone, Default)]
pub struct TaskUpdate {
    pub status: Option<TaskStatus>,
    pub progress: Option<f64>,
    pub total_bytes: Option<i64>,
    pub processed_bytes: Option<i64>,
    pub custom_state: Option<Option<serde_json::Value>>,
    pub error: Option<Option<String>>,
}

impl TaskUpdate {
    pub fn is_empty(&self) -> bool {
        self.status.is_none()
            && self.progress.is_none()
            && self.total_bytes.is_none()
            && self.processed_bytes.is_none()
            && self.custom_state.is_none()
            && self.error.is_none()
    }
}

/// Cached properties for a drive
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct DriveProps {
    pub id: i64,
    pub drive_id: String,
    pub capacity: Option<Capacity>,
    pub capacity_updated_at: Option<i64>,
    pub storage_policies: Option<Vec<StoragePolicy>>,
    pub storage_policies_updated_at: Option<i64>,
    pub user_settings: Option<UserSettings>,
    pub user_settings_updated_at: Option<i64>,
    pub created_at: i64,
    pub updated_at: i64,
}

/// Update entry for drive props
#[derive(Debug, Clone, Default)]
pub struct DrivePropsUpdate {
    pub capacity: Option<Option<Capacity>>,
    pub storage_policies: Option<Option<Vec<StoragePolicy>>>,
    pub user_settings: Option<Option<UserSettings>>,
}

impl DrivePropsUpdate {
    pub fn is_empty(&self) -> bool {
        self.capacity.is_none() && self.storage_policies.is_none() && self.user_settings.is_none()
    }

    pub fn with_capacity(mut self, capacity: Capacity) -> Self {
        self.capacity = Some(Some(capacity));
        self
    }

    pub fn with_storage_policies(mut self, policies: Vec<StoragePolicy>) -> Self {
        self.storage_policies = Some(Some(policies));
        self
    }

    pub fn with_user_settings(mut self, settings: UserSettings) -> Self {
        self.user_settings = Some(Some(settings));
        self
    }
}
