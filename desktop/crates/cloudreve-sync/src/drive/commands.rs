use crate::{
    cfapi::{
        filter::ticket,
        placeholder::{LocalFileInfo, OpenOptions, PinState},
        utility::WriteAt,
    },
    drive::{
        mounts::Mount,
        placeholder::CrPlaceholder,
        sync::{GroupedFsEvents, SyncMode},
        utils::{local_path_to_cr_uri, notify_shell_change},
    },
    inventory::ConflictState,
    tasks::TaskPayload,
    utils::toast,
};
use anyhow::{Context, Result};
use bytes::Bytes;
use cloudreve_api::{
    ApiError,
    api::{ExplorerApi, explorer::ExplorerApiExt},
    models::{
        explorer::{
            DeleteFileService, FileResponse, FileURLService, MoveFileService, RenameFileService,
            metadata,
        },
        uri::CrUri,
        user::Token,
    },
};
use notify_debouncer_full::notify::{
    Event, EventKind,
    event::{CreateKind, ModifyKind, RemoveKind, RenameMode},
};
use std::{
    collections::HashMap,
    ops::Range,
    path::{Path, PathBuf},
};
use tokio::sync::oneshot::Sender;
use uuid::Uuid;
use windows::Win32::UI::Shell::SHCNE_ATTRIBUTES;
const PAGE_SIZE: i32 = 1000;

/// Generate a unique filename by appending a counter suffix before the extension.
/// For example: "document.txt" -> "document (1).txt", "document (2).txt", etc.
/// For files without extension: "README" -> "README (1)", "README (2)", etc.
fn generate_unique_filename(original_path: &Path) -> PathBuf {
    let parent = original_path.parent().unwrap_or(Path::new(""));
    let stem = original_path
        .file_stem()
        .and_then(|s| s.to_str())
        .unwrap_or("file");
    let extension = original_path.extension().and_then(|e| e.to_str());

    let mut counter = 1;
    loop {
        let new_name = match extension {
            Some(ext) => format!("{} ({}).{}", stem, counter, ext),
            None => format!("{} ({})", stem, counter),
        };
        let new_path = parent.join(&new_name);
        if !new_path.exists() {
            return new_path;
        }
        counter += 1;
        // Safety limit to prevent infinite loop
        if counter > 10000 {
            // Fallback to timestamp-based name
            let timestamp = std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .map(|d| d.as_millis())
                .unwrap_or(0);
            let fallback_name = match extension {
                Some(ext) => format!("{}_{}.{}", stem, timestamp, ext),
                None => format!("{}_{}", stem, timestamp),
            };
            return parent.join(fallback_name);
        }
    }
}

#[derive(Debug, Clone)]
pub struct GetPlacehodlerResult {
    pub files: Vec<FileResponse>,
    pub local_path: PathBuf,
    pub remote_path: CrUri,
}

/// Messages sent from OS threads (SyncFilter callbacks) to the async processing task
///
/// # Safety
/// This is safe because Windows CFAPI callbacks are designed to be invoked from arbitrary threads
/// and the data contained in Request, ticket, and info types are meant to be passed between threads
/// during the callback's lifetime.
#[derive(Debug)]
pub enum MountCommand {
    FetchPlaceholders {
        path: PathBuf,
        response: Sender<Result<GetPlacehodlerResult>>,
    },
    RefreshCredentials {
        credentials: Token,
    },
    /// Credential has become invalid (401, 40020, 40089 errors)
    CredentialInvalid,
    FetchData {
        path: PathBuf,
        ticket: ticket::FetchData,
        range: Range<u64>,
        response: Sender<Result<()>>,
    },
    ProcessFsEvents {
        events: GroupedFsEvents,
    },
    Sync {
        local_paths: Vec<PathBuf>,
        mode: SyncMode,
        user_initiated: bool,
    },
    Rename {
        source: PathBuf,
        target: PathBuf,
        response: Sender<Result<()>>,
    },
    Renamed {
        source: PathBuf,
        destination: PathBuf,
    },
}

// SAFETY: Windows CFAPI is designed to allow callbacks from arbitrary threads.
// The Request, ticket, and info types contain data that is valid for the duration
// of the callback and can be safely transferred between threads.
unsafe impl Send for MountCommand {}

