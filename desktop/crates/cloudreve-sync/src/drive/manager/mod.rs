mod command_handlers;
pub(crate) mod favicon;
mod types;

pub use types::*;

use crate::drive::commands::ManagerCommand;
use crate::drive::mounts::{Credentials, DriveConfig, Mount};
use crate::EventBroadcaster;
use crate::inventory::InventoryDb;
use crate::tasks::TaskProgress;
use anyhow::{Context, Result};
use std::collections::HashMap;
use std::path::PathBuf;
use std::sync::Arc;
use std::time::Duration;
use std::{fs, thread};
use tokio::sync::{Mutex, RwLock, mpsc};

pub struct DriveManager {
    pub(super) drives: Arc<RwLock<HashMap<String, Arc<Mount>>>>,
    config_dir: PathBuf,
    pub(super) inventory: Arc<InventoryDb>,
    pub(super) command_tx: mpsc::UnboundedSender<ManagerCommand>,
    pub(super) command_rx: Arc<Mutex<Option<mpsc::UnboundedReceiver<ManagerCommand>>>>,
    pub(super) processor_handle: Arc<Mutex<Option<tokio::task::JoinHandle<()>>>>,
    pub(super) event_broadcaster: Arc<EventBroadcaster>,
}

impl DriveManager {
    /// Create a new DriveManager instance
    pub fn new(event_broadcaster: Arc<EventBroadcaster>) -> Result<Self> {
        let config_dir = Self::get_config_dir()?;

        // Ensure config directory exists
        if !config_dir.exists() {
            fs::create_dir_all(&config_dir)
                .context("Failed to create .cloudreve config directory")?;
        }

        let (command_tx, command_rx) = mpsc::unbounded_channel();

        Ok(Self {
            config_dir,
            drives: Arc::new(RwLock::new(HashMap::new())),
            inventory: Arc::new(InventoryDb::new().context("Failed to create inventory database")?),
            command_tx,
            command_rx: Arc::new(Mutex::new(Some(command_rx))),
            processor_handle: Arc::new(Mutex::new(None)),
            event_broadcaster: event_broadcaster,
        })
    }

    pub fn get_inventory(&self) -> Arc<InventoryDb> {
        self.inventory.clone()
    }

    /// Get the .cloudreve config directory path
    fn get_config_dir() -> Result<PathBuf> {
        let home_dir = dirs::home_dir().context("Failed to get user home directory")?;
        Ok(home_dir.join(".cloudreve"))
    }

    /// Get the config file path
    fn get_config_file(&self) -> PathBuf {
        self.config_dir.join("drives.json")
    }

    /// Load drive configurations from disk
    pub async fn load(&self) -> Result<()> {
        let config_file = self.get_config_file();

        if !config_file.exists() {
            tracing::info!(target: "drive", "No existing drive config found, starting fresh");
            self.event_broadcaster.no_drive();
            return Ok(());
        }

        tracing::debug!(target: "drive", path = %config_file.display(), "Loading drive configurations");

        let content =
            fs::read_to_string(&config_file).context("Failed to read drive config file")?;

        let state: DriveState =
            serde_json::from_str(&content).context("Failed to parse drive config")?;

        // Add drives to manager
        let mut count = 0;
        for config in state.drives.iter() {
            match self.add_drive(config.clone()).await {
                Ok(_) => {
                    count += 1;
                }
                Err(e) => {
                    tracing::error!(target: "drive", drive_id = %config.id, error = ?e, "Failed to add drive, skipping");
                    // crate::utils::toast::send_warning_toast(
                    //     &t!("driveLoadFailed"),
                    //     &format!("{}: {}", config.name, e),
                    // );
                }
            }
        }

        if count == 0 {
            self.event_broadcaster.no_drive();
        }

        tracing::info!(target: "drive", count = count, "Loaded drive(s) from config");

        Ok(())
    }

