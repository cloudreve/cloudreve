//! Linux on-demand filesystem backed by FUSE.
//!
//! `CloudreveFs` projects the remote tree onto a mountpoint: entries known to
//! the inventory but absent from the local store directory appear as
//! regular files/directories ("virtual" entries), and opening one hydrates it
//! into the store on demand. Mutations land in the store, where the normal
//! filesystem watcher picks them up and feeds the existing sync machinery —
//! the FUSE layer itself never talks to the remote API except for hydration
//! fetches and explicit rename/delete of never-materialized entries.
//!
//! Layout:
//!
//! - `data_root` (`~/.cloudreve/fuse-store/<drive>`) holds real bytes. The sync
//!   engine treats it as the local root: watcher, task queue, and URI mapping
//!   all operate on store paths.
//! - `mount_path` (`config.sync_path`) is the user-visible FUSE mountpoint.
//!   It is presentation only — no engine code touches it for IO.
//!
//! Kernel caching is disabled via zero TTLs on every reply so remote changes
//! (applied to inventory + store by the sync engine) are visible immediately
//! without a `notify_inval` plumbing layer.

use std::{
    collections::HashSet,
    ffi::{OsStr, OsString},
    io,
    os::unix::fs::{FileExt, MetadataExt, OpenOptionsExt},
    path::{Path, PathBuf},
    sync::{
        Arc, Mutex,
        atomic::{AtomicU64, Ordering},
    },
    time::{Duration, Instant, SystemTime, UNIX_EPOCH},
};

use anyhow::{Context, Result};
use cloudreve_api::{Client, api::ExplorerApi};
use dashmap::DashMap;
use fuser::{
    AccessFlags, Errno, FopenFlags, FileAttr, FileHandle, FileType, Filesystem, Generation,
    INodeNo, LockOwner, OpenAccMode, OpenFlags, ReplyAttr, ReplyCreate, ReplyData, ReplyDirectory,
    ReplyEmpty, ReplyEntry, ReplyOpen, ReplyStatfs, ReplyWrite, Request, TimeOrNow, WriteFlags,
};
use notify_debouncer_full::{DebouncedEvent, notify::event::RemoveKind, notify::Event};
use notify_debouncer_full::notify::EventKind;
use tokio::sync::mpsc;

use crate::{
    drive::commands::MountCommand,
    drive::sync::group_fs_events,
    inventory::{FileMetadata, InventoryDb},
};

/// Zero TTL: the kernel re-asks for every lookup/attr so remote-side changes
/// surface without explicit invalidation.
const TTL: Duration = Duration::ZERO;
/// Synthetic generation number; the fs does not track generations.
const GENERATION: u64 = 0;
/// First inode handed out past the root (root is always inode 1).
const FIRST_INO: u64 = 2;

/// Shared state the filesystem needs from the owning `Mount`.
/// Kept separate so the FUSE session never holds the whole mount.
pub struct FuseContext {
    /// Engine-visible local root — the backing store directory.
    pub data_root: PathBuf,
    /// `cloudreve://` base URI of the mounted remote folder.
    pub remote_base: String,
    /// Owning drive id (inventory rows are per drive).
    pub drive_id: String,
    pub inventory: Arc<InventoryDb>,
    pub cr_client: Arc<Client>,
    /// Channel into the mount's command loop for remote rename/delete of
    /// entries that have no store file to generate watcher events.
    pub command_tx: mpsc::UnboundedSender<MountCommand>,
    /// Runtime handle used to `block_on` async work from FUSE threads.
    /// FUSE worker threads are not tokio threads, so `block_on` is legal.
    pub runtime: tokio::runtime::Handle,
}

/// Bidirectional inode <-> relative-path table. `1` is the mount root and maps
/// to the empty relative path.
// ponytail: inodes are never evicted — forget() only drops lookup counts.
// A session that touches every remote file leaks ~100B per entry; bounded by
// tree size and mount lifetime, acceptable for a desktop sync client.
struct InodeTable {
    by_ino: DashMap<u64, PathBuf>,
    by_path: DashMap<PathBuf, u64>,
    lookups: DashMap<u64, AtomicU64>,
    next: AtomicU64,
}

impl InodeTable {
    fn new() -> Self {
        let table = Self {
            by_ino: DashMap::new(),
            by_path: DashMap::new(),
            lookups: DashMap::new(),
            next: AtomicU64::new(FIRST_INO),
        };
        table.by_ino.insert(1, PathBuf::new());
        table.by_path.insert(PathBuf::new(), 1);
        table.lookups.insert(1, AtomicU64::new(1));
        table
    }

