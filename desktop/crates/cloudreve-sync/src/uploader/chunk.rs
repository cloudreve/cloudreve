//! Chunk-based upload logic with streaming support and progress tracking

use crate::uploader::UploaderConfig;
use crate::uploader::encrypt::EncryptionConfig;
use crate::uploader::error::UploadError;
use crate::uploader::progress::{ProgressCallback, ProgressTracker};
use crate::uploader::providers::{self, PolicyType};
use crate::uploader::session::UploadSession;
use anyhow::{Context, Result};
use bytes::Bytes;
use cloudreve_api::Client as CrClient;
use futures::Stream;
use reqwest::Client as HttpClient;
use serde::{Deserialize, Serialize};
use std::io;
use std::path::{Path, PathBuf};
use std::pin::Pin;
use std::sync::Arc;
use std::sync::atomic::{AtomicBool, AtomicU64, AtomicUsize, Ordering};
use std::task::{Context as TaskContext, Poll};
use std::time::Duration;
use tokio::fs::File;
use tokio::io::{AsyncRead, AsyncSeekExt, BufReader, ReadBuf, SeekFrom};
use tokio::sync::{Mutex, Notify};
use tokio_util::io::ReaderStream;
use tokio_util::sync::CancellationToken;
use tracing::{debug, error, info, warn};

/// Per-chunk progress tracking
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChunkProgress {
    /// Chunk index
    pub index: usize,
    /// Bytes uploaded for this chunk
    pub loaded: u64,
    /// ETag returned by storage provider (for S3-like providers)
    pub etag: Option<String>,
}

impl ChunkProgress {
    /// Create a new chunk progress entry
    pub fn new(index: usize) -> Self {
        Self {
            index,
            loaded: 0,
            etag: None,
        }
    }

    /// Check if chunk upload is complete
    pub fn is_complete(&self) -> bool {
        self.loaded > 0
    }
}

/// Metadata about a single chunk (without the data)
#[derive(Debug, Clone)]
pub struct ChunkInfo {
    /// Chunk index
    pub index: usize,
    /// Expected chunk size
    pub size: u64,
    /// Byte offset in file
    pub offset: u64,
}

impl ChunkInfo {
    /// Create new chunk info
    pub fn new(index: usize, offset: u64, size: u64) -> Self {
        Self {
            index,
            offset,
            size,
        }
    }
}

/// Buffer size for streaming reads (64KB)
const STREAM_BUFFER_SIZE: usize = 64 * 1024;

/// A limited async reader that reads only a specific range from a file,
/// optionally applying encryption on-the-fly.
pub struct ChunkReader {
    reader: BufReader<File>,
    // handle: ArcWin32Handle,
    // placeholder: Placeholder,
    encryption: Option<EncryptionConfig>,
    start_offset: u64,
    position: u64,
    remaining: u64,
}

impl ChunkReader {
    /// Create a new chunk reader for a specific byte range
    pub async fn new(
        path: &Path,
        offset: u64,
        size: u64,
        encryption: Option<EncryptionConfig>,
    ) -> Result<Self> {
        // let placeholder = OpenOptions::new()
        // .exclusive()
        //     .open(path)
        //     .context("failed to open file")?;
        // let protected_handle = placeholder
        //     .win32_handle()
        //     .context("failed to get win32 handle")?;
        let file = File::open(path).await.context("failed to open file")?;
        let mut reader = BufReader::with_capacity(STREAM_BUFFER_SIZE, file);
        reader.seek(SeekFrom::Start(offset)).await?;

        Ok(Self {
            reader,
            encryption,
            start_offset: offset,
            position: 0,
            remaining: size,
        })
    }

    /// Get the total size of this chunk
    #[allow(dead_code)]
    pub fn size(&self) -> u64 {
        self.position + self.remaining
    }
}