    /// Persist drive configurations to disk
    pub async fn persist(&self) -> Result<()> {
        let config_file = self.get_config_file();
        let write_guard = self.drives.write().await;

        tracing::debug!(target: "drive", path = %config_file.display(), count = write_guard.len(), "Persisting drive configurations");

        let mut new_state = DriveState::default();

        // Update drive states from underlying mounts
        for (_, mount) in write_guard.iter() {
            let config = mount.get_config().await;
            new_state.drives.push(config);
        }

        let content =
            serde_json::to_string_pretty(&new_state).context("Failed to serialize drive state")?;
        fs::write(&config_file, content).context("Failed to write drive config file")?;

        tracing::info!(target: "drive", count = new_state.drives.len(), "Persisted drive(s) to config");

        Ok(())
    }

    /// Register a callback to be invoked when status UI changes
    /// This is a dummy implementation that calls the callback every 30 seconds
    pub fn register_on_status_ui_changed<F>(&self, fnc: F) -> Result<()>
    where
        F: Fn() + Send + 'static,
    {
        thread::spawn(move || {
            loop {
                thread::sleep(Duration::from_secs(30));
                tracing::trace!(target: "drive::manager", "Register_on_status_ui_changed: Invoking status UI changed callback");
                fnc();
            }
        });
        Ok(())
    }

    /// Add a new drive
    pub async fn add_drive(&self, mut config: DriveConfig) -> Result<String> {
        // Fetch favicon if icon_path is not set or doesn't exist
        if config.icon_path.is_none()
            || !config
                .icon_path
                .as_ref()
                .map(|p| std::path::Path::new(p).exists())
                .unwrap_or(false)
        {
            match favicon::fetch_and_save_favicon(&config.instance_url).await {
                Ok(result) => {
                    tracing::info!(target: "drive", ico_path = %result.ico_path, raw_path = %result.raw_path, "Favicon fetched successfully");
                    config.icon_path = Some(result.ico_path);
                    config.raw_icon_path = Some(result.raw_path);
                }
                Err(e) => {
                    tracing::warn!(target: "drive", error = %e, "Failed to fetch favicon, continuing without icon");
                }
            }
        }

        let mut write_guard = self.drives.write().await;
        let mut mount = Mount::new(
            config.clone(),
            self.inventory.clone(),
            self.command_tx.clone(),
        )
        .await;
        if let Err(e) = mount.start().await {
            tracing::error!(target: "drive", error = ?e, "Failed to start drive");
            return Err(e).context("Failed to start drive");
        }

        let mount_arc = Arc::new(mount);
        mount_arc.spawn_command_processor(mount_arc.clone()).await;
        mount_arc
            .spawn_remote_event_processor(mount_arc.clone())
            .await;
        mount_arc.spawn_props_refresh_task().await;
        let id = mount_arc.id.clone();
        write_guard.insert(id.clone(), mount_arc);
        Ok(id)
    }

    // Search drive by child file path.
    // Child path can be up to the sync root path.
    pub async fn search_drive_by_child_path(&self, path: &str) -> Option<Arc<Mount>> {
        let read_guard = self.drives.read().await;

        // Convert the input path to an absolute PathBuf for comparison
        let target_path = PathBuf::from(path);
        let target_path = match target_path.canonicalize() {
            Ok(p) => p,
            Err(_) => {
                // If canonicalize fails (e.g., path doesn't exist), try to work with the original path
                target_path
            }
        };

        // Iterate through all drives and check if the target path is under their sync root
        for (_, mount) in read_guard.iter() {
            let sync_path = mount.get_sync_path().await;

            // Normalize the sync path
            let sync_path = match sync_path.canonicalize() {
                Ok(p) => p,
                Err(_) => sync_path,
            };

            // Check if target_path starts with sync_path (is a child of sync_path)
            if target_path.starts_with(&sync_path) {
                return Some(mount.clone());
            }
        }

        None
    }