    fn intern(&self, rel: PathBuf) -> u64 {
        if let Some(ino) = self.by_path.get(&rel) {
            let ino = *ino;
            self.lookups
                .entry(ino)
                .or_insert_with(|| AtomicU64::new(0))
                .fetch_add(1, Ordering::Relaxed);
            return ino;
        }
        let ino = self.next.fetch_add(1, Ordering::Relaxed);
        self.by_ino.insert(ino, rel.clone());
        self.by_path.insert(rel, ino);
        self.lookups.insert(ino, AtomicU64::new(1));
        ino
    }

    fn path(&self, ino: u64) -> Option<PathBuf> {
        self.by_ino.get(&ino).map(|p| p.clone())
    }

    fn forget(&self, ino: u64, nlookup: u64) {
        if let Some(count) = self.lookups.get(&ino) {
            count.fetch_sub(nlookup, Ordering::Relaxed);
        }
    }
}

/// Handle-side bookkeeping for an open file.
struct OpenFile {
    file: std::fs::File,
    /// Store-relative path; kept so `release` can flush dirty marks later.
    #[allow(dead_code)]
    rel: PathBuf,
}

pub struct CloudreveFs {
    ctx: Arc<FuseContext>,
    inodes: InodeTable,
    open_files: DashMap<u64, OpenFile>,
    next_fh: AtomicU64,
    /// Serializes hydration per path so concurrent openers don't fetch twice.
    hydrate_locks: DashMap<PathBuf, Arc<Mutex<()>>>,
    /// Unique names for hydration temp files.
    tmp_seq: AtomicU64,
}

impl CloudreveFs {
    pub fn new(ctx: Arc<FuseContext>) -> Self {
        Self {
            ctx,
            inodes: InodeTable::new(),
            open_files: DashMap::new(),
            next_fh: AtomicU64::new(1),
            hydrate_locks: DashMap::new(),
            tmp_seq: AtomicU64::new(0),
        }
    }

    /// Temp directory for in-flight hydration. It sits next to `data_root`
    /// rather than inside it so the filesystem watcher never observes
    /// `.part` traffic; being a sibling keeps the final rename atomic on the
    /// same filesystem.
    pub fn tmp_dir(data_root: &Path, drive_id: &str) -> PathBuf {
        data_root
            .parent()
            .unwrap_or(data_root)
            .join(format!(".tmp-{drive_id}"))
    }

    fn tmp_path(&self, rel: &Path) -> PathBuf {
        let seq = self.tmp_seq.fetch_add(1, Ordering::Relaxed);
        let name = rel
            .file_name()
            .map(|n| n.to_string_lossy().into_owned())
            .unwrap_or_else(|| "file".to_string());
        Self::tmp_dir(&self.ctx.data_root, &self.ctx.drive_id)
            .join(format!("{seq}-{name}.part"))
    }

    /// Absolute store path for a mount-relative path.
    fn store_path(&self, rel: &Path) -> PathBuf {
        self.ctx.data_root.join(rel)
    }

    fn store_path_str(&self, rel: &Path) -> Option<String> {
        self.store_path(rel).to_str().map(str::to_string)
    }

    fn inventory_meta(&self, rel: &Path) -> Option<FileMetadata> {
        let store = self.store_path_str(rel)?;
        self.ctx.inventory.query_by_path(&store).ok().flatten()
    }

    fn file_type_for(meta: &FileMetadata) -> FileType {
        if meta.is_folder {
            FileType::Directory
        } else {
            FileType::RegularFile
        }
    }

    fn ts(unix: i64) -> SystemTime {
        if unix <= 0 {
            UNIX_EPOCH
        } else {
            UNIX_EPOCH + Duration::from_secs(unix as u64)
        }
    }

