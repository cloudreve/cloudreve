//! Download task implementation for downloading remote files to local placeholders.
//!
//! This module provides a download task that:
//! - Downloads file content from remote server to a temporary location
//! - Tracks download progress with speed and ETA calculation
//! - Replaces the placeholder file content atomically when finished
//! - Uses CrPlaceholder to convert and mark the file as in-sync
//! - Only operates on hydrated placeholder files

use std::{
    path::PathBuf,
    str::FromStr,
    sync::{
        Arc,
        atomic::{AtomicU64, Ordering},
    },
    time::{Duration, Instant},
};

use anyhow::{Context, Result};
use cloudreve_api::{Client, api::ExplorerApi, models::explorer::FileURLService};
use dashmap::DashMap;
use futures::StreamExt;
use tokio::io::AsyncWriteExt;
use tokio_util::sync::CancellationToken;
use tracing::{debug, info, warn};
use uuid::Uuid;

use crate::{
    cfapi::placeholder::LocalFileInfo,
    drive::{placeholder::CrPlaceholder, utils::local_path_to_cr_uri},
    inventory::{FileMetadata, InventoryDb},
    tasks::queue::QueuedTask,
};

use super::types::TaskProgress;

/// Progress tracker for download operations.
/// Uses atomic counters for thread-safe byte tracking and sliding window for speed calculation.
pub struct DownloadProgressTracker {
    /// Total file size in bytes
    total_size: u64,
    /// Downloaded bytes (atomic for thread safety)
    downloaded_bytes: AtomicU64,
    /// Speed calculation samples
    samples: std::sync::Mutex<Vec<(Instant, u64)>>,
    /// Window duration for speed calculation
    window_duration: Duration,
}

impl DownloadProgressTracker {
    pub fn new(total_size: u64) -> Self {
        Self {
            total_size,
            downloaded_bytes: AtomicU64::new(0),
            samples: std::sync::Mutex::new(Vec::with_capacity(32)),
            window_duration: Duration::from_secs(10),
        }
    }

    /// Add downloaded bytes
    pub fn add_bytes(&self, bytes: u64) {
        self.downloaded_bytes.fetch_add(bytes, Ordering::SeqCst);
    }

    /// Get current downloaded bytes
    pub fn downloaded(&self) -> u64 {
        self.downloaded_bytes.load(Ordering::SeqCst)
    }

    /// Calculate current speed and return progress update
    pub fn create_update(&self) -> DownloadProgressUpdate {
        let downloaded = self.downloaded();
        let now = Instant::now();

        // Calculate speed using sliding window
        let speed = {
            let mut samples = self.samples.lock().unwrap();
            samples.push((now, downloaded));

            // Remove old samples outside window
            let cutoff = now - self.window_duration;
            samples.retain(|(t, _)| *t >= cutoff);

            if samples.len() >= 2 {
                let (oldest_time, oldest_bytes) = samples.first().unwrap();
                let elapsed = now.duration_since(*oldest_time);
                if elapsed.as_millis() > 0 {
                    let bytes_diff = downloaded.saturating_sub(*oldest_bytes);
                    (bytes_diff as f64 / elapsed.as_secs_f64()) as u64
                } else {
                    0
                }
            } else {
                0
            }
        };

        let progress = if self.total_size > 0 {
            (downloaded as f64 / self.total_size as f64).clamp(0.0, 1.0)
        } else {
            1.0
        };

        let eta_seconds = if speed > 0 && downloaded < self.total_size {
            Some((self.total_size - downloaded) / speed)
        } else {
            None
        };

        DownloadProgressUpdate {
            total_size: self.total_size,
            downloaded,
            progress,
            speed_bytes_per_sec: speed,
            eta_seconds,
        }
    }
}

/// Progress update for downloads
#[derive(Debug, Clone)]
pub struct DownloadProgressUpdate {
    /// Total file size in bytes
    pub total_size: u64,
    /// Downloaded bytes so far
    pub downloaded: u64,
    /// Progress percentage (0.0 - 1.0)
    pub progress: f64,
    /// Download speed in bytes per second
    pub speed_bytes_per_sec: u64,
    /// Estimated time remaining in seconds
    pub eta_seconds: Option<u64>,
}

/// In-memory progress reporter for download tasks
pub struct InMemoryDownloadProgressReporter {
    task_id: String,
    progress_map: Arc<DashMap<String, TaskProgress>>,
}

impl InMemoryDownloadProgressReporter {
    pub fn new(task_id: String, progress_map: Arc<DashMap<String, TaskProgress>>) -> Self {
        Self {
            task_id,
            progress_map,
        }
    }

    /// Update progress from download progress update
    pub fn on_progress(&self, update: &DownloadProgressUpdate) {
        if let Some(mut entry) = self.progress_map.get_mut(&self.task_id) {
            entry.progress = update.progress;
            entry.processed_bytes = Some(update.downloaded as i64);
            entry.total_bytes = Some(update.total_size as i64);
            entry.speed_bytes_per_sec = update.speed_bytes_per_sec;
            entry.eta_seconds = update.eta_seconds;
        }
    }
}

