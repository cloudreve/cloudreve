use super::DriveManager;
use crate::drive::commands::{ManagerCommand, MountCommand};
use crate::drive::utils::{local_path_to_cr_uri, view_online_url};
use crate::utils::toast::send_conflict_toast;
use anyhow::{Context, Result};
use std::path::PathBuf;
use std::sync::Arc;
use tokio::spawn;
use tokio::sync::mpsc;

impl DriveManager {
    /// Spawn the command processor task
    pub async fn spawn_command_processor(self: &Arc<Self>) {
        let mut command_rx_guard = self.command_rx.lock().await;
        if let Some(command_rx) = command_rx_guard.take() {
            let manager = self.clone();
            let handle = tokio::spawn(async move {
                Self::process_commands(manager, command_rx).await;
            });
            *self.processor_handle.lock().await = Some(handle);
        }
    }

    /// Process commands from external sources asynchronously
    pub(super) async fn process_commands(
        manager: Arc<Self>,
        mut command_rx: mpsc::UnboundedReceiver<ManagerCommand>,
    ) {
        tracing::info!(target: "drive::manager", "Command processor started");

        while let Some(command) = command_rx.recv().await {
            tracing::trace!(target: "drive::manager", command = ?command, "Processing command");
            let manager = manager.clone();
            match command {
                ManagerCommand::ViewOnline { path } => {
                    let path = path.clone();
                    spawn(async move {
                        let path = path.clone();
                        let result = manager.handle_view_online(path.clone()).await;
                        // TODO: handle result in UI
                        tracing::debug!(target: "drive::manager", path = %path.display(), result = ?result, "ViewOnline command result");
                    });
                }
                ManagerCommand::PersistConfig => {
                    let result = manager.persist().await;
                    if let Err(e) = result {
                        tracing::error!(target: "drive::manager", error = %e, "Failed to persist config");
                    }
                }
                ManagerCommand::SyncNow { paths, mode } => {
                    let paths = paths.clone();
                    if paths.len() < 1 {
                        tracing::error!(target: "drive::manager", "No paths provided for sync command");
                        return;
                    }
                    spawn(async move {
                        let drive = manager
                            .search_drive_by_child_path(
                                paths.get(0).unwrap().to_str().unwrap_or(""),
                            )
                            .await;
                        if let Some(drive) = drive {
                            let _ = drive.command_tx.send(MountCommand::Sync {
                                local_paths: paths,
                                mode: mode,
                                user_initiated: true,
                            });
                        } else {
                            tracing::error!(target: "drive::manager", "No drive found for path: {:?}", paths.get(0).unwrap());
                        }
                    });
                }
                ManagerCommand::GenerateThumbnail { path, response } => {
                    let path = path.clone();
                    spawn(async move {
                        let drive = manager
                            .search_drive_by_child_path(path.to_str().unwrap_or(""))
                            .await;
                        if let Some(drive) = drive {
                            let result = drive.generate_thumbnail(path.clone()).await;
                            if let Err(e) = result {
                                tracing::error!(target: "drive::manager", error = %e, "Failed to generate thumbnail");
                                let _ = response.send(Err(e));
                                return;
                            }

                            let _ = response.send(result);
                            return;
                        }

                        let _ = response
                            .send(Err(anyhow::anyhow!("No drive found for path: {:?}", path)));
                    });
                }
                ManagerCommand::ResolveConflict {
                    drive_id,
                    file_id,
                    path,
                    action,
                } => {
                    spawn(async move {
                        let drive = manager.get_drive(&drive_id).await;
                        if let Some(drive) = drive {
                            let result = drive.resolve_conflict(action, file_id, path).await;
                            if let Err(e) = result {
                                tracing::error!(target: "drive::manager", error = %e, "Failed to resolve conflict");
                            }
                        } else {
                            tracing::error!(target: "drive::manager", "No drive found for drive_id: {:?}", drive_id);
                        }
                    });
                }
                ManagerCommand::ShowConflictToast { path } => {
                    let path = path.clone();
                    spawn(async move {
                        let result = manager.handle_show_conflict_toast(path.clone()).await;
                        if let Err(e) = result {
                            tracing::error!(target: "drive::manager", path = %path.display(), error = %e, "Failed to show conflict toast");
                        }
                    });
                }
                ManagerCommand::GetDriveStatusUI { syncroot_id, response } => {
                    spawn(async move {
                        let result = manager.get_drive_status_by_syncroot_id(&syncroot_id).await;
                        let _ = response.send(result);
                    });
                }
                ManagerCommand::OpenProfileUrl { syncroot_id } => {
                    spawn(async move {
                        let result = manager.handle_open_profile_url(&syncroot_id).await;
                        if let Err(e) = result {
                            tracing::error!(target: "drive::manager", syncroot_id = %syncroot_id, error = %e, "Failed to open profile URL");
                        }
                    });
                }
                ManagerCommand::OpenStorageDetailsUrl { syncroot_id } => {
                    spawn(async move {
                        let result = manager.handle_open_storage_details_url(&syncroot_id).await;
                        if let Err(e) = result {
                            tracing::error!(target: "drive::manager", syncroot_id = %syncroot_id, error = %e, "Failed to open storage details URL");
                        }
                    });
                }
                ManagerCommand::OpenSyncStatusWindow => {
                    manager.event_broadcaster.open_sync_status_window();
                }
                ManagerCommand::OpenSettingsWindow => {
                    manager.event_broadcaster.open_settings_window();
                }
            }
        }

        tracing::info!(target: "drive::manager", "Command processor stopped");
    }