    /// Attributes for a relative path: the store entry wins when it exists so
    /// local writes and hydration state are always presented accurately.
    /// Otherwise fall back to the inventory's remote metadata (virtual entry).
    fn attr_for(&self, ino: u64, rel: &Path, req: &Request) -> Option<FileAttr> {
        let store = self.store_path(rel);
        if let Ok(md) = std::fs::symlink_metadata(&store) {
            return Some(Self::attr_from_std(ino, &md, req));
        }
        let meta = self.inventory_meta(rel)?;
        let kind = Self::file_type_for(&meta);
        Some(FileAttr {
            ino: INodeNo(ino),
            size: if kind == FileType::Directory {
                0
            } else {
                meta.size.max(0) as u64
            },
            blocks: 0,
            atime: Self::ts(meta.updated_at),
            mtime: Self::ts(meta.updated_at),
            ctime: Self::ts(meta.updated_at),
            crtime: Self::ts(meta.created_at),
            kind,
            perm: if kind == FileType::Directory { 0o755 } else { 0o644 },
            nlink: if kind == FileType::Directory { 2 } else { 1 },
            uid: req.uid(),
            gid: req.gid(),
            rdev: 0,
            blksize: 4096,
            flags: 0,
        })
    }

    fn attr_from_std(ino: u64, md: &std::fs::Metadata, req: &Request) -> FileAttr {
        FileAttr {
            ino: INodeNo(ino),
            size: md.len(),
            blocks: md.blocks(),
            atime: md.accessed().unwrap_or(UNIX_EPOCH),
            mtime: md.modified().unwrap_or(UNIX_EPOCH),
            ctime: UNIX_EPOCH + Duration::from_secs(md.ctime() as u64),
            crtime: md.created().unwrap_or(UNIX_EPOCH),
            kind: FileType::from_std(md.file_type()).unwrap_or(FileType::RegularFile),
            perm: (md.mode() & 0o7777) as u16,
            nlink: md.nlink() as u32,
            uid: req.uid(),
            gid: req.gid(),
            rdev: md.rdev() as u32,
            blksize: 4096,
            flags: 0,
        }
    }

    /// Ensure a file's bytes exist in the store, downloading them on first
    /// access. Concurrent opens on the same path share one hydration.
    fn ensure_materialized(&self, rel: &Path) -> io::Result<()> {
        let store = self.store_path(rel);
        if store.exists() {
            return Ok(());
        }
        let meta = self
            .inventory_meta(rel)
            .ok_or_else(|| io::Error::from_raw_os_error(libc::ENOENT))?;
        if meta.is_folder {
            return Err(io::Error::from_raw_os_error(libc::EISDIR));
        }

        let lock = self
            .hydrate_locks
            .entry(rel.to_path_buf())
            .or_insert_with(|| Arc::new(Mutex::new(())))
            .clone();
        let _guard = lock.lock().unwrap_or_else(|e| e.into_inner());

        // Another opener may have hydrated while we waited on the lock.
        if store.exists() {
            self.hydrate_locks.remove(rel);
            return Ok(());
        }

        if let Some(parent) = store.parent() {
            std::fs::create_dir_all(parent)?;
        }

        let tmp = self.tmp_path(rel);
        let result = self
            .ctx
            .runtime
            .block_on(self.fetch_remote_file(&store, &tmp));
        // The store.exists() check above is the real guard against duplicate
        // fetches, so releasing the path lock here is safe and keeps the map
        // bounded over the mount's lifetime.
        self.hydrate_locks.remove(rel);
        result.map_err(|e| {
            tracing::error!(
                target: "drive::fuse",
                path = %store.display(),
                error = %e,
                "Hydration failed"
            );
            let _ = std::fs::remove_file(&tmp);
            io::Error::from_raw_os_error(libc::EIO)
        })?;

        // Refresh the local snapshot so the next sync diff does not treat the
        // freshly hydrated bytes as a local modification.
        if let Ok(Some(mut meta)) = self.ctx.inventory.query_by_path(&self.store_path_str(rel).unwrap_or_default())
            && let Ok(fresh) = std::fs::metadata(&store)
            && let Ok(mtime) = fresh.modified()
            && let Ok(dur) = mtime.duration_since(UNIX_EPOCH)
        {
            meta.local_updated_at = Some(dur.as_millis() as i64);
            meta.local_size = Some(fresh.len() as i64);
            let entry = crate::inventory::MetadataEntry::from(&meta);
            let _ = self.ctx.inventory.upsert(&entry);
        }
        Ok(())
    }