impl AsyncRead for ChunkReader {
    fn poll_read(
        mut self: Pin<&mut Self>,
        cx: &mut TaskContext<'_>,
        buf: &mut ReadBuf<'_>,
    ) -> Poll<io::Result<()>> {
        if self.remaining == 0 {
            return Poll::Ready(Ok(()));
        }

        // Limit read to remaining bytes
        let max_read = (self.remaining as usize).min(buf.remaining());
        let mut limited_buf = buf.take(max_read);
        let before = limited_buf.filled().len();

        // Pin the inner reader - this is safe because BufReader<File> is Unpin
        let reader = Pin::new(&mut self.reader);

        match reader.poll_read(cx, &mut limited_buf) {
            Poll::Ready(Ok(())) => {
                let bytes_read = limited_buf.filled().len() - before;
                if bytes_read == 0 {
                    // EOF reached
                    return Poll::Ready(Ok(()));
                }

                // Apply encryption if configured
                if let Some(ref config) = self.encryption {
                    let file_offset = self.start_offset + self.position;
                    // Get the newly read bytes and encrypt them in place
                    let start = buf.filled().len();
                    unsafe {
                        buf.assume_init(bytes_read);
                    }
                    buf.advance(bytes_read);
                    let filled = buf.filled_mut();
                    let encrypted_slice = &mut filled[start..start + bytes_read];
                    config.encrypt_at_offset(encrypted_slice, file_offset);
                } else {
                    unsafe {
                        buf.assume_init(bytes_read);
                    }
                    buf.advance(bytes_read);
                }

                self.position += bytes_read as u64;
                self.remaining -= bytes_read as u64;

                Poll::Ready(Ok(()))
            }
            Poll::Ready(Err(e)) => Poll::Ready(Err(e)),
            Poll::Pending => Poll::Pending,
        }
    }
}

/// A stream that yields chunks of bytes from a ChunkReader.
/// Uses tokio_util's ReaderStream internally for simplicity.
pub struct ChunkStream {
    inner: ReaderStream<ChunkReader>,
}

impl ChunkStream {
    /// Create a new chunk stream from a reader
    pub fn new(reader: ChunkReader) -> Self {
        Self {
            inner: ReaderStream::with_capacity(reader, STREAM_BUFFER_SIZE),
        }
    }

    /// Create a chunk stream from file path and chunk info
    pub async fn from_chunk(
        path: &Path,
        chunk: &ChunkInfo,
        encryption: Option<EncryptionConfig>,
    ) -> Result<Self> {
        let reader = ChunkReader::new(path, chunk.offset, chunk.size, encryption).await?;
        Ok(Self::new(reader))
    }
}

impl Stream for ChunkStream {
    type Item = Result<Bytes, io::Error>;

    fn poll_next(mut self: Pin<&mut Self>, cx: &mut TaskContext<'_>) -> Poll<Option<Self::Item>> {
        Pin::new(&mut self.inner).poll_next(cx)
    }
}

/// A stream wrapper that tracks progress at byte level.
/// Reports progress through the ProgressTracker with throttling.
pub struct ProgressStream<S> {
    inner: S,
    tracker: Arc<ProgressTracker>,
    /// Shared counter for bytes sent, accessible after stream is consumed
    bytes_sent_counter: Arc<AtomicU64>,
}

impl<S> ProgressStream<S> {
    /// Create a new progress-aware stream
    pub fn new(inner: S, tracker: Arc<ProgressTracker>) -> Self {
        Self {
            inner,
            tracker,
            bytes_sent_counter: Arc::new(AtomicU64::new(0)),
        }
    }

    /// Get a handle to the bytes sent counter that remains valid after stream is consumed
    pub fn bytes_sent_counter(&self) -> Arc<AtomicU64> {
        Arc::clone(&self.bytes_sent_counter)
    }
}

impl<S> Stream for ProgressStream<S>
where
    S: Stream<Item = Result<Bytes, io::Error>> + Unpin,
{
    type Item = Result<Bytes, io::Error>;

    fn poll_next(mut self: Pin<&mut Self>, cx: &mut TaskContext<'_>) -> Poll<Option<Self::Item>> {
        match Pin::new(&mut self.inner).poll_next(cx) {
            Poll::Ready(Some(Ok(bytes))) => {
                let len = bytes.len() as u64;
                self.bytes_sent_counter.fetch_add(len, Ordering::SeqCst);
                self.tracker.add_bytes(len);
                Poll::Ready(Some(Ok(bytes)))
            }
            other => other,
        }
    }
}

/// Chunk uploader that handles uploading chunks to different providers
pub struct ChunkUploader {
    http_client: HttpClient,
    cr_client: Arc<CrClient>,
    policy_type: PolicyType,
    config: UploaderConfig,
}

impl ChunkUploader {
    /// Create a new chunk uploader
    pub fn new(
        http_client: HttpClient,
        cr_client: Arc<CrClient>,
        policy_type: PolicyType,
        config: UploaderConfig,
    ) -> Self {
        Self {
            http_client,
            cr_client,
            policy_type,
            config,
        }
    }

