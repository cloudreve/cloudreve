use std::{path::PathBuf, str::FromStr, sync::Arc, time::SystemTime};

use crate::utils::toast::send_conflict_toast;
use crate::{
    drive::{placeholder::CrPlaceholder, utils::local_path_to_cr_uri},
    inventory::{ConflictState, FileMetadata, InventoryDb},
    tasks::queue::QueuedTask,
    uploader::{ProgressCallback, ProgressUpdate, UploadParams, Uploader, UploaderConfig},
};
use anyhow::{Context, Result};
use bytes::Bytes;
use cloudreve_api::{
    ApiError, Client,
    api::ExplorerApi,
    error::ErrorCode,
    models::explorer::{CreateFileService, FileResponse, FileUpdateService, file_type},
};
use dashmap::DashMap;
use tokio_util::sync::CancellationToken;
use tracing::{debug, info, warn};
use uuid::Uuid;

use super::types::TaskProgress;

/// Progress reporter that updates task progress in-memory via a DashMap reference.
/// Does NOT persist to inventory - only keeps in-memory for real-time queries.
pub struct InMemoryProgressReporter {
    task_id: String,
    progress_map: Arc<DashMap<String, TaskProgress>>,
}

impl InMemoryProgressReporter {
    pub fn new(task_id: String, progress_map: Arc<DashMap<String, TaskProgress>>) -> Self {
        Self {
            task_id,
            progress_map,
        }
    }
}

impl ProgressCallback for InMemoryProgressReporter {
    fn on_progress(&self, update: ProgressUpdate) {
        if let Some(mut entry) = self.progress_map.get_mut(&self.task_id) {
            entry.update_from_progress(&update);
        }
    }
}

pub struct UploadTask<'a> {
    inventory: Arc<InventoryDb>,
    cr_client: Arc<Client>,
    drive_id: &'a str,
    sync_path: PathBuf,
    remote_base: String,
    task: &'a QueuedTask,
    local_file: Option<CrPlaceholder>,
    inventory_meta: Option<FileMetadata>,
    cancel_token: CancellationToken,
    /// Reference to the in-memory progress map for real-time progress updates
    progress_map: Arc<DashMap<String, TaskProgress>>,
}

impl<'a> UploadTask<'a> {
    pub fn new(
        inventory: Arc<InventoryDb>,
        cr_client: Arc<Client>,
        drive_id: &'a str,
        task: &'a QueuedTask,
        sync_path: PathBuf,
        remote_base: String,
        progress_map: Arc<DashMap<String, TaskProgress>>,
    ) -> Self {
        Self {
            inventory,
            cr_client,
            drive_id,
            local_file: None,
            inventory_meta: None,
            task,
            sync_path,
            remote_base,
            cancel_token: CancellationToken::new(),
            progress_map,
        }
    }

    /// Set the cancellation token
    #[allow(dead_code)]
    pub fn with_cancel_token(mut self, token: CancellationToken) -> Self {
        self.cancel_token = token;
        self
    }

    // Upload a local file/folder to cloud
    pub async fn execute(&mut self) -> Result<()> {
        // Get local file info
        let placeholder_file = CrPlaceholder::new(
            &self.task.payload.local_path,
            self.sync_path.clone(),
            Uuid::from_str(self.drive_id)?,
        );
        if !placeholder_file.local_file_info.exists {
            info!(
                target: "tasks::upload",
                task_id = %self.task.task_id,
                local_path = %self.task.payload.local_path_display(),
                "Local file does not exist, skipping upload"
            );
            return Ok(());
        }

        if placeholder_file.local_file_info.in_sync()
            && !placeholder_file.local_file_info.is_directory()
        {
            info!(
                target: "tasks::upload",
                task_id = %self.task.task_id,
                local_path = %self.task.payload.local_path_display(),
                "Local file is in sync, skipping upload"
            );
            return Ok(());
        }

        let is_directory = placeholder_file.local_file_info.is_directory;
        let file_size = placeholder_file.local_file_info.file_size.unwrap_or(0);
        self.local_file = Some(placeholder_file);

        // Get inventory meta
        let path_str = self
            .task
            .payload
            .local_path
            .to_str()
            .context("failed to get local path as str")?;
        self.inventory_meta = self
            .inventory
            .query_by_path(path_str)
            .context("failed to get inventory meta")?;

        // clear file error state
        // Mark file as error state
        if let Err(e) = self
            .local_file
            .as_mut()
            .unwrap()
            .update_sync_error_state(false)
        {
            warn!(target: "tasks::upload", task_id = %self.task.task_id, local_path = %self.task.payload.local_path_display(), error = ?e, "Failed to clear sync error state");
        }

        // Handle empty files and directories separately
        let upload_res = match (
            is_directory,
            file_size == 0 && !self.task.payload.force_override,
            self.inventory_meta.is_none(),
        ) {
            (true, _, _) => self.create_empty_file_or_folder().await,
            (false, true, true) => self.create_empty_file_or_folder().await,
            (false, true, false) => self.clear_file_content().await,
            (false, false, _) => self.upload_file_with_uploader().await,
        };

        self.handle_error(upload_res).await
    }

