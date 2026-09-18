use crate::drive::mounts::DriveConfig;
use crate::inventory::TaskRecord;
use crate::tasks::TaskProgress;
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct DriveState {
    pub drives: Vec<DriveConfig>,
}

/// Summary of the current status including drives and recent tasks
#[derive(Debug, Clone, Serialize)]
pub struct StatusSummary {
    /// All configured drives (unfiltered)
    pub drives: Vec<DriveConfig>,
    /// Active tasks (pending/running) with optional live progress info
    pub active_tasks: Vec<TaskWithProgress>,
    /// Recently finished tasks (completed/failed/cancelled)
    pub finished_tasks: Vec<TaskRecord>,
}

/// A task record with optional live progress information
#[derive(Debug, Clone, Serialize)]
pub struct TaskWithProgress {
    /// The task record from the database
    #[serde(flatten)]
    pub task: TaskRecord,
    /// Live progress information for running tasks (None if task is not currently running)
    pub live_progress: Option<TaskProgress>,
}

/// Capacity summary for UI display
#[derive(Debug, Clone, Serialize)]
pub struct CapacitySummary {
    /// Total capacity in bytes
    pub total: i64,
    /// Used capacity in bytes
    pub used: i64,
    /// Formatted label for display (e.g., "152.1 MB / 1.0 GB (14.9%)")
    pub label: String,
}

/// Sync status for UI display
#[derive(Debug, Clone, Serialize)]
pub enum SyncStatus {
    /// All files are in sync
    InSync,
    /// Currently syncing files
    Syncing,
    /// Sync is paused
    Paused,
    /// There was an error during sync
    Error,
}

/// Drive status information for the Windows Shell UI
#[derive(Debug, Clone, Serialize)]
pub struct DriveStatusUI {
    /// Drive display name
    pub name: String,
    /// Path to the raw (non-ICO) icon image
    pub raw_icon_path: Option<String>,
    /// Capacity summary (None if not available)
    pub capacity: Option<CapacitySummary>,
    /// URL to user profile page
    pub profile_url: String,
    /// URL to settings page
    pub settings_url: String,
    pub storage_url: String,
    /// Current sync status
    pub sync_status: SyncStatus,
    /// Number of active (pending/running) tasks
    pub active_task_count: usize,
}

/// Drive information for the settings UI
#[derive(Debug, Clone, Serialize)]
pub struct DriveInfo {
    /// Drive ID
    pub id: String,
    /// Drive display name
    pub name: String,
    /// Instance URL
    pub instance_url: String,
    pub remote_path: String,
    /// Local sync path
    pub sync_path: String,
    /// Path to the ICO icon
    pub icon_path: Option<String>,
    /// Path to the raw (non-ICO) icon image
    pub raw_icon_path: Option<String>,
    /// Whether the drive is enabled
    pub enabled: bool,
    /// User ID
    pub user_id: String,
    /// Current drive status
    pub status: DriveInfoStatus,
    /// Capacity summary (None if not available)
    pub capacity: Option<CapacitySummary>,
}

/// Drive status for the settings UI
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "snake_case")]
pub enum DriveInfoStatus {
    /// Drive is active and synced
    Active,
    // Event push subscription is lost
    EventPushLost,
    /// Credentials have expired
    CredentialExpired,
}

/// Format bytes into a human-readable string (e.g., "1.5 GB")
pub fn format_bytes(bytes: i64) -> String {
    const KB: f64 = 1024.0;
    const MB: f64 = KB * 1024.0;
    const GB: f64 = MB * 1024.0;
    const TB: f64 = GB * 1024.0;

    let bytes_f = bytes as f64;

    if bytes_f >= TB {
        format!("{:.1} TB", bytes_f / TB)
    } else if bytes_f >= GB {
        format!("{:.1} GB", bytes_f / GB)
    } else if bytes_f >= MB {
        format!("{:.1} MB", bytes_f / MB)
    } else if bytes_f >= KB {
        format!("{:.1} KB", bytes_f / KB)
    } else {
        format!("{} B", bytes)
    }
}