/// Commands for the DriveManager
/// These can be sent from external sources like context menus or other UI components
#[derive(Debug)]
pub enum ManagerCommand {
    /// View a file or folder online in the web interface
    ViewOnline {
        path: PathBuf,
    },
    PersistConfig,
    GenerateThumbnail {
        path: PathBuf,
        response: Sender<Result<Bytes>>,
    },
    SyncNow {
        paths: Vec<PathBuf>,
        mode: SyncMode,
    },
    ResolveConflict {
        drive_id: String,
        file_id: i64,
        path: String,
        action: ConflictAction,
    },
    /// Show conflict resolution toast for a file
    ShowConflictToast {
        path: PathBuf,
    },
    /// Get drive status UI by sync root ID
    GetDriveStatusUI {
        syncroot_id: String,
        response: Sender<Result<Option<crate::drive::manager::DriveStatusUI>>>,
    },
    /// Open user profile URL in browser
    OpenProfileUrl {
        syncroot_id: String,
    },
    /// Open storage/capacity details URL in browser
    OpenStorageDetailsUrl {
        syncroot_id: String,
    },
    /// Request to open the sync status window in the UI
    OpenSyncStatusWindow,
    /// Request to open the settings window in the UI
    OpenSettingsWindow,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ConflictAction {
    KeepRemote,
    OverwriteRemote,
    SaveAsNew,
}

impl ConflictAction {
    pub fn from_str(action: &str) -> Option<Self> {
        match action {
            "keep_remote" => Some(Self::KeepRemote),
            "overwrite_remote" => Some(Self::OverwriteRemote),
            "save_as_new" => Some(Self::SaveAsNew),
            _ => None,
        }
    }
}

impl Mount {
    pub async fn fetch_data(
        &self,
        path: PathBuf,
        ticket: ticket::FetchData,
        range: Range<u64>,
    ) -> Result<()> {
        let config = self.config.read().await;
        let remote_base = config.remote_path.clone();
        let sync_path = config.sync_path.clone();
        drop(config);

        let uri = local_path_to_cr_uri(path.clone(), sync_path, remote_base)
            .context("failed to convert local path to cloudreve uri")?;

        let file_meta = self
            .inventory
            .query_by_path(path.to_str().unwrap_or(""))
            .context("failed to query metadata by path")?;

        let mut request: FileURLService = FileURLService::default();
        request.uris.push(uri.to_string());
        if let Some(meta) = file_meta {
            if !meta.etag.is_empty() {
                request.entity = Some(meta.etag.clone());
            }
        }
        let entity_url_res = self
            .cr_client
            .get_file_url(&request)
            .await
            .context("failed to get file url")?;

        // Get the download URL from the response
        let download_url = entity_url_res
            .urls
            .first()
            .context("no download URL in response")?
            .url
            .clone();

        tracing::debug!(target: "drive::commands", download_url = %download_url, "Download URL");

        // Calculate total bytes to fetch
        let total_bytes = range.end - range.start;

        // 4KB chunk size (required by Windows CFAPI)
        const CHUNK_SIZE: usize = 4096;
        // 64KB buffer for reading from network
        const BUFFER_SIZE: usize = 65536;

        // Create HTTP client and make a single range request
        let client = reqwest::Client::new();
        let range_header = format!("bytes={}-{}", range.start, range.end - 1);

        let response = client
            .get(&download_url)
            .header("Range", range_header)
            .send()
            .await
            .context("failed to send HTTP range request")?;

        if !response.status().is_success() && response.status().as_u16() != 206 {
            anyhow::bail!("HTTP request failed with status: {}", response.status());
        }

        // Stream the response and write in 4KB-aligned chunks
        let mut stream = response.bytes_stream();
        let mut current_offset = range.start;
        let mut bytes_transferred = 0u64;
        let mut accumulator: Vec<u8> = Vec::with_capacity(BUFFER_SIZE);

        use futures::StreamExt;

        while let Some(chunk_result) = stream.next().await {
            let chunk = chunk_result.context("failed to read chunk from stream")?;
            accumulator.extend_from_slice(&chunk);

            // Write out all aligned chunks at once if we have enough data
            if accumulator.len() >= CHUNK_SIZE {
                // Calculate how many complete aligned chunks we can write
                let aligned_size = (accumulator.len() / CHUNK_SIZE) * CHUNK_SIZE;
                let write_data = accumulator.drain(..aligned_size).collect::<Vec<u8>>();

                ticket.write_at(&write_data, current_offset).map_err(|e| {
                    anyhow::anyhow!("failed to write data at offset {}: {:?}", current_offset, e)
                })?;

                bytes_transferred += write_data.len() as u64;
                current_offset += write_data.len() as u64;

                // Report progress to Windows
                ticket
                    .report_progress(total_bytes, bytes_transferred)
                    .map_err(|e| anyhow::anyhow!("failed to report progress: {:?}", e))?;
            }
        }

        // Write any remaining data (last chunk, may be less than 4KB)
        if !accumulator.is_empty() {
            ticket.write_at(&accumulator, current_offset).map_err(|e| {
                anyhow::anyhow!("failed to write data at offset {}: {:?}", current_offset, e)
            })?;

            bytes_transferred += accumulator.len() as u64;
            // current_offset += accumulator.len() as u64;

            // Final progress report
            ticket
                .report_progress(total_bytes, bytes_transferred)
                .map_err(|e| anyhow::anyhow!("failed to report progress: {:?}", e))?;
        }

        tracing::debug!(
            target: "drive::commands",
            bytes_transferred = bytes_transferred,
            total = total_bytes,
            "Fetch data progress"
        );

        tracing::info!(
            target: "drive::commands",
            path = %path.display(),
            bytes = total_bytes,
            "Fetch data completed"
        );

        Ok(())
    }
    pub async fn fetch_placeholders(&self, path: PathBuf) -> Result<GetPlacehodlerResult> {
        let config = self.config.read().await;
        let remote_base = config.remote_path.clone();
        let sync_path = config.sync_path.clone();
        drop(config);

        let uri = local_path_to_cr_uri(path.clone(), sync_path, remote_base)
            .context("failed to convert local path to cloudreve uri")?;
        let mut placehodlers: Vec<FileResponse> = Vec::new();

        let mut previous_response = None;
        loop {
            let response = self
                .cr_client
                .list_files_all(previous_response.as_ref(), &uri.to_string(), PAGE_SIZE)
                .await?;

            for file in &response.res.files {
                tracing::debug!(target: "drive::mounts", file = %file.name, "Server file");
            }

            placehodlers.extend(response.res.files.clone());
            let has_more: bool = response.more;
            previous_response = Some(response);

            if !has_more {
                break;
            }
        }

        tracing::debug!(target: "drive::mounts", uri = %uri.to_string(), "Fetch file list from cloudreve");

        // Filter out files matching ignore patterns
        let matcher = self.ignore_matcher.read().await;
        if !matcher.is_empty() {
            placehodlers.retain(|file| {
                let local_file_path = path.join(&file.name);
                let ignored = matcher.is_match(&local_file_path);
                if ignored {
                    tracing::trace!(
                        target: "drive::commands",
                        path = %local_file_path.display(),
                        "Filtering ignored file from placeholders"
                    );
                }
                !ignored
            });
        }
        drop(matcher);

        Ok(GetPlacehodlerResult {
            files: placehodlers,
            local_path: path.clone(),
            remote_path: uri.clone(),
        })
    }

