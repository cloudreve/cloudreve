use crate::{
    cfapi::{
        metadata::Metadata,
        placeholder::{LocalFileInfo, PinState},
        placeholder_file::PlaceholderFile,
    },
    drive::{
        mounts::Mount,
        placeholder::CrPlaceholder,
        utils::{local_path_to_cr_uri, remote_path_to_local_relative_path},
    },
    inventory::{ConflictState, FileMetadata, MetadataEntry},
    tasks::TaskPayload,
};
use anyhow::{Context, Result};
use chrono::{DateTime, Utc};
use cloudreve_api::{
    ApiError,
    api::explorer::ExplorerApiExt,
    error::ErrorCode,
    models::{
        explorer::{FileResponse, file_type, metadata},
        uri::CrUri,
    },
};
use md5::Md5;
use notify_debouncer_full::notify::event::{
    AccessKind, CreateKind, EventKind, ModifyKind, RemoveKind, RenameMode,
};
use notify_debouncer_full::{DebouncedEvent, notify::Event};
use nt_time::FileTime;
use sha1::Sha1;
use sha2::{Digest, Sha256};
use std::{
    collections::{HashMap, HashSet},
    ffi::OsString,
    fmt, fs, io,
    io::Read,
    path::{Path, PathBuf},
    time::SystemTime,
};
use tokio::task;
use uuid::Uuid;

pub fn cloud_file_to_placeholder(
    file: &FileResponse,
    _local_path: &PathBuf,
    remote_path: &CrUri,
) -> Result<PlaceholderFile> {
    let file_uri = CrUri::new(&file.path)?;
    let relative_path = remote_path_to_local_relative_path(&file_uri, &remote_path)?;
    tracing::trace!(target: "drive::sync", file_uri = %file_uri.to_string(), remote_path = %remote_path.to_string(), relative_path = %relative_path.to_string_lossy(), "Relative path");
    let primary_entity = OsString::from(file.primary_entity.as_ref().unwrap_or(&String::new()));
    // Remove leading slash if presented

    // Parse RFC time string to unix timestamp
    let created_at =
        FileTime::from_unix_time(file.created_at.parse::<DateTime<Utc>>()?.timestamp())?;
    let last_modified =
        FileTime::from_unix_time(file.updated_at.parse::<DateTime<Utc>>()?.timestamp())?;

    tracing::trace!(target: "drive::sync::cloud_file_to_placeholder", relative_path = %relative_path.to_string_lossy(), "Relative path");

    Ok(PlaceholderFile::new(relative_path)
        .metadata(
            match file.file_type == file_type::FOLDER {
                true => Metadata::directory(),
                false => Metadata::file(),
            }
            .size(file.size as u64)
            .changed(last_modified)
            .written(last_modified)
            .created(created_at),
        )
        .mark_in_sync()
        .overwrite()
        .blob(primary_entity.into_encoded_bytes()))
}

pub fn cloud_file_to_metadata_entry(
    file: &FileResponse,
    drive_id: &Uuid,
    local_path: &PathBuf,
) -> Result<MetadataEntry> {
    let mut local_path = local_path.clone();
    local_path.push(file.name.clone());
    let local_path_str = local_path.to_str();
    if local_path_str.is_none() {
        tracing::error!(
            target: "drive::mounts",
            local_path = %local_path.display(),
            error = "Failed to convert local path to string"
        );
        return Err(anyhow::anyhow!("Failed to convert local path to string"));
    }

    // Parse RFC time string to unix timestamp
    let created_at = file.created_at.parse::<DateTime<Utc>>()?.timestamp();
    let last_modified = file.updated_at.parse::<DateTime<Utc>>()?.timestamp();

    Ok(MetadataEntry::new(
        drive_id.clone(),
        local_path_str.unwrap(),
        file.file_type == file_type::FOLDER,
    )
    .with_created_at(created_at)
    .with_updated_at(last_modified)
    .with_permissions(file.permission.as_ref().unwrap_or(&String::new()).clone())
    .with_shared(file.shared.unwrap_or(false))
    .with_size(file.size)
    .with_etag(
        file.primary_entity
            .as_ref()
            .unwrap_or(&String::new())
            .clone(),
    )
    .with_metadata(file.metadata.as_ref().unwrap_or(&HashMap::new()).clone()))
}

fn cloud_file_to_metadata_entry_at_path(
    file: &FileResponse,
    drive_id: &Uuid,
    local_path: &Path,
    conflict_state: Option<ConflictState>,
) -> Result<MetadataEntry> {
    let local_path_str = local_path
        .to_str()
        .ok_or_else(|| anyhow::anyhow!("Failed to convert local path to string"))?;
    let created_at = file.created_at.parse::<DateTime<Utc>>()?.timestamp();
    let last_modified = file.updated_at.parse::<DateTime<Utc>>()?.timestamp();

    let mut entry = MetadataEntry::new(
        drive_id.clone(),
        local_path_str,
        file.file_type == file_type::FOLDER,
    )
    .with_created_at(created_at)
    .with_updated_at(last_modified)
    .with_permissions(file.permission.as_ref().unwrap_or(&String::new()).clone())
    .with_shared(file.shared.unwrap_or(false))
    .with_size(file.size)
    .with_etag(
        file.primary_entity
            .as_ref()
            .unwrap_or(&String::new())
            .clone(),
    )
    .with_metadata(file.metadata.as_ref().unwrap_or(&HashMap::new()).clone());
    entry.conflict_state = conflict_state;
    Ok(entry)
}

pub fn is_symbolic_link(file: &FileResponse) -> bool {
    return file.metadata.is_some()
        && file
            .metadata
            .as_ref()
            .unwrap()
            .get(metadata::SHARE_REDIRECT)
            .is_some();
}

pub type GroupedFsEvents = HashMap<EventKind, Vec<Event>>;

const REMOTE_PAGE_SIZE: i32 = 1000;

/// Groups filesystem events by their first-level EventKind.
///
/// This function groups events into a HashMap where the key is the first-level EventKind
/// (normalized to use ::Any for nested variants) and the value is a vector of events.
///
/// # Arguments
/// * `events` - A vector of DebouncedEvent to be grouped
///
/// # Returns
/// A HashMap mapping EventKind to Vec<DebouncedEvent>
pub fn group_fs_events(events: Vec<DebouncedEvent>) -> GroupedFsEvents {
    let mut grouped: GroupedFsEvents = HashMap::new();

    for event in events {
        let normalized_kind = normalize_event_kind(&event.kind);
        grouped
            .entry(normalized_kind)
            .or_insert_with(Vec::new)
            .push(event.event);
    }

    grouped
}

/// Normalizes an EventKind to its first-level representation.
///
/// This helper function converts all nested EventKind variants to use their ::Any variant,
/// effectively grouping by the first level only. This can be extended to support deeper
/// level matching by adding parameters for match depth or specific variant matching.
///
/// # Arguments
/// * `kind` - The EventKind to normalize
///
/// # Returns
/// A normalized EventKind representing the first level only
fn normalize_event_kind(kind: &EventKind) -> EventKind {
    match kind {
        EventKind::Any => EventKind::Any,
        EventKind::Access(_) => EventKind::Access(AccessKind::Any),
        EventKind::Create(_) => EventKind::Create(CreateKind::Any),
        EventKind::Modify(modify_kind) => match modify_kind {
            ModifyKind::Name(rename_mode) => match rename_mode {
                RenameMode::Both => EventKind::Modify(ModifyKind::Name(RenameMode::Both)),
                _ => EventKind::Modify(ModifyKind::Any),
            },
            _ => EventKind::Modify(ModifyKind::Any),
        },
        EventKind::Remove(_) => EventKind::Remove(RemoveKind::Any),
        EventKind::Other => EventKind::Other,
    }
}

/// Determines how deep a sync operation should traverse for a given path list.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum SyncMode {
    /// Sync only the provided path entries.
    PathOnly,
    /// Sync the provided path entries and their first-level children.
    PathAndFirstLayer,
    /// Sync the provided path entries and every descendant.
    FullHierarchy,
}

const CONFLICT_PREFIX: &str = "__conflict__";