    /// Download the remote entity backing `store` to `tmp` and rename it into
    /// place. The rename keeps concurrent readers from seeing partial content.
    async fn fetch_remote_file(&self, store: &Path, tmp: &Path) -> Result<()> {
        let uri = crate::drive::utils::local_path_to_cr_uri(
            store.to_path_buf(),
            self.ctx.data_root.clone(),
            self.ctx.remote_base.clone(),
        )
        .context("failed to map store path to remote uri")?;

        let mut request = cloudreve_api::models::explorer::FileURLService::default();
        request.uris.push(uri.to_string());
        if let Some(meta) = self
            .ctx
            .inventory
            .query_by_path(&store.to_string_lossy())?
            && !meta.etag.is_empty()
        {
            request.entity = Some(meta.etag);
        }
        let url_res = match self.ctx.cr_client.get_file_url(&request).await {
            Err(e) if e.is_entity_not_exist() && request.entity.is_some() => {
                let mut retry = request.clone();
                retry.entity = None;
                self.ctx.cr_client.get_file_url(&retry).await?
            }
            res => res?,
        };
        let download_url = url_res
            .urls
            .first()
            .context("no download URL in response")?
            .url
            .clone();

        if let Some(parent) = tmp.parent() {
            std::fs::create_dir_all(parent)
                .with_context(|| format!("failed to create temp dir {}", parent.display()))?;
        }
        let mut out = std::fs::File::create(tmp)
            .with_context(|| format!("failed to create temp file {}", tmp.display()))?;
        let client = reqwest::Client::new();
        let response = client
            .get(&download_url)
            .send()
            .await
            .context("failed to send download request")?;
        if !response.status().is_success() {
            anyhow::bail!("download failed with status {}", response.status());
        }
        use futures::StreamExt;
        let mut stream = response.bytes_stream();
        use std::io::Write;
        while let Some(chunk) = stream.next().await {
            let chunk = chunk.context("failed to read download stream")?;
            out.write_all(&chunk)?;
        }
        out.sync_all()?;
        drop(out);
        std::fs::rename(tmp, store).context("failed to move hydrated file into place")?;
        Ok(())
    }

    /// Ensure the entry exists in the store as the right kind: directories
    /// materialize as empty dirs, files hydrate. Used by setattr so `touch`
    /// and friends work on virtual directories too.
    fn ensure_materialized_any(&self, rel: &Path) -> io::Result<()> {
        let store = self.store_path(rel);
        if store.exists() {
            return Ok(());
        }
        match self.inventory_meta(rel) {
            Some(meta) if meta.is_folder => std::fs::create_dir_all(&store),
            _ => self.ensure_materialized(rel),
        }
    }

    /// Route a mutation that produced no store-side watcher event (the entry
    /// was virtual) through the same path real filesystem events take.
    fn emit_fs_event(&self, store_path: PathBuf, kind: EventKind) {
        let event = DebouncedEvent::new(Event::new(kind).add_path(store_path), Instant::now());
        if let Err(e) = self.ctx.command_tx.send(MountCommand::ProcessFsEvents {
            events: group_fs_events(vec![event]),
        }) {
            tracing::error!(
                target: "drive::fuse",
                error = %e,
                "Failed to emit synthesized fs event"
            );
        }
    }

    fn register_file(&self, rel: PathBuf, file: std::fs::File) -> FileHandle {
        let fh = self.next_fh.fetch_add(1, Ordering::Relaxed);
        self.open_files.insert(fh, OpenFile { file, rel });
        FileHandle(fh)
    }
}

impl Filesystem for CloudreveFs {
    fn lookup(&self, req: &Request, parent: INodeNo, name: &OsStr, reply: ReplyEntry) {
        let Some(parent_rel) = self.inodes.path(parent.0) else {
            reply.error(Errno::ENOENT);
            return;
        };
        let rel = parent_rel.join(name);
        let ino = self.inodes.intern(rel.clone());
        match self.attr_for(ino, &rel, req) {
            Some(attr) => reply.entry(&TTL, &attr, Generation(GENERATION)),
            None => reply.error(Errno::ENOENT),
        }
    }

    fn forget(&self, _req: &Request, ino: INodeNo, nlookup: u64) {
        self.inodes.forget(ino.0, nlookup);
    }

    fn getattr(&self, req: &Request, ino: INodeNo, _fh: Option<FileHandle>, reply: ReplyAttr) {
        let Some(rel) = self.inodes.path(ino.0) else {
            reply.error(Errno::ENOENT);
            return;
        };
        match self.attr_for(ino.0, &rel, req) {
            Some(attr) => reply.attr(&TTL, &attr),
            None => reply.error(Errno::ENOENT),
        }
    }