    /// Upload all chunks for a file with progress tracking
    ///
    /// Supports both sequential (concurrency=1) and concurrent chunk uploads
    /// based on the session's storage policy configuration.
    pub async fn upload_all<P: ProgressCallback + 'static>(
        &self,
        local_path: &Path,
        session: &mut UploadSession,
        progress_callback: Arc<P>,
        cancel_token: &CancellationToken,
    ) -> Result<()> {
        let concurrency = session.chunk_concurrency();

        info!(
            target: "uploader::chunk",
            local_path = %local_path.display(),
            num_chunks = session.num_chunks(),
            policy_type = ?self.policy_type,
            concurrency = concurrency,
            "Starting chunk upload"
        );

        // Get encryption config if needed
        let encryption = session
            .encrypt_metadata
            .as_ref()
            .map(|meta| EncryptionConfig::from_metadata(meta))
            .transpose()?;

        // Get pending chunks
        let pending_chunks = session.pending_chunks();
        if pending_chunks.is_empty() {
            info!(
                target: "uploader::chunk",
                "All chunks already uploaded"
            );
            return Ok(());
        }

        info!(
            target: "uploader::chunk",
            pending = pending_chunks.len(),
            total = session.num_chunks(),
            concurrency = concurrency,
            "Uploading pending chunks"
        );

        // Create progress tracker
        let tracker = ProgressTracker::new(session.file_size, session.num_chunks());

        // Initialize tracker with already completed chunks
        for chunk in session.chunk_progress.iter().filter(|c| c.is_complete()) {
            tracker.add_bytes(chunk.loaded);
            tracker.complete_chunk();
        }

        // Spawn progress reporter task
        let reporter_tracker = Arc::clone(&tracker);
        let reporter_cancel = cancel_token.clone();
        let reporter_callback = Arc::clone(&progress_callback);

        let reporter_handle = tokio::spawn(async move {
            loop {
                tokio::select! {
                    _ = tokio::time::sleep(Duration::from_millis(500)) => {
                        let update = reporter_tracker.create_update().await;
                        reporter_callback.on_progress(update);
                    }
                    _ = reporter_cancel.cancelled() => {
                        break;
                    }
                }
            }
        });

        // Upload chunks with dynamic pool (works for any concurrency level)
        let result = self
            .upload_chunks_with_pool(
                local_path,
                session,
                &pending_chunks,
                encryption,
                &tracker,
                cancel_token,
                concurrency,
            )
            .await;

        // Stop the reporter
        reporter_handle.abort();

        // Send final progress update
        let final_update = tracker.create_update().await;
        progress_callback.on_progress(final_update);

        result
    }

    /// Upload chunks using a dynamic worker pool
    ///
    /// Uses a dynamic pool pattern:
    /// - Starts up to `concurrency` uploads simultaneously
    /// - When a chunk completes, immediately starts the next pending chunk
    /// - Stops all uploads on first error
    async fn upload_chunks_with_pool(
        &self,
        local_path: &Path,
        session: &mut UploadSession,
        pending_chunks: &[usize],
        encryption: Option<EncryptionConfig>,
        tracker: &Arc<ProgressTracker>,
        cancel_token: &CancellationToken,
        concurrency: usize,
    ) -> Result<()> {
        // Prepare chunk infos for pending chunks
        let chunk_infos: Vec<ChunkInfo> = pending_chunks
            .iter()
            .map(|&index| {
                let (offset, _end) = session.chunk_range(index);
                let chunk_size = session.chunk_size_for(index);
                ChunkInfo::new(index, offset, chunk_size)
            })
            .collect();

        // Clone session for read-only access in spawned tasks
        let shared_session = Arc::new(session.clone());

        // Shared state for the dynamic pool
        let pool_state = Arc::new(UploadPoolState::new(chunk_infos));

        // Shared progress state (protected by mutex)
        let progress_state = Arc::new(Mutex::new(ChunkProgressState {
            chunk_progress: session.chunk_progress.clone(),
            updated_at: session.updated_at,
        }));

        // Local path needs to be owned for spawned tasks
        let local_path = local_path.to_path_buf();

        // Spawn initial workers up to concurrency limit
        let mut handles = Vec::with_capacity(concurrency);

        for _ in 0..concurrency {
            if let Some(chunk) = pool_state.next_chunk() {
                pool_state.worker_started();
                let handle = self.spawn_chunk_worker(
                    local_path.clone(),
                    chunk,
                    encryption.clone(),
                    Arc::clone(tracker),
                    cancel_token.clone(),
                    Arc::clone(&pool_state),
                    Arc::clone(&progress_state),
                    Arc::clone(&shared_session),
                );
                handles.push(handle);
            }
        }

        // Wait for all handles to complete
        for handle in handles {
            let _ = handle.await;
        }

        // Wait for any dynamically spawned workers to complete
        pool_state.wait_for_completion().await;

        // Update the session with final progress
        {
            let final_state = progress_state.lock().await;
            session.chunk_progress = final_state.chunk_progress.clone();
            session.updated_at = final_state.updated_at;
        }

        // Check if there was an error
        if let Some(error_msg) = pool_state.get_error() {
            error!(
                target: "uploader::chunk",
                error = %error_msg,
                "Upload failed"
            );
            return Err(anyhow::anyhow!("Upload failed: {}", error_msg));
        }

        info!(
            target: "uploader::chunk",
            "All chunks uploaded successfully"
        );

        Ok(())
    }

    /// Spawn a worker task to upload chunks from the pool
    fn spawn_chunk_worker(
        &self,
        local_path: PathBuf,
        initial_chunk: ChunkInfo,
        encryption: Option<EncryptionConfig>,
        tracker: Arc<ProgressTracker>,
        cancel_token: CancellationToken,
        pool_state: Arc<UploadPoolState>,
        progress_state: Arc<Mutex<ChunkProgressState>>,
        session: Arc<UploadSession>,
    ) -> tokio::task::JoinHandle<()> {
        let http_client = self.http_client.clone();
        let cr_client = Arc::clone(&self.cr_client);
        let policy_type = self.policy_type;
        let config = self.config.clone();

        tokio::spawn(async move {
            // Process chunks in a loop until no more chunks or error
            let mut current_chunk = Some(initial_chunk);

            while let Some(chunk) = current_chunk.take() {
                // Don't start new uploads if there's already an error or cancelled
                if pool_state.has_error() || cancel_token.is_cancelled() {
                    pool_state.worker_done();
                    return;
                }

                let chunk_index = chunk.index;
                tracker.start_chunk();

                debug!(
                    target: "uploader::chunk",
                    chunk = chunk_index,
                    active = pool_state.active_count(),
                    "Starting chunk upload"
                );

                // Upload the chunk with retries
                let result = upload_chunk_with_retry(
                    &http_client,
                    &cr_client,
                    policy_type,
                    &config,
                    &local_path,
                    &chunk,
                    encryption.clone(),
                    &tracker,
                    &cancel_token,
                    &session,
                )
                .await;

                match result {
                    Ok(etag) => {
                        tracker.complete_chunk();

                        // Update progress state
                        {
                            let mut state = progress_state.lock().await;
                            let chunk_size = chunk.size;
                            if chunk_index < state.chunk_progress.len() {
                                state.chunk_progress[chunk_index].loaded = chunk_size;
                                state.chunk_progress[chunk_index].etag = etag;
                                state.updated_at = chrono::Utc::now().timestamp();
                            }
                        }

                        debug!(
                            target: "uploader::chunk",
                            chunk = chunk_index,
                            "Chunk uploaded successfully"
                        );

                        // Get next chunk and continue
                        current_chunk = pool_state.next_chunk();
                    }
                    Err(e) => {
                        error!(
                            target: "uploader::chunk",
                            chunk = chunk_index,
                            error = ?e,
                            "Chunk upload failed, stopping all uploads"
                        );

                        // Mark error state to stop all uploads
                        pool_state.set_error(e.to_string());
                        cancel_token.cancel();
                        pool_state.worker_done();
                        return;
                    }
                }
            }

            pool_state.worker_done();
        })
    }
}