#[allow(dead_code)]
#[derive(Debug, Clone)]
enum SyncAction {
    CreatePlaceholderAndInventory {
        path: PathBuf,
        remote: FileResponse,
    },
    RecordInventoryFromRemote {
        path: PathBuf,
        remote: FileResponse,
        mark_conflicted: bool,
    },
    // Update inventory and placehodler metadata, conver to placehodler if it's not one
    UpdateInventoryFromRemote {
        path: PathBuf,
        remote: FileResponse,
        invalidate_all: bool,
    },
    QueueUpload {
        path: PathBuf,
        reason: UploadReason,
    },
    QueueDownload {
        path: PathBuf,
        remote: FileResponse,
    },
    DeleteLocalAndInventory {
        path: PathBuf,
        skip_if_not_empty: bool,
    },
    CreateRemoteFolderIfExist {
        path: PathBuf,
    },
    RenameLocalWithConflict {
        original: PathBuf,
        renamed: PathBuf,
    },
}

#[derive(Debug, Clone, Copy)]
enum UploadReason {
    RemoteMismatch,
    RemoteMissing,
}

#[derive(Debug, Clone, Copy)]
enum WalkReason {
    ModePropagation,
    DiffTriggered,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum WalkTiming {
    Immediate,
    Deferred,
}

#[derive(Debug, Clone)]
struct WalkRequest {
    path: PathBuf,
    mode: SyncMode,
    reason: WalkReason,
    timing: WalkTiming,
}

#[derive(Default)]
struct SyncPlan {
    actions: Vec<SyncAction>,
    walk_requests: Vec<WalkRequest>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) enum HashAlgorithm {
    Md5,
    Sha1,
    Sha256,
    Sha512,
}

impl HashAlgorithm {
    fn from_hex_len(len: usize) -> Option<Self> {
        match len {
            32 => Some(Self::Md5),
            40 => Some(Self::Sha1),
            64 => Some(Self::Sha256),
            128 => Some(Self::Sha512),
            _ => None,
        }
    }

    pub(crate) fn as_str(self) -> &'static str {
        match self {
            Self::Md5 => "md5",
            Self::Sha1 => "sha1",
            Self::Sha256 => "sha256",
            Self::Sha512 => "sha512",
        }
    }
}

#[derive(Debug, Clone)]
pub(crate) struct FileHashFingerprint {
    pub(crate) algorithm: HashAlgorithm,
    pub(crate) value: String,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum ExistingRemoteLocalFileDecision {
    SameContent,
    Conflict,
}

// Debug print for SyncPlan
impl fmt::Debug for SyncPlan {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        writeln!(
            f,
            "SyncPlan ({} actions, {} walks):",
            self.actions.len(),
            self.walk_requests.len()
        )?;

        for (i, action) in self.actions.iter().enumerate() {
            writeln!(f, "  [{}] {:?}", i, action)?;
        }

        for (i, walk) in self.walk_requests.iter().enumerate() {
            writeln!(f, "  [W{}] {:?}", i, walk)?;
        }

        Ok(())
    }
}

#[derive(Debug)]
struct SyncErrorEntry {
    path: PathBuf,
    error: anyhow::Error,
}

#[derive(Debug)]
struct SyncAggregateError {
    context: String,
    entries: Vec<SyncErrorEntry>,
}

impl SyncAggregateError {
    fn new(context: impl Into<String>) -> Self {
        Self {
            context: context.into(),
            entries: Vec::new(),
        }
    }

    fn push<E>(&mut self, path: PathBuf, error: E)
    where
        E: Into<anyhow::Error>,
    {
        self.entries.push(SyncErrorEntry {
            path,
            error: error.into(),
        });
    }

    fn is_empty(&self) -> bool {
        self.entries.is_empty()
    }

    fn into_result(self) -> Result<()> {
        if self.is_empty() {
            Ok(())
        } else {
            Err(self.into())
        }
    }
}

impl fmt::Display for SyncAggregateError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        writeln!(
            f,
            "{} encountered {} error(s):",
            self.context,
            self.entries.len()
        )?;
        for entry in &self.entries {
            writeln!(f, "- {}: {}", entry.path.display(), entry.error)?;
        }
        Ok(())
    }
}

impl std::error::Error for SyncAggregateError {}

/// Parses an RFC3339 timestamp (as returned by the Cloudreve server) into a
/// unix timestamp in seconds. Returns `None` when the value cannot be parsed
/// instead of silently degrading to the epoch.
fn parse_rfc3339_timestamp(value: &str) -> Option<i64> {
    DateTime::parse_from_rfc3339(value)
        .ok()
        .map(|datetime| datetime.timestamp())
}

fn system_time_to_unix_millis(time: SystemTime) -> Option<i64> {
    time.duration_since(SystemTime::UNIX_EPOCH)
        .ok()
        .map(|duration| duration.as_millis() as i64)
}

/// Returns true when the current on-disk state of a local file differs from
/// the snapshot recorded at the last successful sync point.
///
/// The snapshot is only meaningful when it exists; rows written before
/// snapshots were introduced return `false` so the caller falls back to the
/// CFAPI IN_SYNC flag.
pub(crate) fn local_snapshot_differs(entry: &FileMetadata, local: &LocalFileInfo) -> bool {
    let (Some(snapshot_mtime), Some(snapshot_size)) = (entry.local_updated_at, entry.local_size)
    else {
        return false;
    };

    if local
        .file_size
        .is_some_and(|local_size| local_size as i64 != snapshot_size)
    {
        return true;
    }

    if local
        .last_modified
        .and_then(system_time_to_unix_millis)
        .is_some_and(|mtime| mtime != snapshot_mtime)
    {
        return true;
    }

    false
}

/// Decides whether the local file carries changes that are not reflected in
/// the last synced (base) state, using the CFAPI IN_SYNC flag as the primary
/// signal and the recorded local snapshot as a cross-check.
fn local_has_changes(local: &LocalFileInfo, inventory: Option<&FileMetadata>) -> bool {
    if !local.is_placeholder() {
        // On platforms without placeholder support (non-Windows full sync)
        // the inventory snapshot is the only signal available.
        #[cfg(not(windows))]
        if let Some(entry) = inventory
            && entry.local_updated_at.is_some()
            && entry.local_size.is_some()
        {
            return local_snapshot_differs(entry, local);
        }
        // Windows: a non-placeholder file is always treated as changed so it
        // gets converted to a placeholder and synced.
        return true;
    }

    // Windows placeholder: the IN_SYNC flag is authoritative for detecting
    // writes (Windows clears it on any modification). Trust a cleared flag
    // even when the snapshot still matches - writes that restore the original
    // mtime/size are possible.
    if !local.in_sync() {
        return true;
    }

    // Flag says in-sync: cross-check with the snapshot to catch races where
    // the flag was (re)set while local edits existed. Rows written before
    // snapshots existed fall back to trusting the flag.
    matches!(
        inventory,
        Some(entry)
            if entry.local_updated_at.is_some()
                && entry.local_size.is_some()
                && local_snapshot_differs(entry, local)
    )
}

/// How the current remote state compares to the last synced base version.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum RemoteChange {
    /// Remote metadata matches the base version.
    Unchanged,
    /// Remote differs from base and its timestamp did not go backwards.
    Newer,
    /// Remote differs from base and its timestamp is older than the base one.
    /// This is suspicious (stale listing data or a server-side rollback) and
    /// must not silently overwrite local content.
    Older,
}

/// Compares the remote file state against the inventory base state.
///
/// The etag is the primary change signal. When it matches, the entry is
/// considered unchanged even if the timestamps differ - providers can rewrite
/// etags or timestamps without touching content, and a parse failure must
/// never be mistaken for a remote change (which previously caused the remote
/// version to overwrite and dehydrate local content on every sync).
fn remote_change(remote: &FileResponse, inventory: Option<&FileMetadata>) -> RemoteChange {
    let Some(entry) = inventory else {
        return RemoteChange::Newer;
    };

    let remote_etag = remote.primary_entity.as_deref().unwrap_or("");
    if remote_etag != entry.etag {
        let remote_updated_at = parse_rfc3339_timestamp(&remote.updated_at);
        return match remote_updated_at {
            Some(remote_ts) if remote_ts < entry.updated_at => RemoteChange::Older,
            _ => RemoteChange::Newer,
        };
    }

    // Etag matches. Some providers publish no etag at all; fall back to the
    // updated_at field so content-only changes are still noticed.
    if remote_etag.is_empty()
        && let Some(remote_updated_at) = parse_rfc3339_timestamp(&remote.updated_at)
        && remote_updated_at != entry.updated_at
    {
        return if remote_updated_at < entry.updated_at {
            RemoteChange::Older
        } else {
            RemoteChange::Newer
        };
    }

    RemoteChange::Unchanged
}

/// Returns true when both fingerprints are known and carry the same digest.
fn fingerprints_equal(
    base: Option<&FileHashFingerprint>,
    remote: Option<&FileHashFingerprint>,
) -> bool {
    match (base, remote) {
        (Some(base), Some(remote)) => base.value == remote.value,
        _ => false,
    }
}