    fn setattr(
        &self,
        req: &Request,
        ino: INodeNo,
        _mode: Option<u32>,
        _uid: Option<u32>,
        _gid: Option<u32>,
        size: Option<u64>,
        _atime: Option<TimeOrNow>,
        mtime: Option<TimeOrNow>,
        _ctime: Option<SystemTime>,
        fh: Option<FileHandle>,
        _crtime: Option<SystemTime>,
        _chgtime: Option<SystemTime>,
        _bkuptime: Option<SystemTime>,
        _flags: Option<fuser::BsdFileFlags>,
        reply: ReplyAttr,
    ) {
        let Some(rel) = self.inodes.path(ino.0) else {
            reply.error(Errno::ENOENT);
            return;
        };
        let store = self.store_path(&rel);

        if let Some(new_size) = size {
            if let Err(e) = self.ensure_materialized_any(&rel) {
                reply.error(Errno::from(e));
                return;
            }
            let res = match fh.and_then(|fh| self.open_files.get(&fh.0).map(|f| f.file.try_clone())) {
                Some(Ok(file)) => file.set_len(new_size),
                _ => std::fs::File::options()
                    .write(true)
                    .open(&store)
                    .and_then(|file| file.set_len(new_size)),
            };
            if let Err(e) = res {
                reply.error(Errno::from(e));
                return;
            }
        }

        if let Some(mtime) = mtime {
            if let Err(e) = self.ensure_materialized_any(&rel) {
                reply.error(Errno::from(e));
                return;
            }
            let t = match mtime {
                TimeOrNow::SpecificTime(t) => t,
                TimeOrNow::Now => SystemTime::now(),
            };
            let _ = std::fs::File::options()
                .write(true)
                .open(&store)
                .map(|file| file.set_modified(t));
        }

        match self.attr_for(ino.0, &rel, req) {
            Some(attr) => reply.attr(&TTL, &attr),
            None => reply.error(Errno::ENOENT),
        }
    }

    fn mkdir(
        &self,
        req: &Request,
        parent: INodeNo,
        name: &OsStr,
        mode: u32,
        _umask: u32,
        reply: ReplyEntry,
    ) {
        let Some(parent_rel) = self.inodes.path(parent.0) else {
            reply.error(Errno::ENOENT);
            return;
        };
        let rel = parent_rel.join(name);
        let store = self.store_path(&rel);
        if !store.exists() && self.inventory_meta(&rel).is_some() {
            reply.error(Errno::EEXIST);
            return;
        }
        // The parent may be virtual — materialize the path chain first.
        if let Some(p) = store.parent()
            && let Err(e) = std::fs::create_dir_all(p)
        {
            reply.error(Errno::from(e));
            return;
        }
        if let Err(e) = std::fs::create_dir(&store) {
            reply.error(Errno::from(e));
            return;
        }
        if mode != 0 {
            use std::os::unix::fs::PermissionsExt;
            let _ = std::fs::set_permissions(&store, std::fs::Permissions::from_mode(mode));
        }
        let ino = self.inodes.intern(rel.clone());
        match self.attr_for(ino, &rel, req) {
            Some(attr) => reply.entry(&TTL, &attr, Generation(GENERATION)),
            None => reply.error(Errno::EIO),
        }
    }

    fn unlink(&self, _req: &Request, parent: INodeNo, name: &OsStr, reply: ReplyEmpty) {
        let Some(parent_rel) = self.inodes.path(parent.0) else {
            reply.error(Errno::ENOENT);
            return;
        };
        let rel = parent_rel.join(name);
        let store = self.store_path(&rel);
        if store.exists() {
            match std::fs::remove_file(&store) {
                Ok(()) => reply.ok(),
                Err(e) => reply.error(Errno::from(e)),
            }
            return;
        }
        if self.inventory_meta(&rel).is_none() {
            reply.error(Errno::ENOENT);
            return;
        }
        // Virtual entry: no store file exists, so no watcher event will fire.
        // Feed the remove through the synthesized-event path so the remote
        // delete and inventory cleanup still happen.
        self.emit_fs_event(store, EventKind::Remove(RemoveKind::File));
        reply.ok();
    }

    fn rmdir(&self, _req: &Request, parent: INodeNo, name: &OsStr, reply: ReplyEmpty) {
        let Some(parent_rel) = self.inodes.path(parent.0) else {
            reply.error(Errno::ENOENT);
            return;
        };
        let rel = parent_rel.join(name);
        let store = self.store_path(&rel);
        if store.exists() {
            match std::fs::remove_dir(&store) {
                Ok(()) => reply.ok(),
                Err(e) => reply.error(Errno::from(e)),
            }
            return;
        }
        if self.inventory_meta(&rel).is_none() {
            reply.error(Errno::ENOENT);
            return;
        }
        self.emit_fs_event(store, EventKind::Remove(RemoveKind::Folder));
        reply.ok();
    }