/// Download task that downloads a file from remote to local placeholder
pub struct DownloadTask<'a> {
    inventory: Arc<InventoryDb>,
    cr_client: Arc<Client>,
    drive_id: &'a str,
    sync_path: PathBuf,
    remote_base: String,
    task: &'a QueuedTask,
    local_file_info: Option<LocalFileInfo>,
    inventory_meta: Option<FileMetadata>,
    /// Latest file info fetched from remote before download
    remote_file_info: Option<cloudreve_api::models::explorer::FileResponse>,
    cancel_token: CancellationToken,
    progress_map: Arc<DashMap<String, TaskProgress>>,
}

impl<'a> DownloadTask<'a> {
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
            local_file_info: None,
            inventory_meta: None,
            remote_file_info: None,
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

    /// Execute the download task
    pub async fn execute(&mut self) -> Result<()> {
        // Get local file info
        let local_file_info = LocalFileInfo::from_path(&self.task.payload.local_path)
            .context("failed to get local file info")?;

        if !local_file_info.exists {
            info!(
                target: "tasks::download",
                task_id = %self.task.task_id,
                local_path = %self.task.payload.local_path_display(),
                "Local file does not exist, skipping download"
            );
            return Ok(());
        }

        if local_file_info.is_directory {
            info!(
                target: "tasks::download",
                task_id = %self.task.task_id,
                local_path = %self.task.payload.local_path_display(),
                "Cannot download a directory, skipping"
            );
            return Ok(());
        }

        // Check if file is a placeholder and is hydrated (has content on disk)
        if !local_file_info.is_placeholder() {
            info!(
                target: "tasks::download",
                task_id = %self.task.task_id,
                local_path = %self.task.payload.local_path_display(),
                "File is not a placeholder, skipping download"
            );
            return Ok(());
        }

        // partial_on_disk means the file content is NOT fully present locally
        // We need the file to be hydrated (NOT partial_on_disk) to replace its content
        if local_file_info.partial_on_disk() {
            info!(
                target: "tasks::download",
                task_id = %self.task.task_id,
                local_path = %self.task.payload.local_path_display(),
                "File is not fully hydrated, skipping download - file must be hydrated first"
            );
            return Ok(());
        }

        self.local_file_info = Some(local_file_info);

        // Get inventory metadata - required for download
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

        // Fail if file is not in inventory - we need the metadata to download
        if self.inventory_meta.is_none() {
            anyhow::bail!(
                "File not found in inventory: {}. Cannot download without inventory metadata.",
                path_str
            );
        }

        // Perform the download
        self.download_and_replace().await
    }

    /// Download file from remote and replace local placeholder content
    async fn download_and_replace(&mut self) -> Result<()> {
        let local_path = &self.task.payload.local_path;
        info!(
            target: "tasks::download",
            task_id = %self.task.task_id,
            local_path = %self.task.payload.local_path_display(),
            "Starting file download"
        );

        // Get remote URI
        let uri = local_path_to_cr_uri(
            local_path.clone(),
            self.sync_path.clone(),
            self.remote_base.clone(),
        )
        .context("failed to convert local path to cloudreve uri")?
        .to_string();

        // Fetch latest file info from remote before download
        let file_info = self
            .cr_client
            .get_file_info(&cloudreve_api::models::explorer::GetFileInfoService {
                uri: Some(uri.clone()),
                id: None,
                extended: None,
                folder_summary: None,
            })
            .await
            .context("failed to get file info from remote")?;

        let file_size = file_info.size as u64;
        self.remote_file_info = Some(file_info);

        // Get download URL from server using inventory metadata for entity validation
        let mut request = FileURLService::default();
        request.uris.push(uri.clone());
        if self.remote_file_info.as_ref().map(|f|f.primary_entity.is_some()).unwrap_or(false) {
            request.entity = self.remote_file_info.as_ref().unwrap().primary_entity.clone();
        }

        let entity_url_res = self
            .cr_client
            .get_file_url(&request)
            .await
            .context("failed to get file url")?;

        let download_url = entity_url_res
            .urls
            .first()
            .context("no download URL in response")?
            .url
            .clone();

        debug!(
            target: "tasks::download",
            task_id = %self.task.task_id,
            download_url = %download_url,
            file_size = file_size,
            "Got download URL"
        );

        // Create temp file for download
        let temp_dir = std::env::temp_dir();
        let temp_file_name = format!("cloudreve_download_{}", self.task.task_id);
        let temp_path = temp_dir.join(&temp_file_name);

        // Clean up any existing temp file
        if temp_path.exists() {
            std::fs::remove_file(&temp_path).ok();
        }

        // Create progress tracker and reporter
        let tracker = Arc::new(DownloadProgressTracker::new(file_size));
        let reporter = InMemoryDownloadProgressReporter::new(
            self.task.task_id.clone(),
            Arc::clone(&self.progress_map),
        );

        // Download to temp file
        let download_result = self
            .download_to_temp(&download_url, &temp_path, tracker.clone(), &reporter)
            .await;

        match download_result {
            Ok(()) => {
                // Report final progress
                let final_update = tracker.create_update();
                reporter.on_progress(&final_update);

                // Replace placeholder file with downloaded content and commit
                self.replace_and_commit_placeholder(&temp_path)
                    .context("failed to replace and commit placeholder")?;

                // Clean up temp file
                if temp_path.exists() {
                    std::fs::remove_file(&temp_path).ok();
                }

                info!(
                    target: "tasks::download",
                    task_id = %self.task.task_id,
                    local_path = %self.task.payload.local_path_display(),
                    "Download completed successfully"
                );

                Ok(())
            }
            Err(e) => {
                // Clean up temp file on error
                if temp_path.exists() {
                    std::fs::remove_file(&temp_path).ok();
                }
                Err(e)
            }
        }
    }