fn generate_conflict_path(path: &Path) -> PathBuf {
    let timestamp = Utc::now().format("%Y%m%d%H%M%S");
    let stem = path
        .file_stem()
        .and_then(|value| value.to_str())
        .unwrap_or("item");
    let ext = path.extension().and_then(|value| value.to_str());
    let mut new_name = format!("{}{}_{}", CONFLICT_PREFIX, timestamp, stem);
    if let Some(ext) = ext {
        new_name.push('.');
        new_name.push_str(ext);
    }
    let mut conflict_path = path.to_path_buf();
    conflict_path.set_file_name(new_name);
    conflict_path
}

fn next_child_mode(mode: SyncMode) -> SyncMode {
    match mode {
        SyncMode::FullHierarchy => SyncMode::FullHierarchy,
        SyncMode::PathAndFirstLayer => SyncMode::PathOnly,
        SyncMode::PathOnly => SyncMode::PathOnly,
    }
}

fn normalize_hash_value(value: &str) -> Option<String> {
    let normalized = value
        .trim()
        .trim_matches('"')
        .to_ascii_lowercase()
        .replace('-', "");
    if normalized.chars().all(|ch| ch.is_ascii_hexdigit()) {
        Some(normalized)
    } else {
        None
    }
}

fn hash_fingerprint_from_value(value: &str) -> Option<FileHashFingerprint> {
    let normalized = normalize_hash_value(value)?;
    let algorithm = HashAlgorithm::from_hex_len(normalized.len())?;
    Some(FileHashFingerprint {
        algorithm,
        value: normalized,
    })
}

fn normalize_hash_key(key: &str) -> String {
    key.chars()
        .filter(|ch| ch.is_ascii_alphanumeric())
        .collect::<String>()
        .to_ascii_lowercase()
}

const HASH_METADATA_KEYS: &[(&str, HashAlgorithm)] = &[
    ("md5", HashAlgorithm::Md5),
    ("hashmd5", HashAlgorithm::Md5),
    ("checksummd5", HashAlgorithm::Md5),
    ("sha1", HashAlgorithm::Sha1),
    ("hashsha1", HashAlgorithm::Sha1),
    ("checksumsha1", HashAlgorithm::Sha1),
    ("sha256", HashAlgorithm::Sha256),
    ("hashsha256", HashAlgorithm::Sha256),
    ("checksumsha256", HashAlgorithm::Sha256),
    ("sha512", HashAlgorithm::Sha512),
    ("hashsha512", HashAlgorithm::Sha512),
    ("checksumsha512", HashAlgorithm::Sha512),
];

/// Extracts a content-hash fingerprint from a Cloudreve metadata map.
pub(crate) fn metadata_hash_fingerprint(
    metadata: &HashMap<String, String>,
) -> Option<FileHashFingerprint> {
    for (key, value) in metadata {
        let normalized_key = normalize_hash_key(key);

        // Exact key match (e.g. "md5", "sha256").
        for (hash_key, algorithm) in HASH_METADATA_KEYS {
            if normalized_key == *hash_key {
                if let Some(value) = normalize_hash_value(value) {
                    return Some(FileHashFingerprint {
                        algorithm: *algorithm,
                        value,
                    });
                }
            }
        }

        // Generic keys like "hash" or "checksum" where the algorithm is
        // inferred from the digest length.
        if normalized_key == "hash" || normalized_key == "checksum" {
            if let Some(fingerprint) = hash_fingerprint_from_value(value) {
                return Some(fingerprint);
            }
        }
    }

    None
}

pub(crate) fn remote_file_hash_fingerprint(remote: &FileResponse) -> Option<FileHashFingerprint> {
    // Cloudreve does not expose a dedicated typed content-hash field in
    // FileResponse. Prefer explicit metadata keys first so the comparison uses
    // whatever hash algorithm the server actually publishes.
    //
    // primary_entity is used as an opaque etag/entity identifier elsewhere
    // (see cloud_file_to_metadata_entry and CrPlaceholder::with_remote_file).
    // It is not a reliable content hash, so do not use it for equality checks;
    // falling back to size comparison avoids false conflicts when the remote
    // does not publish a real content hash.
    metadata_hash_fingerprint(remote.metadata.as_ref().unwrap_or(&HashMap::new()))
}

pub(crate) async fn calculate_file_hash(path: PathBuf, algorithm: HashAlgorithm) -> Result<String> {
    task::spawn_blocking(move || -> Result<String> {
        let mut file = fs::File::open(&path)
            .with_context(|| format!("failed to open file {}", path.display()))?;
        let mut buffer = [0u8; 64 * 1024];

        match algorithm {
            HashAlgorithm::Md5 => {
                let mut hasher = Md5::new();
                loop {
                    let read = file.read(&mut buffer)?;
                    if read == 0 {
                        break;
                    }
                    hasher.update(&buffer[..read]);
                }
                Ok(format!("{:x}", hasher.finalize()))
            }
            HashAlgorithm::Sha1 => {
                let mut hasher = Sha1::new();
                loop {
                    let read = file.read(&mut buffer)?;
                    if read == 0 {
                        break;
                    }
                    hasher.update(&buffer[..read]);
                }
                Ok(format!("{:x}", hasher.finalize()))
            }
            HashAlgorithm::Sha256 => {
                let mut hasher = Sha256::new();
                loop {
                    let read = file.read(&mut buffer)?;
                    if read == 0 {
                        break;
                    }
                    hasher.update(&buffer[..read]);
                }
                Ok(format!("{:x}", hasher.finalize()))
            }
            HashAlgorithm::Sha512 => {
                let mut hasher = sha2::Sha512::new();
                loop {
                    let read = file.read(&mut buffer)?;
                    if read == 0 {
                        break;
                    }
                    hasher.update(&buffer[..read]);
                }
                Ok(format!("{:x}", hasher.finalize()))
            }
        }
    })
    .await?
}

/// Result of collecting child targets, including pre-fetched remote file info.
struct CollectChildResult {
    /// All child paths (union of local and remote).
    paths: Vec<PathBuf>,
    /// Pre-fetched remote file info keyed by local path.
    remote_files: HashMap<PathBuf, FileResponse>,
}

impl Mount {
    /// Syncs a list of local paths by grouping them under their parent directories.
    pub async fn sync_paths(&self, local_paths: Vec<PathBuf>, mode: SyncMode) -> Result<()> {
        let _sync_guard = self.sync_lock.lock().await;

        if local_paths.is_empty() {
            tracing::debug!(target: "drive::sync", id = %self.id, "No paths provided for sync");
            return Ok(());
        }

        let mut grouped: HashMap<PathBuf, Vec<PathBuf>> = HashMap::new();

        for path in local_paths {
            let parent = path
                .parent()
                .map(|p| p.to_path_buf())
                .unwrap_or_else(|| path.clone());
            grouped.entry(parent).or_default().push(path);
        }

        let mut aggregate_error = SyncAggregateError::new(format!("Mount {} sync_paths", self.id));

        for (parent, paths) in grouped.iter() {
            if let Err(err) = self.sync_group(parent, paths, mode, None).await {
                let target_path = paths.first().cloned().unwrap_or_else(|| parent.clone());
                aggregate_error.push(target_path, err);
            }
        }

        drop(_sync_guard);
        aggregate_error.into_result()
    }