    pub async fn generate_thumbnail(&self, path: PathBuf) -> Result<Bytes> {
        let file_meta = self
            .inventory
            .query_by_path(path.to_str().unwrap_or(""))
            .context("failed to query metadata by path")?
            .ok_or_else(|| anyhow::anyhow!("no metadata found for path: {:?}", path))?;

        if file_meta.is_folder
            || file_meta
                .metadata
                .get(metadata::THUMBNAIL_DISABLED)
                .is_some()
        {
            return Err(anyhow::anyhow!("thumbnail disabled for path: {:?}", path));
        }

        let (sync_path, remote_base) = {
            let config = self.config.read().await;
            (config.sync_path.clone(), config.remote_path.to_string())
        };
        let uri = local_path_to_cr_uri(path.clone(), sync_path, remote_base)
            .context("failed to convert local path to cloudreve uri")?
            .to_string();
        let thumb_res = self.cr_client.get_file_thumb(uri.as_str(), None).await?;

        // Download the thumbnail
        let thumb_url = thumb_res.url;
        tracing::trace!(target: "drive::commands", thumb_url = %thumb_url, "Thumbnail URL");
        let thumb_response = reqwest::get(thumb_url).await?;
        // Make sure the response is successful
        if !thumb_response.status().is_success() {
            return Err(anyhow::anyhow!(
                "failed to download thumbnail: {:?}",
                thumb_response.status()
            ));
        }
        Ok(thumb_response.bytes().await?)
    }