// =========================================================================
// Upload Pool Support Types
// =========================================================================

/// Chunk progress state for uploads (protected by mutex)
#[derive(Debug, Clone)]
struct ChunkProgressState {
    chunk_progress: Vec<ChunkProgress>,
    updated_at: i64,
}

/// State management for chunk uploads with dynamic worker pool
struct UploadPoolState {
    /// Chunks waiting to be uploaded
    pending_chunks: Mutex<Vec<ChunkInfo>>,
    /// Error message if any upload failed
    error: Mutex<Option<String>>,
    /// Flag indicating an error occurred
    has_error: AtomicBool,
    /// Number of active workers
    active_workers: AtomicUsize,
    /// Notify when all workers are done
    all_done: Notify,
}

impl UploadPoolState {
    fn new(chunks: Vec<ChunkInfo>) -> Self {
        Self {
            pending_chunks: Mutex::new(chunks),
            error: Mutex::new(None),
            has_error: AtomicBool::new(false),
            active_workers: AtomicUsize::new(0),
            all_done: Notify::new(),
        }
    }

    /// Increment active worker count - call before spawning a worker
    fn worker_started(&self) {
        self.active_workers.fetch_add(1, Ordering::SeqCst);
    }

    /// Get the next chunk to upload, if any
    fn next_chunk(&self) -> Option<ChunkInfo> {
        // Use try_lock to avoid blocking in async context
        // If we can't get the lock, another worker is getting a chunk
        if let Ok(mut chunks) = self.pending_chunks.try_lock() {
            if !chunks.is_empty() {
                return Some(chunks.remove(0));
            }
        }
        None
    }