    async fn sync_group(
        &self,
        parent: &PathBuf,
        paths: &[PathBuf],
        mode: SyncMode,
        prefetched_remote_files: Option<HashMap<PathBuf, FileResponse>>,
    ) -> Result<()> {
        tracing::info!(
            target: "drive::sync",
            id = %self.id,
            parent = %parent.display(),
            paths = paths.len(),
            mode = ?mode,
            prefetched = prefetched_remote_files.is_some(),
            "Queued grouped sync"
        );

        let mut aggregate_error = SyncAggregateError::new(format!(
            "Mount {} sync_group({})",
            self.id,
            parent.display()
        ));

        // For sync root, directly walk to descendants
        let sync_root = {
            let config = self.config.read().await;
            config.data_root()
        };
        if paths.len() == 1 && paths[0] == sync_root {
            tracing::debug!(
                target: "drive::sync",
                id = %self.id,
                parent = %parent.display(),
                "Syncing sync root"
            );
            self.process_walk_requests(
                vec![WalkRequest {
                    path: sync_root,
                    mode,
                    reason: WalkReason::ModePropagation,
                    timing: WalkTiming::Immediate,
                }],
                &mut aggregate_error,
            )
            .await;
            return aggregate_error.into_result();
        }

        let remote_files = match prefetched_remote_files {
            Some(files) => files,
            None => self.fetch_remote_file_infos(parent, paths).await?,
        };
        tracing::debug!(
            target: "drive::sync",
            id = %self.id,
            parent = %parent.display(),
            requested = paths.len(),
            fetched = remote_files.len(),
            "Fetched remote metadata for sync group"
        );
        tracing::trace!("{:?}", remote_files);

        let local_files = self.fetch_local_file_infos(paths).await?;
        tracing::debug!(
            target: "drive::sync",
            id = %self.id,
            parent = %parent.display(),
            locals = local_files.len(),
            "Fetched local metadata for sync group"
        );
        tracing::trace!("{:?}", local_files);

        let inventory_files = self.fetch_inventory_entries(paths).await?;
        tracing::trace!("{:?}", inventory_files);

        let plan = self
            .build_sync_plan(
                parent,
                mode,
                paths,
                &remote_files,
                &local_files,
                &inventory_files,
            )
            .await;

        tracing::debug!(
            target: "drive::sync",
            id = %self.id,
            parent = %parent.display(),
            actions = plan.actions.len(),
            walks = plan.walk_requests.len(),
            "Planned sync actions"
        );
        tracing::trace!(target: "drive::sync", plan = ?plan, "Planned actions detail");

        let SyncPlan {
            actions,
            walk_requests,
        } = plan;
        let (immediate_walks, deferred_walks): (Vec<_>, Vec<_>) = walk_requests
            .into_iter()
            .partition(|request| request.timing == WalkTiming::Immediate);

        self.process_walk_requests(immediate_walks, &mut aggregate_error)
            .await;

        if let Err(err) = self
            .process_sync_plan_actions_list(&actions, &mut aggregate_error)
            .await
        {
            aggregate_error.push(parent.clone(), err);
        }

        self.process_walk_requests(deferred_walks, &mut aggregate_error)
            .await;
        aggregate_error.into_result()
    }

    async fn process_sync_plan_actions_list(
        &self,
        actions: &[SyncAction],
        aggregate_error: &mut SyncAggregateError,
    ) -> Result<()> {
        let (drive_id, sync_root) = {
            let config = self.config.read().await;
            (Uuid::parse_str(&config.id)?, config.data_root())
        };

        for action in actions {
            self.process_action(action, &sync_root, &drive_id, aggregate_error)
                .await;
        }

        Ok(())
    }

    async fn process_action(
        &self,
        action: &SyncAction,
        sync_root: &PathBuf,
        drive_id: &Uuid,
        aggregate_error: &mut SyncAggregateError,
    ) {
        match action {
            SyncAction::CreatePlaceholderAndInventory { path, remote } => {
                let cr_placeholder =
                    CrPlaceholder::new(path.clone(), sync_root.clone(), drive_id.clone());
                if let Err(err) = cr_placeholder
                    .with_remote_file(remote)
                    .commit(self.inventory.clone())
                {
                    tracing::error!(
                        target: "drive::sync",
                        id = %self.id,
                        path = %path.display(),
                        error = ?err,
                        "Failed to create placeholder and inventory"
                    );
                    aggregate_error.push(path.clone(), err);
                } else {
                    #[cfg(not(windows))]
                    if remote.file_type != file_type::FOLDER
                        && !self.config.read().await.is_ondemand()
                    {
                        // Non-Windows `CrPlaceholder` only records inventory metadata here.
                        // There is no CFAPI-style dehydrated placeholder, so queue a normal
                        // download to create the real full-sync file on disk. On-demand
                        // (FUSE) drives skip this — bytes arrive on first open.
                        if let Err(err) = self
                            .task_queue
                            .enqueue(TaskPayload::download(path.clone()))
                            .await
                        {
                            aggregate_error.push(path.clone(), anyhow::Error::from(err));
                        }
                    }
                }
            }
            SyncAction::RecordInventoryFromRemote {
                path,
                remote,
                mark_conflicted,
            } => {
                if *mark_conflicted {
                    match cloud_file_to_metadata_entry_at_path(
                        remote,
                        drive_id,
                        path,
                        Some(ConflictState::Pending),
                    )
                    .and_then(|entry| {
                        self.inventory
                            .upsert(&entry)
                            .context("failed to upsert conflicted inventory metadata")
                    }) {
                        Ok(_) => {}
                        Err(err) => {
                            tracing::error!(
                                target: "drive::sync",
                                id = %self.id,
                                path = %path.display(),
                                error = ?err,
                                "Failed to record existing local file conflict"
                            );
                            aggregate_error.push(path.clone(), err);
                        }
                    }
                    return;
                }

                let cr_placeholder =
                    CrPlaceholder::new(path.clone(), sync_root.clone(), drive_id.clone());
                if let Err(err) = cr_placeholder
                    .with_remote_file(remote)
                    .commit(self.inventory.clone())
                {
                    tracing::error!(
                        target: "drive::sync",
                        id = %self.id,
                        path = %path.display(),
                        error = ?err,
                        "Failed to record existing local file inventory"
                    );
                    aggregate_error.push(path.clone(), err);
                }
            }
            SyncAction::UpdateInventoryFromRemote {
                path,
                remote,
                invalidate_all,
            } => {
                let cr_placeholder =
                    CrPlaceholder::new(path.clone(), sync_root.clone(), drive_id.clone());
                if let Err(err) = cr_placeholder
                    .with_invalidate_all_range(*invalidate_all)
                    .with_remote_file(remote)
                    .commit(self.inventory.clone())
                {
                    tracing::error!(
                        target: "drive::sync",
                        id = %self.id,
                        path = %path.display(),
                        error = ?err,
                        "Failed to update inventory from remote"
                    );
                    aggregate_error.push(path.clone(), err);
                } else {
                    #[cfg(not(windows))]
                    if remote.file_type != file_type::FOLDER
                        && !self.config.read().await.is_ondemand()
                    {
                        // Non-Windows `CrPlaceholder` only refreshes inventory metadata here.
                        // Full-sync drives fetch contents immediately through the task
                        // queue; on-demand (FUSE) drives defer bytes to first open.
                        if let Err(err) = self
                            .task_queue
                            .enqueue(TaskPayload::download(path.clone()))
                            .await
                        {
                            aggregate_error.push(path.clone(), anyhow::Error::from(err));
                        }
                    }
                }
            }
            SyncAction::QueueUpload { path, reason } => {
                tracing::info!(
                    target: "drive::sync",
                    id = %self.id,
                    path = %path.display(),
                    reason = ?reason,
                    "Queueing upload task"
                );

                if let Err(err) = self
                    .task_queue
                    .enqueue(TaskPayload::upload(path.clone()))
                    .await
                {
                    tracing::error!(
                        target: "drive::sync",
                        id = %self.id,
                        path = %path.display(),
                        error = ?err,
                        "Failed to enqueue upload task"
                    );
                    aggregate_error.push(path.clone(), anyhow::Error::from(err));
                }
            }
            SyncAction::QueueDownload { path, remote: _ } => {
                tracing::info!(
                    target: "drive::sync",
                    id = %self.id,
                    path = %path.display(),
                    "Queueing download task"
                );

                // Cancel ongoing tasks
                let _ = self.task_queue.cancel_by_path(path.clone()).await;

                // On-demand drives never fetch eagerly: dropping the hydrated
                // copy forces re-hydration on next open, which is the FUSE
                // equivalent of re-downloading a changed remote file.
                if self.config.read().await.is_ondemand() {
                    if let Ok(md) = std::fs::symlink_metadata(path) {
                        // Suppress the watcher Remove the eviction is about to
                        // emit — otherwise it propagates as a remote delete of
                        // the very file we want to re-fetch.
                        if md.is_dir() {
                            // A stale directory at a file's path must go too —
                            // otherwise it permanently shadows re-hydration.
                            self.event_blocker.register_prefix(
                                &EventKind::Remove(RemoveKind::Any),
                                path.clone(),
                                std::time::Duration::from_secs(15),
                            );
                        } else {
                            self.event_blocker.register_once(
                                &EventKind::Remove(RemoveKind::Any),
                                path.clone(),
                            );
                        }
                        let res = if md.is_dir() {
                            std::fs::remove_dir_all(path)
                        } else {
                            std::fs::remove_file(path)
                        };
                        if let Err(err) = res {
                            tracing::warn!(
                                target: "drive::sync",
                                id = %self.id,
                                path = %path.display(),
                                error = %err,
                                "Failed to evict hydrated copy for re-fetch"
                            );
                        }
                    }
                    return;
                }

                if let Err(err) = self
                    .task_queue
                    .enqueue(TaskPayload::download(path.clone()))
                    .await
                {
                    tracing::error!(
                        target: "drive::sync",
                        id = %self.id,
                        path = %path.display(),
                        error = ?err,
                        "Failed to enqueue download task"
                    );
                    aggregate_error.push(path.clone(), anyhow::Error::from(err));
                }
            }
            SyncAction::DeleteLocalAndInventory {
                path,
                skip_if_not_empty,
            } => {
                if *skip_if_not_empty {
                    // Check if folder is not empty
                    if let Ok(entries) = std::fs::read_dir(path) {
                        if entries.count() > 0 {
                            tracing::info!(
                                target: "drive::sync",
                                id = %self.id,
                                path = %path.display(),
                                "Folder is not empty, skipping deletion"
                            );
                            return;
                        }
                    }
                }

                tracing::info!(
                    target: "drive::sync",
                    id = %self.id,
                    path = %path.display(),
                    "Deleting local file/folder and inventory entry"
                );

                let cr_placeholder =
                    CrPlaceholder::new(path.clone(), sync_root.clone(), drive_id.clone());
                if let Err(err) = cr_placeholder.delete_placeholder(self.inventory.clone()) {
                    tracing::error!(
                        target: "drive::sync",
                        id = %self.id,
                        path = %path.display(),
                        error = ?err,
                        "Failed to delete local file/folder and inventory entry"
                    );
                    aggregate_error.push(path.clone(), anyhow::Error::from(err));
                };
                self.event_blocker
                    .register_once(&EventKind::Remove(RemoveKind::Any), path.clone());
            }
            SyncAction::CreateRemoteFolderIfExist { path } => {
                if !path.exists() {
                    return;
                }
                tracing::info!(
                    target: "drive::sync",
                    id = %self.id,
                    path = %path.display(),
                    "Creating remote folder"
                );
                if let Err(err) = self
                    .task_queue
                    .enqueue(TaskPayload::upload(path.clone()))
                    .await
                {
                    tracing::error!(
                        target: "drive::sync",
                        id = %self.id,
                        path = %path.display(),
                        error = ?err,
                        "Failed to enqueue upload task"
                    );
                    aggregate_error.push(path.clone(), anyhow::Error::from(err));
                }
            }
            SyncAction::RenameLocalWithConflict { original, renamed } => {
                tracing::info!(
                    target: "drive::sync",
                    id = %self.id,
                    original = %original.display(),
                    renamed = %renamed.display(),
                    "Renaming local file to resolve conflict"
                );

                // Cancel tasks for the original path
                _ = self.task_queue.cancel_by_path(original.clone()).await;

                if let Err(err) = std::fs::rename(original, renamed) {
                    tracing::error!(
                        target: "drive::sync",
                        id = %self.id,
                        original = %original.display(),
                        renamed = %renamed.display(),
                        error = ?err,
                        "Failed to rename local file"
                    );
                    aggregate_error.push(original.clone(), anyhow::Error::from(err));
                }
            }
        }
    }