    pub async fn rename_completed(&self, source: PathBuf, destination: PathBuf) -> Result<()> {
        // If source or destination is ignored, do nothing
        let matcher = self.ignore_matcher.read().await;
        if matcher.is_match(&source) || matcher.is_match(&destination) {
            tracing::debug!(target: "drive::commands", source = %source.display(), destination = %destination.display(), "Ignoring rename operation");
            return Ok(());
        }
        drop(matcher);

        // Commit rename in inventory
        self.inventory
            .rename_path(
                source
                    .to_str()
                    .context("failed to convert source path to string")?,
                destination
                    .to_str()
                    .context("failed to convert destination path to string")?,
            )
            .context("failed to rename path in inventory")?;

        // Cancel ongoing/pending tasks
        match self.task_queue.cancel_by_path(source.clone()).await {
            Ok(0) => {
                // Mark file as in-sync
                tracing::trace!(target: "drive::commands", path = %destination.display(), "Marking file as in-sync: OPEN");
                match OpenOptions::new()
                    .write_access()
                    .exclusive()
                    .open_with_retry(&destination)
                    .await
                {
                    Ok(mut handle) => {
                        tracing::trace!(target: "drive::commands", path = %destination.display(), "Marking file as in-sync");
                        if let Err(e) = handle.mark_in_sync(true, None) {
                            tracing::error!(target: "drive::commands", error = %e, "Failed to mark as in-sync");
                            return Err(e.into());
                        }
                        tracing::trace!(target: "drive::commands", path = %destination.display(), "Marked file as in-sync: complete");
                        Ok(())
                    }
                    Err(e) => {
                        tracing::error!(target: "drive::commands", error = %e, "Failed to open file after retries");
                        Err(e.into())
                    }
                }
            }
            Ok(count) => {
                tracing::info!(target: "drive::commands", path = %source.display(), count = count, "Cancelled tasks");
                // We have tasks canceled, we need to trigger sync on the moved file
                self.command_tx
                    .send(MountCommand::Sync {
                        local_paths: vec![destination.clone()],
                        mode: SyncMode::FullHierarchy,
                        user_initiated: false,
                    })
                    .context("failed to send sync command")?;
                Ok(())
            }
            Err(e) => {
                tracing::error!(target: "drive::commands", error = %e, "Failed to cancel tasks");
                Err(e.into())
            }
        }
    }

    pub async fn rename(&self, source: PathBuf, target: PathBuf) -> Result<()> {
        let (sync_path, remote_path) = {
            let config = self.config.read().await;
            (config.sync_path.clone(), config.remote_path.to_string())
        };

        // if target or source is not under sync root, do nothing
        if !target.starts_with(&sync_path) {
            // Source is being moved out of sync root
            //self.event_blocker
            //    .register_once(&EventKind::Remove(RemoveKind::Any), source.clone());
            return Ok(());
        }

        // If source or target is ignored, do nothing
        let matcher = self.ignore_matcher.read().await;
        if matcher.is_match(&source) || matcher.is_match(&target) {
            tracing::debug!(target: "drive::commands", source = %source.display(), target = %target.display(), "Ignoring rename operation");
            return Ok(());
        }
        drop(matcher);

        if !source.starts_with(&sync_path) {
            // Target is being moved into sync root - block the create event
            self.event_blocker
                .register_once(&EventKind::Create(CreateKind::Any), target.clone());
            return Ok(());
        }

        // if target and src under the same dir, trigger rename call
        let target_parent = target.parent().context("root cannot be moved")?;
        let source_parent = source.parent().context("root cannot be moved")?;
        if target_parent == source_parent {
            match self
                .cr_client
                .rename_file(&RenameFileService {
                    uri: local_path_to_cr_uri(source.clone(), sync_path, remote_path)?.to_string(),
                    new_name: target
                        .file_name()
                        .context("target cannot be moved")?
                        .to_string_lossy()
                        .to_string(),
                })
                .await
            {
                Ok(_) => {
                    // Block the modify name events for rename (From for source, To for target)
                    self.event_blocker.register_once(
                        &EventKind::Modify(ModifyKind::Name(RenameMode::From)),
                        source.clone(),
                    );

                    //self.task_queue.cancel_by_path(source.clone()).await?;
                    return Ok(());
                }
                Err(e) => {
                    tracing::error!(target: "drive::commands", error = %e, "Failed to rename file");
                    return Err(e.into());
                }
            }
        }

        // Process move call
        match self
            .cr_client
            .move_files(&MoveFileService {
                uris: vec![
                    local_path_to_cr_uri(source.clone(), sync_path.clone(), remote_path.clone())?
                        .to_string(),
                ],
                dst: local_path_to_cr_uri(
                    target_parent.to_path_buf(),
                    sync_path.clone(),
                    remote_path.clone(),
                )?
                .to_string(),
                copy: None,
            })
            .await
        {
            Ok(_) => {
                // Block remove event for source and create event for target
                self.event_blocker
                    .register_once(&EventKind::Remove(RemoveKind::Any), source.clone());
                self.event_blocker
                    .register_once(&EventKind::Create(CreateKind::Any), target.clone());
                return Ok(());
            }
            Err(e) => {
                tracing::error!(target: "drive::commands", error = %e, "Failed to move file");
                return Err(e.into());
            }
        }
    }