    /// Download file content to a temporary file
    async fn download_to_temp(
        &self,
        url: &str,
        temp_path: &PathBuf,
        tracker: Arc<DownloadProgressTracker>,
        reporter: &InMemoryDownloadProgressReporter,
    ) -> Result<()> {
        let client = reqwest::Client::new();
        let response = client
            .get(url)
            .send()
            .await
            .context("failed to send download request")?;

        if !response.status().is_success() {
            anyhow::bail!("Download request failed with status: {}", response.status());
        }

        // Create temp file
        let mut file = tokio::fs::File::create(&temp_path)
            .await
            .context("failed to create temp file")?;

        // Stream download with progress tracking
        let mut stream = response.bytes_stream();
        let mut last_report = Instant::now();
        const REPORT_INTERVAL: Duration = Duration::from_millis(100);

        while let Some(chunk_result) = stream.next().await {
            // Check for cancellation
            if self.cancel_token.is_cancelled() {
                anyhow::bail!("Download cancelled");
            }

            let chunk = chunk_result.context("failed to read chunk from stream")?;
            file.write_all(&chunk)
                .await
                .context("failed to write chunk to temp file")?;

            tracker.add_bytes(chunk.len() as u64);

            // Report progress at intervals to avoid too frequent updates
            if last_report.elapsed() >= REPORT_INTERVAL {
                let update = tracker.create_update();
                reporter.on_progress(&update);
                last_report = Instant::now();
            }
        }

        file.flush().await.context("failed to flush temp file")?;

        Ok(())
    }

    /// Replace the placeholder file content with the downloaded file and commit using CrPlaceholder
    fn replace_and_commit_placeholder(&mut self, temp_path: &PathBuf) -> Result<()> {
        let local_path = &self.task.payload.local_path;
        let remote_file_info = self
            .remote_file_info
            .as_ref()
            .expect("remote_file_info should be set");

        debug!(
            target: "tasks::download",
            task_id = %self.task.task_id,
            temp_path = %temp_path.display(),
            local_path = %local_path.display(),
            "Replacing placeholder content"
        );

        // Use Windows ReplaceFileW for atomic replacement
        // This preserves file attributes and provides atomicity
        #[cfg(windows)]
        {
            use std::ffi::OsStr;
            use std::os::windows::ffi::OsStrExt;
            use windows::Win32::Storage::FileSystem::{REPLACE_FILE_FLAGS, ReplaceFileW};
            use windows::core::PCWSTR;

            // Convert paths to wide strings
            let local_wide: Vec<u16> = OsStr::new(local_path)
                .encode_wide()
                .chain(std::iter::once(0))
                .collect();
            let temp_wide: Vec<u16> = OsStr::new(temp_path)
                .encode_wide()
                .chain(std::iter::once(0))
                .collect();

            // ReplaceFileW replaces the destination with the source
            // The source file is deleted after the operation
            let result = unsafe {
                ReplaceFileW(
                    PCWSTR::from_raw(local_wide.as_ptr()),
                    PCWSTR::from_raw(temp_wide.as_ptr()),
                    PCWSTR::null(),        // No backup file
                    REPLACE_FILE_FLAGS(0), // No flags
                    None,                  // Reserved
                    None,                  // Reserved
                )
            };

            if let Err(e) = result {
                // If ReplaceFileW fails, fall back to copy + delete
                warn!(
                    target: "tasks::download",
                    task_id = %self.task.task_id,
                    error = ?e,
                    "ReplaceFileW failed, falling back to copy"
                );
                std::fs::copy(temp_path, local_path)
                    .context("failed to copy temp file to local path")?;
            }
        }

        #[cfg(not(windows))]
        {
            // On non-Windows, just copy the file
            std::fs::copy(temp_path, local_path)
                .context("failed to copy temp file to local path")?;
        }

        // Use CrPlaceholder to convert and mark as in-sync
        let drive_id = Uuid::from_str(self.drive_id).context("invalid drive ID")?;
        // Create CrPlaceholder and commit changes using the latest remote file info
        let mut cr_placeholder =
            CrPlaceholder::new(local_path.clone(), self.sync_path.clone(), drive_id)
                .with_remote_file(remote_file_info);

        cr_placeholder
            .commit(self.inventory.clone())
            .context("failed to commit placeholder")?;

        debug!(
            target: "tasks::download",
            task_id = %self.task.task_id,
            local_path = %local_path.display(),
            "Placeholder committed successfully"
        );

        Ok(())
    }
}