    async fn fetch_local_file_infos(
        &self,
        paths: &[PathBuf],
    ) -> Result<HashMap<PathBuf, LocalFileInfo>> {
        if paths.is_empty() {
            return Ok(HashMap::new());
        }

        let targets: Vec<PathBuf> = paths.to_vec();
        let mut entries = HashMap::with_capacity(targets.len());
        for path in targets {
            let info = LocalFileInfo::from_path(&path)?;
            entries.insert(path, info);
        }

        Ok(entries)
    }

    async fn fetch_remote_file_infos(
        &self,
        parent: &PathBuf,
        paths: &[PathBuf],
    ) -> Result<HashMap<PathBuf, FileResponse>> {
        if paths.is_empty() {
            return Ok(HashMap::new());
        }

        let (remote_base, sync_root) = {
            let config = self.config.read().await;
            (config.remote_path.clone(), config.data_root())
        };

        let mut target_remote_paths: HashMap<String, PathBuf> = HashMap::with_capacity(paths.len());
        for path in paths {
            let remote_uri =
                local_path_to_cr_uri(path.clone(), sync_root.clone(), remote_base.clone())
                    .with_context(|| format!("failed to map {} to remote uri", path.display()))?;
            target_remote_paths.insert(remote_uri.to_string(), path.clone());
        }

        let parent_remote_uri =
            local_path_to_cr_uri(parent.clone(), sync_root.clone(), remote_base.clone())
                .with_context(|| {
                    format!("failed to map parent {} to remote uri", parent.display())
                })?;
        let parent_uri_str = parent_remote_uri.to_string();

        let mut remote_entries: HashMap<PathBuf, FileResponse> =
            HashMap::with_capacity(paths.len());
        let mut remaining: HashSet<String> = target_remote_paths.keys().cloned().collect();
        let mut previous_response = None;

        while !remaining.is_empty() {
            let response = match self
                .cr_client
                .list_files_all(
                    previous_response.as_ref(),
                    parent_uri_str.as_str(),
                    REMOTE_PAGE_SIZE,
                )
                .await
            {
                Ok(resp) => resp,
                Err(ApiError::ApiError { code, .. })
                    if code == ErrorCode::ParentNotExist as i32 =>
                {
                    tracing::debug!(
                        target: "drive::sync",
                        id = %self.id,
                        parent = %parent.display(),
                        "Remote parent directory missing during fetch"
                    );
                    return Ok(HashMap::new());
                }
                Err(err) => {
                    return Err(err.into());
                }
            };

            for file in &response.res.files {
                if let Some(local_path) = target_remote_paths.get(&file.path) {
                    if remote_entries.contains_key(local_path) {
                        continue;
                    }
                    remote_entries.insert(local_path.clone(), file.clone());
                    remaining.remove(&file.path);
                }
            }

            let has_more = response.more && !remaining.is_empty();
            previous_response = Some(response);

            if !has_more {
                break;
            }
        }

        if !remaining.is_empty() {
            for missing in remaining {
                if let Some(local_path) = target_remote_paths.get(&missing) {
                    tracing::warn!(
                        target: "drive::sync",
                        id = %self.id,
                        path = %local_path.display(),
                        remote_path = %missing,
                        "Remote entry missing during sync"
                    );
                }
            }
        }

        Ok(remote_entries)
    }

    async fn fetch_inventory_entries(
        &self,
        paths: &[PathBuf],
    ) -> Result<HashMap<PathBuf, FileMetadata>> {
        if paths.is_empty() {
            return Ok(HashMap::new());
        }

        let mut targets: Vec<(PathBuf, String)> = Vec::with_capacity(paths.len());
        for path in paths {
            match path.to_str() {
                Some(path_str) => targets.push((path.clone(), path_str.to_string())),
                None => {
                    tracing::warn!(
                        target: "drive::sync",
                        id = %self.id,
                        path = %path.display(),
                        "Unable to convert path to UTF-8 for inventory lookup"
                    );
                }
            }
        }

        if targets.is_empty() {
            return Ok(HashMap::new());
        }

        let inventory = self.inventory.clone();
        let entries = task::spawn_blocking(move || -> Result<HashMap<PathBuf, FileMetadata>> {
            let mut results = HashMap::with_capacity(targets.len());
            for (path_buf, path_str) in targets {
                match inventory.query_by_path(&path_str)? {
                    Some(entry) => {
                        results.insert(path_buf, entry);
                    }
                    None => {}
                }
            }
            Ok(results)
        })
        .await??;

        Ok(entries)
    }