    /// Remove a drive by ID
    ///
    /// This will:
    /// 1. Stop and delete the mount (unregister sync root, cleanup inventory)
    /// 2. Remove the drive from the manager's drive map
    ///
    /// Note: The caller is responsible for calling `persist()` after this to save the config.
    pub async fn remove_drive(&self, id: &str) -> Result<Option<DriveConfig>> {
        let mut write_guard = self.drives.write().await;

        // Remove the mount from the map
        let mount = match write_guard.remove(id) {
            Some(m) => m,
            None => return Ok(None),
        };

        // Get the config before deleting the mount
        let config = mount.get_config().await;

        // Drop the write guard before calling delete to avoid potential deadlocks
        drop(write_guard);

        // Delete the mount (unregister sync root, cleanup, etc.)
        mount.delete().await.context("Failed to delete mount")?;

        // Broadcast no_drive event if no drives remain
        if self.drives.read().await.is_empty() {
            self.event_broadcaster.no_drive();
        }

        tracing::info!(target: "drive::manager", drive_id = %id, "Drive removed successfully");

        Ok(Some(config))
    }

    /// Get a drive by ID
    pub async fn get_drive(&self, id: &str) -> Option<Arc<Mount>> {
        let read_guard = self.drives.read().await;
        read_guard.get(id).cloned()
    }

    /// List all drives
    pub async fn list_drives(&self) -> Vec<DriveConfig> {
        // let read_guard = self.drives.read().await;
        // read_guard
        //     .values()
        //     .map(|mount| mount.get_config())
        //     .collect()
        Vec::new()
    }

    /// Update drive configuration
    pub async fn update_drive(&self, _id: &str, _config: DriveConfig) -> Result<()> {
        // let mut write_guard = self.drives.write().await;
        // if write_guard.contains_key(id) {
        //     // write_guard.insert(id.to_string(), Mount::new(config.clone()));
        //     Ok(())
        // } else {
        //     anyhow::bail!("Drive not found: {}", id)
        // }
        Err(anyhow::anyhow!("Not implemented"))
    }

    /// Update drive credentials for reauthorization.
    ///
    /// This updates the name, instance_url, and credentials for an existing drive.
    /// It also clears and re-fetches the site icon.
    ///
    /// # Arguments
    /// * `id` - The drive ID to update
    /// * `name` - New drive name
    /// * `instance_url` - New instance URL
    /// * `credentials` - New credentials
    /// * `user_id` - The user ID from the new authorization (must match original)
    ///
    /// # Errors
    /// Returns an error if:
    /// - Drive is not found
    /// - The user_id doesn't match the original drive's user_id
    pub async fn update_drive_credentials(
        &self,
        id: &str,
        name: String,
        instance_url: String,
        credentials: Credentials,
        user_id: &str,
    ) -> Result<()> {
        let read_guard = self.drives.read().await;
        let mount = read_guard
            .get(id)
            .ok_or_else(|| anyhow::anyhow!("Drive not found: {}", id))?;

        // Check if user_id matches
        {
            let config = mount.config.read().await;
            if config.user_id != user_id {
                return Err(anyhow::anyhow!(t!("userIdMismatch")));
            }
        }

        // Update the config
        let mut config = mount.config.write().await;

        // Clear old icon files if they exist
        if let Some(ref ico_path) = config.icon_path {
            if std::path::Path::new(ico_path).exists() {
                if let Err(e) = std::fs::remove_file(ico_path) {
                    tracing::warn!(target: "drive::manager", drive_id = %id, error = %e, "Failed to remove old ICO file");
                }
            }
        }
        if let Some(ref raw_path) = config.raw_icon_path {
            if std::path::Path::new(raw_path).exists() {
                if let Err(e) = std::fs::remove_file(raw_path) {
                    tracing::warn!(target: "drive::manager", drive_id = %id, error = %e, "Failed to remove old raw icon file");
                }
            }
        }

        // Update fields
        config.name = name;
        config.instance_url = instance_url.clone();
        config.credentials = credentials.clone();

        // Clear icon paths - will be re-fetched
        config.icon_path = None;
        config.raw_icon_path = None;

        // Fetch new favicon
        match favicon::fetch_and_save_favicon(&instance_url).await {
            Ok(result) => {
                tracing::info!(target: "drive::manager", drive_id = %id, ico_path = %result.ico_path, raw_path = %result.raw_path, "Favicon re-fetched successfully");
                config.icon_path = Some(result.ico_path);
                config.raw_icon_path = Some(result.raw_path);
            }
            Err(e) => {
                tracing::warn!(target: "drive::manager", drive_id = %id, error = %e, "Failed to re-fetch favicon, continuing without icon");
            }
        }

        drop(config);

        // Update the client's tokens
        mount
            .cr_client
            .set_tokens_with_expiry(&cloudreve_api::models::user::Token {
                access_token: credentials.access_token.clone().unwrap_or_default(),
                refresh_token: credentials.refresh_token.clone(),
                access_expires: credentials.access_expires.clone().unwrap_or_default(),
                refresh_expires: credentials.refresh_expires.clone(),
            })
            .await?;

        // Clear the credential expired flag since we got new credentials
        mount.set_credential_expired(false).await;

        tracing::info!(target: "drive::manager", drive_id = %id, "Drive credentials updated successfully");

        Ok(())
    }