    fn rename(
        &self,
        _req: &Request,
        parent: INodeNo,
        name: &OsStr,
        newparent: INodeNo,
        newname: &OsStr,
        _flags: fuser::RenameFlags,
        reply: ReplyEmpty,
    ) {
        let (Some(from_rel), Some(to_parent_rel)) =
            (self.inodes.path(parent.0), self.inodes.path(newparent.0))
        else {
            reply.error(Errno::ENOENT);
            return;
        };
        let from_rel = from_rel.join(name);
        let to_rel = to_parent_rel.join(newname);
        let src_store = self.store_path(&from_rel);
        let dst_store = self.store_path(&to_rel);

        // Remote first: the Rename command performs the server-side rename/move
        // and registers event blockers so resulting watcher events are
        // suppressed. Bail out without touching local state on failure.
        let (tx, rx) = tokio::sync::oneshot::channel();
        if self
            .ctx
            .command_tx
            .send(MountCommand::Rename {
                source: src_store.clone(),
                target: dst_store.clone(),
                response: tx,
            })
            .is_err()
        {
            reply.error(Errno::EIO);
            return;
        }
        match rx.blocking_recv() {
            Ok(Ok(())) => {}
            Ok(Err(e)) => {
                tracing::error!(
                    target: "drive::fuse",
                    from = %src_store.display(),
                    to = %dst_store.display(),
                    error = %e,
                    "Remote rename failed"
                );
                reply.error(Errno::EIO);
                return;
            }
            Err(_) => {
                reply.error(Errno::EIO);
                return;
            }
        }

        // Local second: real rename when the entry exists in the store. For a
        // virtual entry there is nothing to move — inventory gets updated by
        // the Renamed command below.
        if src_store.exists() {
            if let Some(parent) = dst_store.parent() {
                let _ = std::fs::create_dir_all(parent);
            }
            if let Err(e) = std::fs::rename(&src_store, &dst_store) {
                reply.error(Errno::from(e));
                return;
            }
        }

        let _ = self.ctx.command_tx.send(MountCommand::Renamed {
            source: src_store,
            destination: dst_store,
        });
        reply.ok();
    }

    fn open(&self, _req: &Request, ino: INodeNo, flags: OpenFlags, reply: ReplyOpen) {
        let Some(rel) = self.inodes.path(ino.0) else {
            reply.error(Errno::ENOENT);
            return;
        };
        let store = self.store_path(&rel);
        let write_intent = flags.acc_mode() != OpenAccMode::O_RDONLY;

        if (write_intent || !store.exists())
            && let Err(e) = self.ensure_materialized(&rel)
        {
            reply.error(Errno::from(e));
            return;
        }

        let mut options = std::fs::OpenOptions::new();
        options.read(true);
        if write_intent {
            options.write(true);
        }
        if flags.0 & libc::O_TRUNC != 0 {
            options.truncate(true);
        }
        match options.open(&store) {
            Ok(file) => {
                let fh = self.register_file(rel, file);
                reply.opened(fh, FopenFlags::empty());
            }
            Err(e) => reply.error(Errno::from(e)),
        }
    }

    fn read(
        &self,
        _req: &Request,
        _ino: INodeNo,
        fh: FileHandle,
        offset: u64,
        size: u32,
        _flags: OpenFlags,
        _lock_owner: Option<LockOwner>,
        reply: ReplyData,
    ) {
        let Some(entry) = self.open_files.get(&fh.0) else {
            reply.error(Errno::EBADF);
            return;
        };
        let mut buf = vec![0u8; size as usize];
        match entry.file.read_at(&mut buf, offset) {
            Ok(n) => {
                buf.truncate(n);
                reply.data(&buf);
            }
            Err(e) => reply.error(Errno::from(e)),
        }
    }

    fn write(
        &self,
        _req: &Request,
        _ino: INodeNo,
        fh: FileHandle,
        offset: u64,
        data: &[u8],
        _write_flags: WriteFlags,
        _flags: OpenFlags,
        _lock_owner: Option<LockOwner>,
        reply: ReplyWrite,
    ) {
        let Some(entry) = self.open_files.get(&fh.0) else {
            reply.error(Errno::EBADF);
            return;
        };
        match entry.file.write_at(data, offset) {
            Ok(n) => reply.written(n as u32),
            Err(e) => reply.error(Errno::from(e)),
        }
    }