    async fn handle_error(&mut self, r: Result<()>) -> Result<()> {
        match r {
            Ok(()) => Ok(()),
            Err(e) => {
                // Check if the error is an ApiError with StaleVersion (40076)
                // The error might be wrapped with anyhow context, so check the chain
                let is_conflict_error = e.chain().any(|cause| {
                    if let Some(api_err) = cause.downcast_ref::<ApiError>() {
                        matches!(
                            api_err,
                            ApiError::ApiError { code, .. }
                                if *code == ErrorCode::StaleVersion as i32
                                || *code == ErrorCode::ObjectExisted as i32
                        )
                    } else {
                        false
                    }
                });

                if is_conflict_error {
                    warn!(
                        target: "tasks::upload",
                        task_id = %self.task.task_id,
                        local_path = %self.task.payload.local_path_display(),
                        "Conflict detected, server has newer version or object exists"
                    );

                    // Mark the file as conflicted in the inventory
                    let path_str = self.task.payload.local_path.to_str().unwrap_or_default();
                    if let Err(mark_err) = self
                        .inventory
                        .mark_as_conflicted(path_str, Some(ConflictState::Pending))
                    {
                        warn!(
                            target: "tasks::upload",
                            task_id = %self.task.task_id,
                            local_path = %self.task.payload.local_path_display(),
                            error = ?mark_err,
                            "Failed to mark file as conflicted"
                        );
                    }

                    // Send conflict toast
                    send_conflict_toast(
                        self.drive_id,
                        &self.task.payload.local_path,
                        self.inventory_meta
                            .as_ref()
                            .map(|meta| meta.id)
                            .unwrap_or(0),
                    )
                }

                // Mark file as error state
                if let Err(e) = self
                    .local_file
                    .as_mut()
                    .unwrap()
                    .update_sync_error_state(true)
                {
                    warn!(target: "tasks::upload", task_id = %self.task.task_id, local_path = %self.task.payload.local_path_display(), error = ?e, "Failed to update sync error state");
                }
                Err(e)
            }
        }
    }

    async fn clear_file_content(&mut self) -> Result<()> {
        info!(
            target: "tasks::upload",
            task_id = %self.task.task_id,
            local_path = %self.task.payload.local_path_display(),
            "Clearing file content with update request"
        );

        let uri = local_path_to_cr_uri(
            self.task.payload.local_path.clone(),
            self.sync_path.clone(),
            self.remote_base.clone(),
        )
        .context("failed to convert local path to cloudreve uri")?
        .to_string();
        let etag = self.inventory_meta.as_ref().unwrap().etag.clone();
        let res = self
            .cr_client
            .update_file(
                &FileUpdateService {
                    uri,
                    previous: Some(etag),
                },
                Bytes::new(),
            )
            .await;

        match res {
            Ok(file) => self.file_uploaded(&file),
            Err(e) => Err(e.into()),
        }
    }