    /// Get the ignore patterns for a drive
    pub async fn get_ignore_patterns(&self, id: &str) -> Result<Vec<String>> {
        let read_guard = self.drives.read().await;
        let mount = read_guard
            .get(id)
            .ok_or_else(|| anyhow::anyhow!("Drive not found: {}", id))?;
        let config = mount.config.read().await;
        Ok(config.ignore_patterns.clone())
    }

    /// Update the ignore patterns for a drive.
    ///
    /// Validates patterns, updates the config, and rebuilds the `IgnoreMatcher`.
    pub async fn update_ignore_patterns(&self, id: &str, patterns: Vec<String>) -> Result<()> {
        let read_guard = self.drives.read().await;
        let mount = read_guard
            .get(id)
            .ok_or_else(|| anyhow::anyhow!("Drive not found: {}", id))?;
        mount.update_ignore_patterns(patterns).await
    }

    /// Enable/disable a drive
    pub async fn set_drive_enabled(&self, _id: &str, _enabled: bool) -> Result<()> {
        Err(anyhow::anyhow!("Not implemented"))
    }

    /// Placeholder: Start syncing a drive
    pub async fn start_sync(&self, _id: &str) -> Result<()> {
        Err(anyhow::anyhow!("Not implemented"))
    }

    /// Placeholder: Stop syncing a drive
    pub async fn stop_sync(&self, _id: &str) -> Result<()> {
        Err(anyhow::anyhow!("Not implemented"))
    }

    /// Placeholder: Get sync status for a drive
    pub async fn get_sync_status(&self, id: &str) -> Result<serde_json::Value> {
        // TODO: Implement actual status retrieval
        tracing::debug!(target: "drive::sync", drive_id = %id, "Getting sync status");
        Ok(serde_json::json!({
            "drive_id": id,
            "status": "idle",
            "last_sync": null,
            "files_synced": 0,
        }))
    }