    fn flush(
        &self,
        _req: &Request,
        _ino: INodeNo,
        fh: FileHandle,
        _lock_owner: LockOwner,
        reply: ReplyEmpty,
    ) {
        if let Some(entry) = self.open_files.get(&fh.0) {
            let _ = entry.file.sync_data();
        }
        reply.ok();
    }

    fn release(
        &self,
        _req: &Request,
        _ino: INodeNo,
        fh: FileHandle,
        _flags: OpenFlags,
        _lock_owner: Option<LockOwner>,
        _flush: bool,
        reply: ReplyEmpty,
    ) {
        self.open_files.remove(&fh.0);
        reply.ok();
    }

    fn fsync(
        &self,
        _req: &Request,
        _ino: INodeNo,
        fh: FileHandle,
        _datasync: bool,
        reply: ReplyEmpty,
    ) {
        if let Some(entry) = self.open_files.get(&fh.0)
            && let Err(e) = entry.file.sync_data()
        {
            reply.error(Errno::from(e));
            return;
        }
        reply.ok();
    }

    fn opendir(&self, _req: &Request, _ino: INodeNo, _flags: OpenFlags, reply: ReplyOpen) {
        reply.opened(FileHandle(0), FopenFlags::empty());
    }

    fn readdir(&self, _req: &Request, ino: INodeNo, _fh: FileHandle, offset: u64, mut reply: ReplyDirectory) {
        let Some(rel) = self.inodes.path(ino.0) else {
            reply.error(Errno::ENOENT);
            return;
        };
        let store = self.store_path(&rel);

        // Union store children with inventory (remote) children, deduplicated
        // by name. The store is authoritative for entries it holds.
        let mut seen: HashSet<OsString> = HashSet::new();
        let mut entries: Vec<(OsString, FileType, u64)> = Vec::new();

        if let Ok(read_dir) = std::fs::read_dir(&store) {
            for entry in read_dir.flatten() {
                let name = entry.file_name();
                let kind = entry
                    .file_type()
                    .ok()
                    .and_then(FileType::from_std)
                    .unwrap_or(FileType::RegularFile);
                let child_rel = rel.join(&name);
                let child_ino = self.inodes.intern(child_rel);
                if seen.insert(name.clone()) {
                    entries.push((name, kind, child_ino));
                }
            }
        }

        if let Some(store_str) = store.to_str()
            && let Ok(children) = self.ctx.inventory.query_children(&self.ctx.drive_id, store_str)
        {
            for child in children {
                let child_path = PathBuf::from(&child.local_path);
                let Some(name) = child_path.file_name().map(|n| n.to_os_string()) else {
                    continue;
                };
                if !seen.insert(name.clone()) {
                    continue;
                }
                let child_rel = rel.join(&name);
                let child_ino = self.inodes.intern(child_rel);
                entries.push((name, Self::file_type_for(&child), child_ino));
            }
        }

        entries.sort_by(|a, b| a.0.cmp(&b.0));

        let parent_rel = rel.parent().map(|p| p.to_path_buf()).unwrap_or_default();
        let all: Vec<(u64, FileType, OsString)> = std::iter::once((ino.0, FileType::Directory, OsString::from(".")))
            .chain(std::iter::once((
                self.inodes.intern(parent_rel),
                FileType::Directory,
                OsString::from(".."),
            )))
            .chain(entries.into_iter().map(|(name, kind, child_ino)| (child_ino, kind, name)))
            .collect();

        for (i, (entry_ino, kind, name)) in all.iter().enumerate().skip(offset as usize) {
            if reply.add(INodeNo(*entry_ino), (i + 1) as u64, *kind, name) {
                break;
            }
        }
        reply.ok();
    }

    fn statfs(&self, _req: &Request, _ino: INodeNo, reply: ReplyStatfs) {
        let path = &self.ctx.data_root;
        match statvfs(path) {
            Some(stat) => reply.statfs(
                stat.f_blocks,
                stat.f_bfree,
                stat.f_bavail,
                stat.f_files,
                stat.f_ffree,
                stat.f_bsize as u32,
                stat.f_namemax as u32,
                stat.f_frsize as u32,
            ),
            None => reply.statfs(0, 0, 0, 0, 0, 4096, 255, 4096),
        }
    }