    /// Check if an error has occurred
    fn has_error(&self) -> bool {
        self.has_error.load(Ordering::SeqCst)
    }

    /// Set error state
    fn set_error(&self, msg: String) {
        self.has_error.store(true, Ordering::SeqCst);
        if let Ok(mut error) = self.error.try_lock() {
            if error.is_none() {
                *error = Some(msg);
            }
        }
    }

    /// Get the error message if any
    fn get_error(&self) -> Option<String> {
        self.error.try_lock().ok().and_then(|e| e.clone())
    }

    /// Get current active worker count
    fn active_count(&self) -> usize {
        self.active_workers.load(Ordering::SeqCst)
    }

    /// Mark a worker as done
    fn worker_done(&self) {
        let prev = self.active_workers.fetch_sub(1, Ordering::SeqCst);
        if prev == 1 {
            // Last worker, notify waiters
            self.all_done.notify_waiters();
        }
    }

    /// Wait for all workers to complete
    async fn wait_for_completion(&self) {
        loop {
            // Register for notification BEFORE checking condition
            // This prevents missing a notification that happens between check and await
            let notified = self.all_done.notified();

            if self.active_workers.load(Ordering::SeqCst) == 0 {
                return;
            }

            // Now safe to await - if notification happened after our check,
            // we'll still receive it since we registered first
            notified.await;
        }
    }
}

/// Upload a single chunk with retry logic
async fn upload_chunk_with_retry(
    http_client: &HttpClient,
    cr_client: &Arc<CrClient>,
    policy_type: PolicyType,
    config: &UploaderConfig,
    local_path: &Path,
    chunk: &ChunkInfo,
    encryption: Option<EncryptionConfig>,
    tracker: &Arc<ProgressTracker>,
    cancel_token: &CancellationToken,
    session: &Arc<UploadSession>,
) -> Result<Option<String>> {
    for attempt in 0..=config.max_retries {
        if cancel_token.is_cancelled() {
            return Err(anyhow::anyhow!("Upload cancelled"));
        }

        if attempt > 0 {
            let base = config.retry_base_delay.as_millis() as u64;
            let delay_ms = base * (1 << attempt.min(10));
            let delay = Duration::from_millis(delay_ms).min(config.retry_max_delay);

            debug!(
                target: "uploader::chunk",
                chunk = chunk.index,
                attempt,
                delay_ms = delay.as_millis(),
                "Retrying chunk upload"
            );

            tokio::select! {
                _ = tokio::time::sleep(delay) => {}
                _ = cancel_token.cancelled() => {
                    return Err(anyhow::anyhow!("Upload cancelled during retry delay"));
                }
            }
        }

        // Create a fresh stream for each attempt
        let inner_stream = ChunkStream::from_chunk(local_path, chunk, encryption.clone())
            .await
            .map_err(|e| UploadError::FileReadError(format!("Failed to create stream: {}", e)))?;

        // Wrap with progress tracking
        let progress_stream = ProgressStream::new(inner_stream, Arc::clone(tracker));
        // Capture bytes counter before stream is consumed
        let bytes_sent_counter = progress_stream.bytes_sent_counter();

        match providers::upload_chunk_with_progress(
            http_client,
            cr_client,
            policy_type,
            chunk,
            progress_stream,
            session.as_ref(),
        )
        .await
        {
            Ok(etag) => {
                debug!(
                    target: "uploader::chunk",
                    chunk = chunk.index,
                    etag = ?etag,
                    "Chunk uploaded successfully"
                );
                return Ok(etag);
            }
            Err(e) => {
                // Use the captured counter to get bytes sent after stream was consumed
                let bytes_sent = bytes_sent_counter.load(Ordering::SeqCst);
                tracker.reset_chunk_bytes(bytes_sent);
                if attempt == config.max_retries {
                    error!(
                        target: "uploader::chunk",
                        chunk = chunk.index,
                        error = ?e,
                        attempt,
                        "Chunk upload failed after retries"
                    );
                    return Err(e);
                }
                warn!(
                    target: "uploader::chunk",
                    chunk = chunk.index,
                    error = ?e,
                    attempt,
                    "Chunk upload failed, will retry"
                );
            }
        }
    }

    Err(anyhow::anyhow!("Chunk upload failed, max retries exceeded"))
}