    pub async fn process_fs_events(&self, events: GroupedFsEvents) -> Result<()> {
        for (event_kind, events) in events {
            // Filter out events that were pre-registered by rename operations
            let filtered_events = self.event_blocker.filter_events(events, &event_kind);

            // Filter out events that are ignored
            let matcher = self.ignore_matcher.read().await;
            let filtered_events: Vec<Event> = filtered_events
                .into_iter()
                .filter(|event| {
                    let dominated_path = &event.paths[0];
                    let is_ignored = matcher.is_match(dominated_path);
                    if is_ignored {
                        tracing::trace!(
                            target: "drive::commands",
                            path = %dominated_path.display(),
                            "Ignoring event for path matching ignore pattern"
                        );
                    }
                    !is_ignored
                })
                .collect();
            drop(matcher);

            if filtered_events.is_empty() {
                continue;
            }

            // Extract configuration once to avoid repeated lock acquisition
            let (sync_path, remote_base) = {
                let config = self.config.read().await;
                (config.sync_path.clone(), config.remote_path.to_string())
            };

            let path_uri_mappings =
                self.build_path_uri_mappings(&filtered_events, &sync_path, &remote_base);

            if path_uri_mappings.is_empty() {
                tracing::warn!(target: "drive::commands", "No valid URIs to process");
                return Ok(());
            }

            match event_kind {
                EventKind::Remove(_) => {
                    self.process_fs_delete_events(path_uri_mappings, sync_path, remote_base)
                        .await?
                }
                EventKind::Create(_) => {
                    self.process_fs_create_events(path_uri_mappings, sync_path, remote_base)
                        .await?
                }
                EventKind::Modify(ModifyKind::Name(RenameMode::Both)) => {
                    self.process_fs_modify_name_event(filtered_events).await?
                }
                EventKind::Modify(_) => {
                    self.process_fs_modify_events(path_uri_mappings, sync_path, remote_base)
                        .await?
                }
                _ => (),
            }
        }
        Ok(())
    }

    pub async fn resolve_conflict(
        &self,
        action: ConflictAction,
        file_id: i64,
        path: String,
    ) -> Result<()> {
        let (local_path, conflict_state) = match file_id {
            0 => (path, None),
            _ => {
                let file_meta = self
                    .inventory
                    .query_by_id(file_id)
                    .context("failed to query file by ID")?
                    .ok_or_else(|| anyhow::anyhow!("file not found in inventory"))?;
                (file_meta.local_path, file_meta.conflict_state)
            }
        };

        if file_id > 0 && conflict_state.is_none() {
            return Err(anyhow::anyhow!("file is not conflicted"));
        }

        let (sync_root, drive_id) = {
            let config = self.config.read().await;
            (
                config.sync_path.clone(),
                Uuid::parse_str(&config.id).context("invalid drive ID")?,
            )
        };

        match action {
            ConflictAction::KeepRemote => {
                // Delete local file and trigger sync on origin path
                let cr_placeholder =
                    CrPlaceholder::new(local_path.clone(), sync_root.clone(), drive_id.clone());
                cr_placeholder
                    .delete_placeholder(self.inventory.clone())
                    .context("failed to delete local placeholder")?;
                self.event_blocker.register_once(
                    &EventKind::Remove(RemoveKind::Any),
                    local_path.clone().into(),
                );
                let command = MountCommand::Sync {
                    local_paths: vec![local_path.clone().into()],
                    mode: SyncMode::PathOnly,
                    user_initiated: false,
                };
                if let Err(e) = self.command_tx.send(command) {
                    tracing::error!(target: "drive::commands", error = %e, "Failed to send Sync command");
                }
            }
            ConflictAction::OverwriteRemote => {
                if file_id > 0 {
                    // Update conflict state with overwrite and trigger upload
                    self.inventory
                        .mark_as_conflicted(&local_path, Some(ConflictState::Override))
                        .context("failed to mark file as conflicted")?;
                    if conflict_state.unwrap() != ConflictState::Pending {
                        return Err(anyhow::anyhow!("file is not pending conflict resolution"));
                    }
                }

                tracing::info!(
                    target: "drive::sync",
                    id = %self.id,
                    path = %local_path,
                    "Queueing upload task"
                );

                if let Err(err) = self
                    .task_queue
                    .enqueue(TaskPayload::upload(local_path.clone()).with_force_override(true))
                    .await
                {
                    tracing::error!(
                        target: "drive::sync",
                        id = %self.id,
                        path = %local_path,
                        error = ?err,
                        "Failed to enqueue upload task"
                    );
                    return Err(err.into());
                }
            }
            ConflictAction::SaveAsNew => {
                // Generate a unique new filename for the conflicted local file
                let local_path_buf: PathBuf = local_path.clone().into();
                let new_path = generate_unique_filename(&local_path_buf);

                // Copy the local file to the new name
                std::fs::copy(&local_path_buf, &new_path)
                    .context("failed to copy file to new name")?;

                tracing::info!(
                    target: "drive::commands",
                    original = %local_path,
                    new = %new_path.display(),
                    "Copied conflicted file to new name"
                );

                // Delete local file and trigger sync on origin path (same as KeepRemote)
                let cr_placeholder =
                    CrPlaceholder::new(local_path.clone(), sync_root.clone(), drive_id.clone());
                cr_placeholder
                    .delete_placeholder(self.inventory.clone())
                    .context("failed to delete local placeholder")?;
                self.event_blocker.register_once(
                    &EventKind::Remove(RemoveKind::Any),
                    local_path.clone().into(),
                );

                // Trigger sync to restore the remote version at original path
                let command = MountCommand::Sync {
                    local_paths: vec![local_path.clone().into()],
                    mode: SyncMode::PathOnly,
                    user_initiated: false,
                };
                if let Err(e) = self.command_tx.send(command) {
                    tracing::error!(target: "drive::commands", error = %e, "Failed to send Sync command");
                }

                toast::send_general_text_toast(
                    &t!("conflictRenamed"),
                    &t!("newName","name" => new_path.
                file_name().unwrap_or_default().to_string_lossy().to_string()),
                );
            }
        }

        Ok(())
    }