    /// Get a summary of the current status including all drives and recent tasks.
    ///
    /// # Arguments
    /// * `drive_id` - Optional drive ID to filter tasks. If None, returns tasks from all drives.
    ///                Note: drives list always returns all drives regardless of this filter.
    pub async fn get_status_summary(&self, drive_id: Option<&str>) -> Result<StatusSummary> {
        // Get all drive configs (unfiltered)
        let read_guard = self.drives.read().await;
        let mut drives = Vec::with_capacity(read_guard.len());
        for mount in read_guard.values() {
            drives.push(mount.get_config().await);
        }

        // Query recent tasks from inventory (filtered by drive_id if provided)
        let recent_tasks = self
            .inventory
            .query_recent_tasks(drive_id)
            .context("Failed to query recent tasks")?;

        // Collect running task progress from all task queues
        // Build a map of task_id -> TaskProgress for quick lookup
        let mut progress_map: HashMap<String, TaskProgress> = HashMap::new();

        if let Some(drive_filter) = drive_id {
            // If filtering by drive, only get progress from that drive's task queue
            if let Some(mount) = read_guard.get(drive_filter) {
                for progress in mount.task_queue.ongoing_progress().await {
                    progress_map.insert(progress.task_id.clone(), progress);
                }
            }
        } else {
            // Get progress from all drives
            for mount in read_guard.values() {
                for progress in mount.task_queue.ongoing_progress().await {
                    progress_map.insert(progress.task_id.clone(), progress);
                }
            }
        }

        // Merge progress info into active tasks
        let active_tasks: Vec<TaskWithProgress> = recent_tasks
            .active
            .into_iter()
            .map(|task| {
                let progress = progress_map.remove(&task.id);
                TaskWithProgress { task, live_progress: progress }
            })
            .collect();

        Ok(StatusSummary {
            drives,
            active_tasks,
            finished_tasks: recent_tasks.finished,
        })
    }

    /// Get drive status by sync root ID (CFAPI ID) for the Windows Shell Status UI.
    ///
    /// # Arguments
    /// * `syncroot_id` - The sync root ID string (e.g., "cloudreve<hash>!S-1-5-21-xxx!user_id")
    ///
    /// # Returns
    /// * `Ok(Some(DriveStatusUI))` - Drive status if found
    /// * `Ok(None)` - No drive found with the given sync root ID
    /// * `Err` - An error occurred
    pub async fn get_drive_status_by_syncroot_id(
        &self,
        syncroot_id: &str,
    ) -> Result<Option<DriveStatusUI>> {
        let read_guard = self.drives.read().await;

        // Find the drive with matching sync root ID
        let mut found_mount: Option<&Arc<Mount>> = None;
        for mount in read_guard.values() {
            let config = mount.config.read().await;
            if let Some(ref sync_root) = config.sync_root_id {
                let sync_root_str = sync_root.to_os_string().to_string_lossy().to_string();
                if sync_root_str == syncroot_id {
                    drop(config);
                    found_mount = Some(mount);
                    break;
                }
            }
        }

        let mount = match found_mount {
            Some(m) => m,
            None => {
                tracing::debug!(target: "drive::manager", syncroot_id = %syncroot_id, "No drive found for sync root ID");
                return Ok(None);
            }
        };

        let config = mount.get_config().await;
        let drive_id = &config.id;

        let capacity = Self::get_capacity_summary(mount, drive_id, &config.remote_path);

        // Build profile URL: siteURL/profile/<user_id>?user_hint=<user_id>
        let profile_url = format!(
            "{}/profile/{}?user_hint={}",
            config.instance_url.trim_end_matches('/'),
            config.user_id,
            config.user_id
        );

        // Build settings URL: siteURL/settings?user_hint=<user_id>
        let settings_url = format!(
            "{}/settings?user_hint={}",
            config.instance_url.trim_end_matches('/'),
            config.user_id
        );

        let storage_url = format!(
            "{}/settings?tab=storage&user_hint={}",
            config.instance_url.trim_end_matches('/'),
            config.user_id
        );

        // Determine sync status based on active tasks
        let active_task_count = self.get_active_task_count(drive_id);

        let sync_status = if active_task_count > 0 {
            SyncStatus::Syncing
        } else {
            SyncStatus::InSync
        };

        Ok(Some(DriveStatusUI {
            name: config.name.clone(),
            raw_icon_path: config.raw_icon_path.clone(),
            capacity,
            profile_url,
            settings_url,
            storage_url,
            sync_status,
            active_task_count,
        }))
    }