    /// Handle ViewOnline command
    pub(super) async fn handle_view_online(&self, path: PathBuf) -> Result<()> {
        tracing::debug!(target: "drive::manager", path = %path.display(), "ViewOnline command");

        // Find the drive that contains this path
        let mount = self
            .search_drive_by_child_path(path.to_str().unwrap_or(""))
            .await
            .ok_or_else(|| anyhow::anyhow!("No drive found for path: {:?}", path))?;

        let file_meta = self
            .inventory
            .query_by_path(path.to_str().unwrap_or(""))
            .context("Failed to query file metadata")?;

        let config = mount.get_config().await;
        let (sync_path, remote_path) =
            { (config.sync_path.clone(), config.remote_path.to_string()) };
        let uri = local_path_to_cr_uri(path.clone(), sync_path, remote_path)
            .context("failed to convert local path to cloudreve uri")?
            .to_string();

        // Determine which URL to open
        let url = match file_meta {
            // If no metadata, assume it's the sync root, open folder
            None => view_online_url(&config.remote_path, None, &config)?,
            Some(ref meta) if meta.is_folder => view_online_url(&uri, None, &config)?,
            Some(ref _meta) => {
                use cloudreve_api::models::uri::CrUri;
                let parent_path = CrUri::new(&uri)?.parent()?.to_string();
                view_online_url(&parent_path, Some(&uri), &config)?
            }
        };

        open::that(url)?;
        Ok(())
    }

    /// Handle ShowConflictToast command
    pub(super) async fn handle_show_conflict_toast(&self, path: PathBuf) -> Result<()> {
        tracing::debug!(target: "drive::manager", path = %path.display(), "ShowConflictToast command");

        // Find the drive that contains this path
        let mount = self
            .search_drive_by_child_path(path.to_str().unwrap_or(""))
            .await
            .ok_or_else(|| anyhow::anyhow!("No drive found for path: {:?}", path))?;

        // Query inventory for file metadata
        let file_meta = self
            .inventory
            .query_by_path(path.to_str().unwrap_or(""))
            .context("Failed to query file metadata")?
            .ok_or_else(|| anyhow::anyhow!("File not found in inventory: {:?}", path))?;

        let config = mount.get_config().await;

        // Send the conflict toast
        send_conflict_toast(&config.id, &path, file_meta.id);

        Ok(())
    }

    /// Handle OpenProfileUrl command - opens user profile page in browser
    pub(super) async fn handle_open_profile_url(&self, syncroot_id: &str) -> Result<()> {
        tracing::debug!(target: "drive::manager", syncroot_id = %syncroot_id, "OpenProfileUrl command");

        let status = self
            .get_drive_status_by_syncroot_id(syncroot_id)
            .await?
            .ok_or_else(|| anyhow::anyhow!("No drive found for syncroot_id: {}", syncroot_id))?;

        open::that(&status.profile_url)?;
        Ok(())
    }

    /// Handle OpenStorageDetailsUrl command - opens storage/capacity page in browser
    pub(super) async fn handle_open_storage_details_url(&self, syncroot_id: &str) -> Result<()> {
        tracing::debug!(target: "drive::manager", syncroot_id = %syncroot_id, "OpenStorageDetailsUrl command");

        let status = self
            .get_drive_status_by_syncroot_id(syncroot_id)
            .await?
            .ok_or_else(|| anyhow::anyhow!("No drive found for syncroot_id: {}", syncroot_id))?;

        // Open the profile URL which shows storage details
        open::that(&status.storage_url)?;
        Ok(())
    }
}