    async fn process_fs_modify_name_event(&self, events: Vec<Event>) -> Result<()> {
        tracing::trace!(target: "drive::commands", count=events.len(), "Processing filesystem modify name event");
        for event in events {
            if event.paths.len() != 2 {
                tracing::error!(target: "drive::commands", count=event.paths.len(), "Invalid modify name event: not 2 paths");
                continue;
            }

            let to_file_info = match LocalFileInfo::from_path(event.paths[1].as_path()) {
                Ok(info) => info,
                Err(e) => {
                    tracing::error!(target: "drive::commands", path = %event.paths[1].display(), error = %e, "Failed to get local file info");
                    continue;
                }
            };

            if to_file_info.is_placeholder() {
                tracing::debug!(target: "drive::commands", path = %event.paths[1].display(), "Skip for placeholder rename event");
                continue;
            }

            // Cancel ongoing/pending tasks
            let result = self.task_queue.cancel_by_path(event.paths[0].clone()).await;
            match (result, to_file_info.in_sync()) {
                (Ok(0), true) => {
                    tracing::debug!(target: "drive::commands", path = %event.paths[0].display(), "No ongoing/pending tasks");
                }
                (Ok(count), _) => {
                    // Trigger sync on the moved file
                    self.command_tx
                        .send(MountCommand::Sync {
                            local_paths: vec![event.paths[1].clone()],
                            mode: SyncMode::FullHierarchy,
                            user_initiated: false,
                        })
                        .context("failed to send sync command")?;
                    tracing::info!(target: "drive::commands", path = %event.paths[0].display(), count = count, "Cancelled tasks");
                }
                (Err(e), _) => {
                    tracing::error!(target: "drive::commands", path = %event.paths[0].display(), error = %e, "Failed to cancel tasks");
                    continue;
                }
            }
        }
        Ok(())
    }