    /// Get all drives with their status information for the settings UI.
    pub async fn get_drives_info(&self) -> Result<Vec<DriveInfo>> {
        let read_guard = self.drives.read().await;
        let mut drives_info = Vec::with_capacity(read_guard.len());

        for mount in read_guard.values() {
            let config = mount.get_config().await;
            let drive_id = &config.id;

            let capacity = Self::get_capacity_summary(mount, drive_id, &config.remote_path);

            let drive_state = mount.get_status_flags().await;

            // Determine drive status
            let status = if drive_state.is_credential_expired() {
                DriveInfoStatus::CredentialExpired
            } else {
                if !drive_state.is_event_push_subscribed(){
                    DriveInfoStatus::EventPushLost
                } else {
                    DriveInfoStatus::Active
                }
            };

            drives_info.push(DriveInfo {
                id: config.id.clone(),
                name: config.name.clone(),
                instance_url: config.instance_url.clone(),
                sync_path: config.sync_path.to_string_lossy().to_string(),
                icon_path: config.icon_path.clone(),
                remote_path: config.remote_path.clone(),
                raw_icon_path: config.raw_icon_path.clone(),
                enabled: config.enabled,
                user_id: config.user_id.clone(),
                status,
                capacity,
            });
        }

        Ok(drives_info)
    }

    /// Get a command sender for external code to send commands to the manager
    pub fn get_command_sender(&self) -> mpsc::UnboundedSender<ManagerCommand> {
        self.command_tx.clone()
    }

    pub async fn shutdown(&self) {
        tracing::info!(target: "drive::manager", "Shutting down DriveManager");

        // Close the command channel to signal the processor task to stop
        drop(self.command_tx.clone());

        // Wait for the processor task to finish
        if let Some(handle) = self.processor_handle.lock().await.take() {
            tracing::debug!(target: "drive::manager", "Waiting for command processor to finish");
            handle.abort();
        }

        let write_guard = self.drives.write().await;
        for (_, mount) in write_guard.iter() {
            mount.shutdown().await;
        }
        tracing::info!(target: "drive", "All drives shutdown");
    }
}

impl DriveManager {
    /// Get capacity summary from a mount's drive props.
    /// Only returns capacity if the remote_path filesystem is "my".
    fn get_capacity_summary(mount: &Mount, drive_id: &str, remote_path: &str) -> Option<CapacitySummary> {
        // Only show capacity for "my" filesystem
        use cloudreve_api::models::uri::CrUri;
        let is_my_fs = CrUri::new(remote_path)
            .map(|uri| uri.fs() == "my")
            .unwrap_or(false);

        if !is_my_fs {
            return None;
        }

        match mount.get_drive_props() {
            Ok(Some(props)) => props.capacity.map(|cap| {
                let percentage = if cap.total > 0 {
                    (cap.used as f64 / cap.total as f64) * 100.0
                } else {
                    0.0
                };
                CapacitySummary {
                    total: cap.total,
                    used: cap.used,
                    label: format!(
                        "{} / {} ({:.1}%)",
                        format_bytes(cap.used),
                        format_bytes(cap.total),
                        percentage
                    ),
                }
            }),
            Ok(None) => None,
            Err(e) => {
                tracing::warn!(target: "drive::manager", drive_id = %drive_id, error = %e, "Failed to get drive props");
                None
            }
        }
    }

    /// Get the count of active tasks for a drive
    fn get_active_task_count(&self, drive_id: &str) -> usize {
        match self.inventory.query_recent_tasks(Some(drive_id)) {
            Ok(tasks) => tasks.active.len(),
            Err(e) => {
                tracing::warn!(target: "drive::manager", drive_id = %drive_id, error = %e, "Failed to query recent tasks");
                0
            }
        }
    }
}