    async fn build_sync_plan(
        &self,
        _parent: &PathBuf,
        mode: SyncMode,
        paths: &[PathBuf],
        remote_files: &HashMap<PathBuf, FileResponse>,
        local_files: &HashMap<PathBuf, LocalFileInfo>,
        inventory_entries: &HashMap<PathBuf, FileMetadata>,
    ) -> SyncPlan {
        let mut plan = SyncPlan::default();
        let matcher = self.ignore_matcher.read().await;

        for path in paths {
            if matcher.is_match(path) {
                tracing::trace!(
                    target: "drive::sync",
                    id = %self.id,
                    path = %path.display(),
                    "Skipping ignored path in sync plan"
                );
                continue;
            }
            let local_info = local_files
                .get(path)
                .cloned()
                .unwrap_or_else(LocalFileInfo::missing);
            let remote = remote_files.get(path);
            let inventory = inventory_entries.get(path);
            if let (Some(remote), Some(local_info)) = (remote, local_files.get(path)) {
                // This handles attaching a non-empty local directory to an
                // existing remote tree. With no inventory row yet, a same-name
                // local/remote file must be resolved before normal sync would
                // treat the local file as a fresh upload or the remote file as
                // a placeholder/download target.
                if let Some(decision) = self
                    .decide_existing_local_remote_file(path, remote, local_info, inventory)
                    .await
                {
                    let mark_conflicted = decision == ExistingRemoteLocalFileDecision::Conflict;
                    plan.actions.push(SyncAction::RecordInventoryFromRemote {
                        path: path.clone(),
                        remote: remote.clone(),
                        mark_conflicted,
                    });
                    if mark_conflicted {
                        tracing::info!(
                            target: "drive::sync",
                            id = %self.id,
                            path = %path.display(),
                            "Existing local file conflicts with same-name remote file; queued conflict metadata"
                        );
                    }
                    continue;
                }
            }
            self.plan_entry_actions(path, mode, remote, &local_info, inventory, &mut plan)
                .await;
        }

        plan
    }

    async fn decide_existing_local_remote_file(
        &self,
        path: &PathBuf,
        remote: &FileResponse,
        local: &LocalFileInfo,
        inventory: Option<&FileMetadata>,
    ) -> Option<ExistingRemoteLocalFileDecision> {
        if inventory.is_some()
            || !local.exists
            || local.is_directory
            || local.is_placeholder()
            || remote.file_type == file_type::FOLDER
        {
            return None;
        }

        let Some(remote_fingerprint) = remote_file_hash_fingerprint(remote) else {
            // Remote does not expose a hash. Fall back to size comparison so
            // identical files are not falsely reported as conflicts.
            let local_size = local.file_size.unwrap_or(0);
            let remote_size = remote.size as u64;
            if local_size == remote_size {
                // Sizes match but content equality cannot be proven. If the
                // local file was modified after the remote entry, assume the
                // local version is newer and treat it as a conflict instead of
                // silently adopting the (possibly older) remote version.
                let remote_updated_at = parse_rfc3339_timestamp(&remote.updated_at).unwrap_or(0);
                let local_is_newer = local
                    .last_modified
                    .and_then(system_time_to_unix_millis)
                    .map(|local_ms| local_ms > remote_updated_at.saturating_mul(1000) + 2000)
                    .unwrap_or(false);
                if local_is_newer {
                    tracing::info!(
                        target: "drive::sync",
                        id = %self.id,
                        path = %path.display(),
                        size = local_size,
                        "Remote file has no hash and sizes match, but local file is newer; treating same-name local file as conflict"
                    );
                    return Some(ExistingRemoteLocalFileDecision::Conflict);
                }
                tracing::info!(
                    target: "drive::sync",
                    id = %self.id,
                    path = %path.display(),
                    size = local_size,
                    "Remote file has no hash but sizes match; treating as same content"
                );
                return Some(ExistingRemoteLocalFileDecision::SameContent);
            }
            tracing::info!(
                target: "drive::sync",
                id = %self.id,
                path = %path.display(),
                local_size = local_size,
                remote_size = remote_size,
                "Remote file has no hash and sizes differ; treating same-name local file as conflict"
            );
            return Some(ExistingRemoteLocalFileDecision::Conflict);
        };

        match calculate_file_hash(path.clone(), remote_fingerprint.algorithm).await {
            Ok(local_hash) if local_hash.eq_ignore_ascii_case(&remote_fingerprint.value) => {
                tracing::info!(
                    target: "drive::sync",
                    id = %self.id,
                    path = %path.display(),
                    algorithm = %remote_fingerprint.algorithm.as_str(),
                    "Existing local file matches remote hash; recording inventory without upload/download"
                );
                Some(ExistingRemoteLocalFileDecision::SameContent)
            }
            Ok(local_hash) => {
                tracing::info!(
                    target: "drive::sync",
                    id = %self.id,
                    path = %path.display(),
                    algorithm = %remote_fingerprint.algorithm.as_str(),
                    local_hash = %local_hash,
                    remote_hash = %remote_fingerprint.value,
                    "Existing local file differs from remote hash; treating as conflict candidate"
                );
                Some(ExistingRemoteLocalFileDecision::Conflict)
            }
            Err(err) => {
                tracing::warn!(
                    target: "drive::sync",
                    id = %self.id,
                    path = %path.display(),
                    error = %err,
                    algorithm = %remote_fingerprint.algorithm.as_str(),
                    "Failed to calculate local hash; treating same-name local file as conflict candidate"
                );
                Some(ExistingRemoteLocalFileDecision::Conflict)
            }
        }
    }

    async fn plan_entry_actions(
        &self,
        path: &PathBuf,
        mode: SyncMode,
        remote: Option<&FileResponse>,
        local: &LocalFileInfo,
        inventory: Option<&FileMetadata>,
        plan: &mut SyncPlan,
    ) {
        match (remote, local.exists) {
            (Some(remote_entry), true) => {
                self.plan_entry_with_remote_and_local(
                    path,
                    mode,
                    remote_entry,
                    local,
                    inventory,
                    plan,
                )
                .await
            }
            (Some(remote_entry), false) => {
                plan.actions
                    .push(SyncAction::CreatePlaceholderAndInventory {
                        path: path.clone(),
                        remote: remote_entry.clone(),
                    });
            }
            (None, true) => {
                self.plan_entry_with_local_only(path, mode, local, inventory, plan);
            }
            (None, false) => {}
        }
    }

    async fn plan_entry_with_remote_and_local(
        &self,
        path: &PathBuf,
        mode: SyncMode,
        remote: &FileResponse,
        local: &LocalFileInfo,
        inventory: Option<&FileMetadata>,
        plan: &mut SyncPlan,
    ) {
        let remote_is_dir = remote.file_type == file_type::FOLDER;

        if local.is_directory != remote_is_dir {
            if local.is_placeholder() && local.partial_on_disk() {
                plan.actions.push(SyncAction::DeleteLocalAndInventory {
                    path: path.clone(),
                    skip_if_not_empty: false,
                });
            } else {
                let conflict_path = generate_conflict_path(path);
                plan.actions.push(SyncAction::RenameLocalWithConflict {
                    original: path.clone(),
                    renamed: conflict_path,
                });
            }

            plan.actions
                .push(SyncAction::CreatePlaceholderAndInventory {
                    path: path.clone(),
                    remote: remote.clone(),
                });
            return;
        }

        let remote_state = remote_change(remote, inventory);

        if remote_is_dir {
            if remote_state != RemoteChange::Unchanged {
                plan.actions.push(SyncAction::UpdateInventoryFromRemote {
                    path: path.clone(),
                    remote: remote.clone(),
                    invalidate_all: false,
                });
            }
            self.maybe_enqueue_walk_for_directory(path, mode, local, false, false, plan);
            return;
        }

        self.plan_file_actions(path, remote, local, inventory, plan)
            .await;
    }

    fn plan_entry_with_local_only(
        &self,
        path: &PathBuf,
        mode: SyncMode,
        local: &LocalFileInfo,
        inventory: Option<&FileMetadata>,
        plan: &mut SyncPlan,
    ) {
        if !local.exists {
            return;
        }

        if local.is_directory {
            let hydrated = local.is_folder_populated();
            if !hydrated {
                plan.actions.push(SyncAction::DeleteLocalAndInventory {
                    path: path.clone(),
                    skip_if_not_empty: false,
                });
                return;
            }

            self.maybe_enqueue_walk_for_directory(path, mode, local, true, hydrated, plan);
            plan.actions.push(SyncAction::DeleteLocalAndInventory {
                path: path.clone(),
                skip_if_not_empty: true,
            });
            plan.actions
                .push(SyncAction::CreateRemoteFolderIfExist { path: path.clone() });
            return;
        }

        if !local_has_changes(local, inventory) {
            plan.actions.push(SyncAction::DeleteLocalAndInventory {
                path: path.clone(),
                skip_if_not_empty: false,
            });
            return;
        }

        // TODO: search queue if not exist:
        plan.actions.push(SyncAction::QueueUpload {
            path: path.clone(),
            reason: UploadReason::RemoteMissing,
        });
    }