    async fn process_fs_modify_events(
        &self,
        path_uri_mappings: HashMap<String, PathBuf>,
        _sync_path: PathBuf,
        _remote_base: String,
    ) -> Result<()> {
        tracing::debug!(
            target: "drive::commands",
            uri_count = path_uri_mappings.len(),
            uris = ?path_uri_mappings,
            "Processing filesystem modify events"
        );

        for (_, path) in path_uri_mappings {
            let placeholder_info = match LocalFileInfo::from_path(path.as_path()) {
                Ok(info) => info,
                Err(e) => {
                    tracing::error!(target: "drive::commands", path = %path.display(), error = %e, "Failed to get local file info");
                    continue;
                }
            };
            if placeholder_info.is_directory() {
                continue;
            }

            // For pinned file but not on disk, hydrate it
            let pin_state = placeholder_info.pinned();
            if pin_state == PinState::Pinned && placeholder_info.partial_on_disk() {
                tracing::debug!(target: "drive::commands", path = %path.display(), "Hydrate pinned not on disk placeholder");
                let mut placeholder = match OpenOptions::new().open_win32(path.as_path()) {
                    Ok(p) => p,
                    Err(e) => {
                        tracing::error!(target: "drive::commands", path = %path.display(), error = %e, "Failed to open win32 file");
                        continue;
                    }
                };
                if let Err(e) = placeholder.hydrate(0..) {
                    tracing::error!(target: "drive::commands", path = %path.display(), error = %e, "Failed to hydrate placeholder");
                    continue;
                }
                tracing::trace!(target: "drive::commands", path = %path.display(), "Hydration complete");
                _ = notify_shell_change(&path, SHCNE_ATTRIBUTES);
                continue;
            } else if pin_state == PinState::Unpinned {
                tracing::debug!(target: "drive::commands", path = %path.display(), "Dehydrate unpinned file");

                let mut placeholder = match OpenOptions::new()
                    .open_win32_with_retry(path.as_path())
                    .await
                {
                    Ok(p) => p,
                    Err(e) => {
                        tracing::error!(
                            target: "drive::commands",
                            path = %path.display(),
                            error = %e,
                            "Failed to open win32 file for dehydration after retries"
                        );
                        continue;
                    }
                };

                match placeholder.dehydrate(0..) {
                    Ok(_) => {
                        tracing::trace!(target: "drive::commands", path = %path.display(), "Dehydration complete");
                        _ = notify_shell_change(&path, SHCNE_ATTRIBUTES);
                    }
                    Err(e) => {
                        tracing::error!(
                            target: "drive::commands",
                            path = %path.display(),
                            error = %e,
                            "Failed to dehydrate placeholder"
                        );
                    }
                }
                continue;
            }

            // General modification, quque a upload task if not exist
            if !placeholder_info.in_sync() {
                tracing::debug!(target: "drive::commands", path = %path.display(), "Queuing upload task for modified file");
                let payload = TaskPayload::upload(path.clone());
                let result = self
                    .task_queue
                    .enqueue(payload)
                    .await
                    .context("Failed to enqueue upload task");
                if result.is_err() {
                    tracing::error!(target: "drive::commands", path = %path.display(), error = ?result, "Failed to enqueue upload task");
                    continue;
                }
                continue;
            }
        }

        Ok(())
    }

    async fn process_fs_create_events(
        &self,
        path_uri_mappings: HashMap<String, PathBuf>,
        _sync_path: PathBuf,
        _remote_base: String,
    ) -> Result<()> {
        tracing::debug!(
            target: "drive::commands",
            uri_count = path_uri_mappings.len(),
            uris = ?path_uri_mappings,
            "Processing filesystem create events"
        );

        for (_remote_uri, path) in path_uri_mappings {
            let payload = TaskPayload::upload(path.clone());

            self.task_queue
                .enqueue(payload)
                .await
                .context("Failed to enqueue upload task")?;
        }

        Ok(())
    }

    /// Process filesystem delete events by synchronizing deletions with the remote server
    /// and updating the local inventory.
    ///
    /// This function:
    /// 1. Converts local paths to remote URIs
    /// 2. Sends batch delete request to the server
    /// 3. Handles partial failures in batch operations
    /// 4. Updates local inventory for successfully deleted files
    async fn process_fs_delete_events(
        &self,
        path_uri_mappings: HashMap<String, PathBuf>,
        _sync_path: PathBuf,
        _remote_base: String,
    ) -> Result<()> {
        tracing::debug!(
            target: "drive::commands",
            uri_count = path_uri_mappings.len(),
            uris = ?path_uri_mappings,
            "Processing filesystem delete events"
        );

        let uris: Vec<String> = path_uri_mappings.keys().cloned().collect();

        // cancel related tasks
        for path in path_uri_mappings.values() {
            let result = self.task_queue.cancel_by_path(path.as_path()).await;
            match result {
                Ok(count) => {
                    tracing::info!(target: "drive::commands", path = %path.display(), count = count, "Cancelled tasks");
                }
                Err(e) => {
                    tracing::error!(target: "drive::commands", path = %path.display(), error = %e, "Failed to cancel tasks");
                    continue;
                }
            }
        }

        tracing::info!(
            target: "drive::commands",
            uri_count = uris.len(),
            "Sending batch delete request to server"
        );

        // Attempt to delete files on the remote server
        let delete_result = self
            .cr_client
            .delete_files(&DeleteFileService {
                uris: uris.clone(),
                unlink: None,
                skip_soft_delete: None,
            })
            .await;

        // Determine which files were successfully deleted
        let successful_paths = match delete_result {
            Ok(_) => {
                tracing::info!(
                    target: "drive::commands",
                    count = uris.len(),
                    "Successfully deleted all files from server"
                );
                // All deletions succeeded
                path_uri_mappings.values().cloned().collect()
            }
            Err(e) => {
                tracing::error!(
                    target: "drive::commands",
                    error = %e,
                    "Batch delete operation failed"
                );
                self.handle_delete_error(e, &path_uri_mappings).await?
            }
        };

        if !successful_paths.is_empty() {
            // Update local inventory to reflect successful deletions
            self.update_inventory_for_deletions(&successful_paths)
                .context("Failed to update local inventory after deletions")?;
        }

        Ok(())
    }