    fn access(&self, _req: &Request, _ino: INodeNo, _mask: AccessFlags, reply: ReplyEmpty) {
        reply.ok();
    }

    fn create(
        &self,
        req: &Request,
        parent: INodeNo,
        name: &OsStr,
        mode: u32,
        _umask: u32,
        flags: i32,
        reply: ReplyCreate,
    ) {
        let Some(parent_rel) = self.inodes.path(parent.0) else {
            reply.error(Errno::ENOENT);
            return;
        };
        let rel = parent_rel.join(name);
        let store = self.store_path(&rel);
        let virtual_entry = !store.exists() && self.inventory_meta(&rel).is_some();

        if virtual_entry {
            if flags & libc::O_EXCL != 0 && flags & libc::O_CREAT != 0 {
                reply.error(Errno::EEXIST);
                return;
            }
            // O_CREAT|O_TRUNC on a remote file must hydrate first: creating an
            // empty store file would make the watcher upload the truncation
            // over remote content.
            if let Err(e) = self.ensure_materialized(&rel) {
                reply.error(Errno::from(e));
                return;
            }
        } else if !store.exists()
            && let Some(p) = store.parent()
            && let Err(e) = std::fs::create_dir_all(p)
        {
            // New file under a possibly-virtual parent — materialize the path
            // chain so the create below does not fail with ENOENT.
            reply.error(Errno::from(e));
            return;
        }

        let mut options = std::fs::OpenOptions::new();
        options.read(true).write(true);
        if flags & libc::O_EXCL != 0 && flags & libc::O_CREAT != 0 {
            options.create_new(true);
        } else {
            options.create(true);
        }
        if flags & libc::O_TRUNC != 0 {
            options.truncate(true);
        }
        options.mode(mode & 0o7777);

        match options.open(&store) {
            Ok(file) => {
                let ino = self.inodes.intern(rel.clone());
                let fh = self.register_file(rel.clone(), file);
                match self.attr_for(ino, &rel, req) {
                    Some(attr) => reply.created(&TTL, &attr, Generation(GENERATION), fh, FopenFlags::empty()),
                    None => reply.error(Errno::EIO),
                }
            }
            Err(e) => reply.error(Errno::from(e)),
        }
    }
}

/// statvfs wrapper so `statfs` reports real store capacity without pulling in
/// a full `nix` dependency.
fn statvfs(path: &Path) -> Option<libc::statvfs> {
    use std::os::unix::ffi::OsStrExt;
    let c_path = std::ffi::CString::new(path.as_os_str().as_bytes()).ok()?;
    let mut stat: libc::statvfs = unsafe { std::mem::zeroed() };
    let rc = unsafe { libc::statvfs(c_path.as_ptr(), &mut stat) };
    if rc == 0 { Some(stat) } else { None }
}

/// Spawn the FUSE session in the background. The returned session unmounts on
/// drop; callers must keep it alive for the mount's lifetime.
pub fn spawn(fs: CloudreveFs, mountpoint: &Path) -> io::Result<fuser::BackgroundSession> {
    let mut config = fuser::Config::default();
    config.mount_options = vec![
        fuser::MountOption::FSName("cloudreve".to_string()),
        fuser::MountOption::AutoUnmount,
    ];
    config.n_threads = Some(4);
    fuser::spawn_mount(fs, mountpoint, &config)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn inode_table_interns_paths_bidirectionally() {
        let table = InodeTable::new();
        assert_eq!(table.path(1).as_deref(), Some(Path::new("")));

        let ino_a = table.intern(PathBuf::from("dir/file.txt"));
        let ino_b = table.intern(PathBuf::from("dir/other.txt"));
        assert_ne!(ino_a, ino_b);
        assert!(ino_a >= FIRST_INO);

        // Same path interns to the same inode.
        assert_eq!(table.intern(PathBuf::from("dir/file.txt")), ino_a);
        assert_eq!(
            table.path(ino_a).as_deref(),
            Some(Path::new("dir/file.txt"))
        );
    }

    #[test]
    fn tmp_dir_sits_next_to_data_root() {
        let tmp = CloudreveFs::tmp_dir(Path::new("/x/fuse-store/d1"), "d1");
        assert_eq!(tmp, PathBuf::from("/x/fuse-store/.tmp-d1"));
    }
}