    /// Upload a file using the new uploader module
    async fn upload_file_with_uploader(&mut self) -> Result<()> {
        let local_file = self.local_file.as_ref().unwrap();
        let file_size = local_file.local_file_info.file_size.unwrap_or(0);
        let is_new_file = self.inventory_meta.is_none();

        info!(
            target: "tasks::upload",
            task_id = %self.task.task_id,
            local_path = %self.task.payload.local_path_display(),
            file_size = file_size,
            "Starting file upload"
        );

        // Get remote URI
        let uri = local_path_to_cr_uri(
            self.task.payload.local_path.clone(),
            self.sync_path.clone(),
            self.remote_base.clone(),
        )
        .context("failed to convert local path to cloudreve uri")?
        .to_string();

        // If conflict state is set to Override, omit previous_version to force upload without version check
        let previous_version = if let Some(meta) = &self.inventory_meta {
            if matches!(meta.conflict_state, Some(ConflictState::Override)) {
                String::new() // Omit previous version when user chose to override
            } else {
                meta.etag.clone()
            }
        } else {
            String::new()
        };

        let params = UploadParams {
            local_path: self.task.payload.local_path.clone(),
            remote_uri: uri,
            file_size,
            mime_type: None, // Could be detected from file extension
            last_modified: local_file.local_file_info.last_modified.map(|t| {
                t.duration_since(SystemTime::UNIX_EPOCH)
                    .unwrap()
                    .as_millis() as i64
            }),
            overwrite: !is_new_file || self.task.payload.force_override,
            previous_version,
            task_id: self.task.task_id.clone(),
            drive_id: self.drive_id.to_string(),
        };

        // Create uploader configuration
        let config = UploaderConfig::default();

        // Create uploader
        let uploader = Uploader::new(self.cr_client.clone(), self.inventory.clone(), config)
            .with_cancel_token(self.cancel_token.clone());

        // Create in-memory progress reporter (does not persist to inventory)
        let progress = InMemoryProgressReporter::new(
            self.task.task_id.clone(),
            Arc::clone(&self.progress_map),
        );

        // Execute upload
        uploader
            .upload(params, progress)
            .await
            .context("failed to upload file")?;

        // Update local file placeholder status after successful upload
        self.finalize_upload().await?;

        Ok(())
    }

    /// Finalize upload by updating local file placeholder
    async fn finalize_upload(&mut self) -> Result<()> {
        // Get file info from server to confirm upload
        let uri = local_path_to_cr_uri(
            self.task.payload.local_path.clone(),
            self.sync_path.clone(),
            self.remote_base.clone(),
        )
        .context("failed to convert local path to cloudreve uri")?
        .to_string();

        let file_info = self
            .cr_client
            .get_file_info(&cloudreve_api::models::explorer::GetFileInfoService {
                uri: Some(uri),
                id: None,
                extended: None,
                folder_summary: None,
            })
            .await
            .context("failed to get file info after upload")?;

        self.file_uploaded(&file_info)
            .context("failed to commit uploaded file")?;
        Ok(())
    }

    async fn create_empty_file_or_folder(&mut self) -> Result<()> {
        info!(
            target: "tasks::upload",
            task_id = %self.task.task_id,
            local_path = %self.task.payload.local_path_display(),
            "Creating empty file/folder"
        );
        let local_file = &self.local_file.as_ref().unwrap().local_file_info;
        let uri = local_path_to_cr_uri(
            self.task.payload.local_path.clone(),
            self.sync_path.clone(),
            self.remote_base.clone(),
        )
        .context("failed to convert local path to cloudreve uri")?
        .to_string();

        debug!(target: "tasks::upload", task_id = %self.task.task_id, local_path = %self.task.payload.local_path_display(), uri = %uri, "Send test toast");

        // Create file in remote
        let res = self
            .cr_client
            .create_file(&CreateFileService {
                uri,
                file_type: if local_file.is_directory {
                    "folder".to_string()
                } else {
                    "file".to_string()
                },
                err_on_conflict: Some(!local_file.is_directory),
                metadata: None,
            })
            .await;
        match res {
            Ok(folder) => self.file_uploaded(&folder),
            Err(e) => Err(e.into()),
        }
    }

    fn file_uploaded(&mut self, file: &FileResponse) -> Result<()> {
        info!(
            target: "tasks::upload",
            task_id = %self.task.task_id,
            local_path = %self.task.payload.local_path_display(),
            "File uploaded"
        );

        self.local_file = Some(
            self.local_file
                .take()
                .unwrap()
                .with_mark_no_children(file.file_type == file_type::FOLDER)
                .with_remote_file(file),
        );

        self.local_file
            .as_mut()
            .unwrap()
            .commit(self.inventory.clone())
            .context("failed to commit placeholder")?;

        self.local_file
            .as_mut()
            .unwrap()
            .update_sync_error_state(false)
            .context("failed to clear sync error state")?;
        Ok(())
    }
}