    /// Build a mapping from remote URIs to local paths for the given events.
    /// Logs warnings for any paths that cannot be converted to URIs.
    fn build_path_uri_mappings(
        &self,
        events: &[Event],
        sync_path: &Path,
        remote_base: &str,
    ) -> HashMap<String, PathBuf> {
        events
            .iter()
            .flat_map(|event| &event.paths)
            .filter_map(|path| {
                match local_path_to_cr_uri(
                    path.clone(),
                    sync_path.to_path_buf(),
                    remote_base.to_string(),
                ) {
                    Ok(uri) => Some((uri.to_string(), path.clone())),
                    Err(e) => {
                        tracing::warn!(
                            target: "drive::commands",
                            path = %path.display(),
                            error = %e,
                            "Failed to convert local path to remote URI"
                        );
                        None
                    }
                }
            })
            .collect()
    }

    /// Handle deletion errors and determine which paths were successfully deleted.
    /// Returns the list of paths that were successfully deleted despite partial failures.
    async fn handle_delete_error(
        &self,
        error: ApiError,
        path_uri_mappings: &HashMap<String, PathBuf>,
    ) -> Result<Vec<PathBuf>> {
        match error {
            ApiError::BatchError {
                message: _,
                aggregated_errors: Some(errors),
            } => {
                // Collect failed URIs
                let failed_uris: std::collections::HashSet<_> = errors.keys().cloned().collect();

                tracing::warn!(
                    target: "drive::commands",
                    failed_count = failed_uris.len(),
                    total_count = path_uri_mappings.len(),
                    "Partial batch delete failure"
                );

                let mut successful_paths = Vec::new();
                let mut failed_paths = Vec::new();

                for (uri, path) in path_uri_mappings {
                    if failed_uris.contains(uri) {
                        failed_paths.push(path.clone());
                    } else {
                        successful_paths.push(path.clone());
                    }
                }

                if !failed_paths.is_empty() {
                    tracing::info!(
                        target: "drive::commands",
                        failed_count = failed_paths.len(),
                        "Scheduling resync for failed deletions"
                    );
                    let command = MountCommand::Sync {
                        local_paths: failed_paths.clone(),
                        mode: SyncMode::PathOnly,
                        user_initiated: false,
                    };
                    if let Err(e) = self.command_tx.send(command) {
                        tracing::error!(
                            target: "drive::commands",
                            error = %e,
                            "Failed to send Sync command"
                        );
                    }
                }

                Ok(successful_paths)
            }
            _ => {
                // For non-batch errors, all operations failed
                tracing::error!(
                    target: "drive::commands",
                    error = %error,
                    "Complete batch delete failure - no files were deleted"
                );
                let command = MountCommand::Sync {
                    local_paths: path_uri_mappings.values().cloned().collect(),
                    mode: SyncMode::PathOnly,
                    user_initiated: false,
                };
                if let Err(e) = self.command_tx.send(command) {
                    tracing::error!(target: "drive::commands", error = %e, "Failed to send Sync command");
                }
                Ok(Vec::new())
            }
        }
    }

    /// Update the local inventory to remove entries for successfully deleted paths.
    fn update_inventory_for_deletions(&self, paths: &[PathBuf]) -> Result<()> {
        let path_strs: Vec<&str> = paths
            .iter()
            .filter_map(|path| {
                path.to_str().or_else(|| {
                    tracing::warn!(
                        target: "drive::commands",
                        path = ?path,
                        "Cannot convert path to string for inventory update"
                    );
                    None
                })
            })
            .collect();

        if !path_strs.is_empty() {
            let count = path_strs.len();
            self.inventory.batch_delete_by_path(path_strs)?;
            tracing::debug!(
                target: "drive::commands",
                count = count,
                "Updated local inventory after deletions"
            );
        }

        Ok(())
    }
}