    /// Plans actions for a file that exists both locally and remotely using a
    /// git-like three-way comparison between the base version (inventory), the
    /// local version and the remote version. Content hashes are consulted
    /// whenever the cheaper signals are ambiguous.
    async fn plan_file_actions(
        &self,
        path: &PathBuf,
        remote: &FileResponse,
        local: &LocalFileInfo,
        inventory: Option<&FileMetadata>,
        plan: &mut SyncPlan,
    ) {
        let conflicting =
            inventory.is_some_and(|inv| inv.conflict_state == Some(ConflictState::Pending));

        let local_changed = local_has_changes(local, inventory);
        let remote_state = remote_change(remote, inventory);

        let base_fingerprint =
            inventory.and_then(|entry| metadata_hash_fingerprint(&entry.metadata));
        let remote_fingerprint = remote_file_hash_fingerprint(remote);

        // ----- Local unchanged -----
        if !local_changed {
            match remote_state {
                RemoteChange::Unchanged => {}
                // The remote version moved backwards in time relative to the
                // last synced state. Verify with content hashes when possible;
                // otherwise treat it as a conflict instead of silently
                // overwriting (and dehydrating) local content with an older
                // remote version.
                RemoteChange::Older => {
                    if fingerprints_equal(base_fingerprint.as_ref(), remote_fingerprint.as_ref()) {
                        tracing::info!(
                            target: "drive::sync",
                            id = %self.id,
                            path = %path.display(),
                            "Remote version older than base but content identical; refreshing metadata only"
                        );
                        plan.actions.push(SyncAction::UpdateInventoryFromRemote {
                            path: path.clone(),
                            remote: remote.clone(),
                            invalidate_all: false,
                        });
                    } else {
                        tracing::warn!(
                            target: "drive::sync",
                            id = %self.id,
                            path = %path.display(),
                            "Remote version is older than the last synced version; marking as conflict instead of overwriting local content"
                        );
                        plan.actions.push(SyncAction::RecordInventoryFromRemote {
                            path: path.clone(),
                            remote: remote.clone(),
                            mark_conflicted: true,
                        });
                    }
                }
                RemoteChange::Newer => {
                    if fingerprints_equal(base_fingerprint.as_ref(), remote_fingerprint.as_ref()) {
                        // Etag/date moved but content is unchanged: refresh the
                        // inventory without destroying the local bytes.
                        plan.actions.push(SyncAction::UpdateInventoryFromRemote {
                            path: path.clone(),
                            remote: remote.clone(),
                            invalidate_all: false,
                        });
                    } else if local.pinned() == PinState::Pinned {
                        plan.actions.push(SyncAction::QueueDownload {
                            path: path.clone(),
                            remote: remote.clone(),
                        });
                    } else {
                        plan.actions.push(SyncAction::UpdateInventoryFromRemote {
                            path: path.clone(),
                            remote: remote.clone(),
                            invalidate_all: !local.partial_on_disk(),
                        });
                    }
                }
            }
            return;
        }

        // ----- Local changed -----
        if remote_state == RemoteChange::Unchanged {
            if !conflicting {
                plan.actions.push(SyncAction::QueueUpload {
                    path: path.clone(),
                    reason: UploadReason::RemoteMismatch,
                });
            }
            return;
        }

        // ----- Both changed -----
        if conflicting {
            // Already waiting for the user to resolve the conflict.
            return;
        }

        // Both sides diverged. When the local content is on disk and the
        // remote publishes a hash, auto-merge identical content instead of
        // raising a conflict.
        if !local.partial_on_disk() {
            if let Some(remote_fingerprint) = remote_fingerprint.clone() {
                match calculate_file_hash(path.clone(), remote_fingerprint.algorithm).await {
                    Ok(local_hash)
                        if local_hash.eq_ignore_ascii_case(&remote_fingerprint.value) =>
                    {
                        tracing::info!(
                            target: "drive::sync",
                            id = %self.id,
                            path = %path.display(),
                            "Local and remote diverged but content hashes match; adopting remote metadata"
                        );
                        plan.actions.push(SyncAction::UpdateInventoryFromRemote {
                            path: path.clone(),
                            remote: remote.clone(),
                            invalidate_all: false,
                        });
                        return;
                    }
                    Ok(_) => {}
                    Err(err) => {
                        tracing::debug!(
                            target: "drive::sync",
                            id = %self.id,
                            path = %path.display(),
                            error = %err,
                            "Failed to hash local file for divergence check"
                        );
                    }
                }
            }
        }

        // Genuine divergence: upload with the optimistic-lock etag so the
        // server rejects the request when its version moved, which triggers
        // the existing conflict-resolution flow.
        plan.actions.push(SyncAction::QueueUpload {
            path: path.clone(),
            reason: UploadReason::RemoteMismatch,
        });
    }

    fn maybe_enqueue_walk_for_directory(
        &self,
        path: &PathBuf,
        parent_mode: SyncMode,
        local: &LocalFileInfo,
        force_diff: bool,
        immediate: bool,
        plan: &mut SyncPlan,
    ) {
        if !local.is_directory {
            return;
        }

        let timing = if immediate {
            WalkTiming::Immediate
        } else {
            WalkTiming::Deferred
        };

        if matches!(
            parent_mode,
            SyncMode::FullHierarchy | SyncMode::PathAndFirstLayer
        ) && (local.is_folder_populated() || !local.is_placeholder())
        {
            let mode = next_child_mode(parent_mode);
            self.insert_walk_request(
                path.clone(),
                mode,
                WalkReason::ModePropagation,
                timing,
                plan,
            );
            return;
        }

        if force_diff && parent_mode == SyncMode::PathOnly {
            self.insert_walk_request(
                path.clone(),
                SyncMode::PathOnly,
                WalkReason::DiffTriggered,
                timing,
                plan,
            );
        }
    }

    fn insert_walk_request(
        &self,
        path: PathBuf,
        mode: SyncMode,
        reason: WalkReason,
        timing: WalkTiming,
        plan: &mut SyncPlan,
    ) {
        if plan
            .walk_requests
            .iter()
            .any(|request| request.path == path && request.mode == mode)
        {
            return;
        }

        plan.walk_requests.push(WalkRequest {
            path,
            mode,
            reason,
            timing,
        });
    }

    async fn process_walk_requests(
        &self,
        requests: Vec<WalkRequest>,
        aggregate_error: &mut SyncAggregateError,
    ) {
        for walk in requests {
            match self.collect_child_targets(&walk.path).await {
                Ok(result) => {
                    if result.paths.is_empty() {
                        tracing::trace!(
                            target: "drive::sync",
                            id = %self.id,
                            path = %walk.path.display(),
                            timing = ?walk.timing,
                            "Skipping walk, no children discovered"
                        );
                        continue;
                    }

                    tracing::debug!(
                        target: "drive::sync",
                        id = %self.id,
                        directory = %walk.path.display(),
                        reason = ?walk.reason,
                        next_mode = ?walk.mode,
                        children = result.paths.len(),
                        timing = ?walk.timing,
                        "Walking child directory"
                    );

                    let prefetched = if result.remote_files.is_empty() {
                        None
                    } else {
                        Some(result.remote_files)
                    };
                    let child_future =
                        Box::pin(self.sync_group(&walk.path, &result.paths, walk.mode, prefetched));
                    if let Err(err) = child_future.await {
                        tracing::error!(
                            target: "drive::sync",
                            id = %self.id,
                            directory = %walk.path.display(),
                            error = %err,
                            timing = ?walk.timing,
                            "Failed to walk child directory"
                        );
                        aggregate_error.push(walk.path.clone(), err);
                    }
                }
                Err(err) => {
                    tracing::warn!(
                        target: "drive::sync",
                        id = %self.id,
                        directory = %walk.path.display(),
                        error = %err,
                        timing = ?walk.timing,
                        "Failed to enumerate child directory"
                    );
                    aggregate_error.push(walk.path.clone(), err);
                }
            }
        }
    }

    async fn collect_child_targets(&self, directory: &PathBuf) -> Result<CollectChildResult> {
        let dir_clone = directory.clone();
        let mut children = Vec::new();
        match fs::read_dir(&dir_clone) {
            Ok(entries) => {
                for entry in entries.flatten() {
                    children.push(entry.path());
                }
            }
            Err(err) if err.kind() == io::ErrorKind::NotFound => {}
            Err(err) => {
                return Err(err).context(format!(
                    "failed to enumerate local directory {}",
                    dir_clone.display()
                ));
            }
        };

        let (remote_children, remote_files) = self.list_remote_children(directory).await?;

        let matcher = self.ignore_matcher.read().await;
        let mut dedup: HashSet<PathBuf> = HashSet::new();
        for child in children.into_iter().chain(remote_children.into_iter()) {
            if matcher.is_match(&child) {
                tracing::trace!(
                    target: "drive::sync",
                    id = %self.id,
                    path = %child.display(),
                    "Skipping ignored path during child collection"
                );
                continue;
            }
            dedup.insert(child);
        }
        drop(matcher);

        Ok(CollectChildResult {
            paths: dedup.into_iter().collect(),
            remote_files,
        })
    }

    /// Lists remote children and returns both the local paths and the file info map.
    async fn list_remote_children(
        &self,
        directory: &PathBuf,
    ) -> Result<(Vec<PathBuf>, HashMap<PathBuf, FileResponse>)> {
        let (remote_base, sync_root) = {
            let config = self.config.read().await;
            (config.remote_path.clone(), config.data_root())
        };

        let remote_dir_uri =
            match local_path_to_cr_uri(directory.clone(), sync_root.clone(), remote_base.clone()) {
                Ok(uri) => uri,
                Err(err) => {
                    tracing::warn!(
                        target: "drive::sync",
                        id = %self.id,
                        path = %directory.display(),
                        error = %err,
                        "Failed to map local directory to remote URI while walking"
                    );
                    return Ok((Vec::new(), HashMap::new()));
                }
            };
        let remote_dir_uri_str = remote_dir_uri.to_string();

        let remote_base_uri = match CrUri::new(&remote_base) {
            Ok(uri) => uri,
            Err(err) => {
                tracing::warn!(
                    target: "drive::sync",
                    id = %self.id,
                    remote_base = %remote_base,
                    error = %err,
                    "Failed to parse remote base URI while walking"
                );
                return Ok((Vec::new(), HashMap::new()));
            }
        };

        let mut previous_response = None;
        let mut children = Vec::new();
        let mut remote_files: HashMap<PathBuf, FileResponse> = HashMap::new();

        loop {
            let response = match self
                .cr_client
                .list_files_all(
                    previous_response.as_ref(),
                    remote_dir_uri_str.as_str(),
                    REMOTE_PAGE_SIZE,
                )
                .await
            {
                Ok(resp) => resp,
                Err(ApiError::ApiError { code, .. })
                    if code == ErrorCode::ParentNotExist as i32 =>
                {
                    tracing::debug!(
                        target: "drive::sync",
                        id = %self.id,
                        directory = %directory.display(),
                        "Remote directory missing during walk"
                    );
                    return Ok((Vec::new(), HashMap::new()));
                }
                Err(err) => {
                    return Err(err.into());
                }
            };

            for file in &response.res.files {
                if is_symbolic_link(file) {
                    continue;
                }

                match CrUri::new(&file.path).and_then(|file_uri| {
                    remote_path_to_local_relative_path(&file_uri, &remote_base_uri)
                }) {
                    Ok(relative) => {
                        let mut local_path = sync_root.clone();
                        local_path.push(relative);
                        if local_path
                            .parent()
                            .map(|p| p == directory.as_path())
                            .unwrap_or(false)
                        {
                            children.push(local_path.clone());
                            remote_files.insert(local_path, file.clone());
                        }
                    }
                    Err(err) => {
                        tracing::warn!(
                            target: "drive::sync",
                            id = %self.id,
                            remote_path = %file.path,
                            error = %err,
                            "Failed to map remote child to local path"
                        );
                    }
                }
            }

            if !response.more {
                break;
            }

            previous_response = Some(response);
        }

        Ok((children, remote_files))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use cloudreve_api::models::explorer::file_type;

    fn file_metadata(etag: &str, updated_at: i64) -> FileMetadata {
        FileMetadata {
            id: 0,
            drive_id: Uuid::new_v4(),
            is_folder: false,
            local_path: "/tmp/test.txt".to_string(),
            created_at: 0,
            updated_at,
            etag: etag.to_string(),
            metadata: HashMap::new(),
            props: None,
            permissions: String::new(),
            shared: false,
            size: 0,
            conflict_state: None,
            local_updated_at: None,
            local_size: None,
        }
    }

    fn file_response(etag: Option<&str>, updated_at: &str) -> FileResponse {
        FileResponse {
            file_type: file_type::FILE,
            id: "file-id".to_string(),
            name: "test.txt".to_string(),
            created_at: "1970-01-01T00:00:00Z".to_string(),
            updated_at: updated_at.to_string(),
            size: 0,
            path: "cloudreve://test.txt".to_string(),
            primary_entity: etag.map(|value| value.to_string()),
            ..Default::default()
        }
    }

    fn local_info(mtime_ms: Option<i64>, size: Option<u64>) -> LocalFileInfo {
        let mut info = LocalFileInfo::missing();
        info.exists = true;
        info.is_directory = false;
        info.file_size = size;
        info.last_modified =
            mtime_ms.map(|ms| SystemTime::UNIX_EPOCH + std::time::Duration::from_millis(ms as u64));
        info
    }

    #[test]
    fn remote_change_etag_match_is_unchanged_even_if_date_differs() {
        let entry = file_metadata("etag-a", 100);
        let remote = file_response(Some("etag-a"), "1970-01-01T00:01:41Z");
        assert_eq!(
            remote_change(&remote, Some(&entry)),
            RemoteChange::Unchanged
        );
    }

    #[test]
    fn remote_change_unparseable_date_never_forces_change() {
        let entry = file_metadata("etag-a", 100);
        let remote = file_response(Some("etag-a"), "not-a-date");
        assert_eq!(
            remote_change(&remote, Some(&entry)),
            RemoteChange::Unchanged
        );
    }

    #[test]
    fn remote_change_etag_mismatch_with_newer_date_is_newer() {
        let entry = file_metadata("etag-a", 100);
        let remote = file_response(Some("etag-b"), "1970-01-01T00:01:42Z");
        assert_eq!(remote_change(&remote, Some(&entry)), RemoteChange::Newer);
    }

    #[test]
    fn remote_change_etag_mismatch_with_older_date_is_older() {
        let entry = file_metadata("etag-a", 100);
        let remote = file_response(Some("etag-b"), "1970-01-01T00:01:38Z");
        assert_eq!(remote_change(&remote, Some(&entry)), RemoteChange::Older);
    }

    #[test]
    fn remote_change_etag_mismatch_with_unparseable_date_is_newer() {
        let entry = file_metadata("etag-a", 100);
        let remote = file_response(Some("etag-b"), "garbage");
        assert_eq!(remote_change(&remote, Some(&entry)), RemoteChange::Newer);
    }

    #[test]
    fn remote_change_without_inventory_is_newer() {
        let remote = file_response(Some("etag-a"), "garbage");
        assert_eq!(remote_change(&remote, None), RemoteChange::Newer);
    }

    #[test]
    fn remote_change_with_empty_etag_falls_back_to_date() {
        let entry = file_metadata("", 100);
        let remote = file_response(None, "1970-01-01T00:01:42Z");
        assert_eq!(remote_change(&remote, Some(&entry)), RemoteChange::Newer);
        let same = file_response(None, "1970-01-01T00:01:40Z");
        assert_eq!(remote_change(&same, Some(&entry)), RemoteChange::Unchanged);
    }

    #[test]
    fn snapshot_differs_detects_mtime_and_size_changes() {
        let entry = FileMetadata {
            local_updated_at: Some(1000),
            local_size: Some(10),
            ..file_metadata("etag", 1)
        };
        assert!(!local_snapshot_differs(
            &entry,
            &local_info(Some(1000), Some(10))
        ));
        assert!(local_snapshot_differs(
            &entry,
            &local_info(Some(2000), Some(10))
        ));
        assert!(local_snapshot_differs(
            &entry,
            &local_info(Some(1000), Some(11))
        ));
    }

    #[test]
    fn snapshot_differs_absent_snapshot_is_not_a_change() {
        let entry = FileMetadata {
            local_updated_at: None,
            local_size: None,
            ..file_metadata("etag", 1)
        };
        assert!(!local_snapshot_differs(
            &entry,
            &local_info(Some(999), Some(9))
        ));
    }

    #[test]
    fn local_has_changes_without_placeholder_tracks_snapshot_on_non_windows() {
        #[cfg(not(windows))]
        {
            let entry = FileMetadata {
                local_updated_at: Some(1000),
                local_size: Some(10),
                ..file_metadata("etag", 1)
            };
            assert!(!local_has_changes(
                &local_info(Some(1000), Some(10)),
                Some(&entry)
            ));
            assert!(local_has_changes(
                &local_info(Some(2000), Some(10)),
                Some(&entry)
            ));
            assert!(local_has_changes(&local_info(Some(1000), Some(10)), None));
        }
    }
}
