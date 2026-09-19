use std::{path::PathBuf, sync::Arc};

use anyhow::{Context, Result};
use chrono::DateTime;
use cloudreve_api::models::explorer::{FileResponse, file_type};
use uuid::Uuid;

use crate::{
    cfapi::placeholder::LocalFileInfo,
    inventory::{FileMetadata, InventoryDb, MetadataEntry},
};

/// Non-Windows compatibility implementation for the Windows `CrPlaceholder` API.
///
/// Despite the type name, this implementation does **not** create dehydrated,
/// on-demand, or filesystem-provider placeholders. Linux and other non-Windows
/// platforms use full sync only, so this type is a small adapter that keeps the
/// shared sync code compiling while mapping placeholder operations to ordinary
/// local filesystem and inventory operations:
///
/// - remote folders create real local directories;
/// - remote files only update inventory metadata here and are downloaded by the
///   normal download queue;
/// - delete operations remove real local files or directories if they exist.
///
/// The public method names intentionally mirror the Windows implementation so
/// callers can use one cross-platform code path. Review this file as the
/// non-Windows full-sync metadata helper, not as a placeholder backend.
pub struct CrPlaceholder {
    pub local_file_info: LocalFileInfo,
    local_path: PathBuf,
    drive_id: Uuid,
    file_meta: Option<FileMetadata>,
    mark_no_children: bool,
}

impl CrPlaceholder {
    /// Creates a non-Windows full-sync adapter for a local path.
    ///
    /// The `sync_root` parameter is accepted for API parity with the Windows
    /// placeholder implementation, but it is not needed here because no cloud
    /// filter root or provider registration exists on non-Windows platforms.
    ///
    /// `local_file_info` is populated from the real local path when possible.
    /// If the path does not exist yet, the adapter records a missing file state;
    /// this is expected for remote files that will be downloaded later.
    pub fn new(local_path: impl Into<PathBuf>, _sync_root: PathBuf, drive_id: Uuid) -> Self {
        let local_path = local_path.into();
        Self {
            local_file_info: LocalFileInfo::from_path(&local_path)
                .unwrap_or_else(|_| LocalFileInfo::missing()),
            local_path,
            drive_id,
            file_meta: None,
            mark_no_children: false,
        }
    }

    /// No-op compatibility hook for the Windows range invalidation option.
    ///
    /// Windows placeholders can invalidate byte ranges through CFAPI. Non-Windows
    /// full sync has no dehydrated file ranges, so this method deliberately
    /// returns `self` unchanged.
    pub fn with_invalidate_all_range(self, _enable: bool) -> Self {
        self
    }

    /// Records the requested "no children" marker for API parity.
    ///
    /// The flag is kept so shared builder chains behave consistently, but no
    /// non-Windows filesystem metadata is written from this value.
    pub fn with_mark_no_children(mut self, enable: bool) -> Self {
        self.mark_no_children = enable;
        self
    }

    /// Attaches inventory metadata that will be committed later.
    ///
    /// On non-Windows platforms this metadata describes a fully synced local
    /// entry. For files, committing the metadata does not create any file bytes;
    /// the download task is responsible for creating the real file.
    pub fn with_file_meta(mut self, file_meta: FileMetadata) -> Self {
        self.file_meta = Some(file_meta);
        self
    }

    /// Deletes the real local file or directory and removes its inventory row.
    ///
    /// The name mirrors the Windows placeholder API, but the operation here is
    /// not limited to placeholder metadata. If the path exists, this removes the
    /// actual synced file or directory from disk before deleting inventory data.
    pub fn delete_placeholder(&self, inventory: Arc<InventoryDb>) -> Result<()> {
        if self.local_file_info.exists {
            if self.local_path.is_dir() {
                std::fs::remove_dir_all(&self.local_path)
                    .context("failed to delete local directory")?;
            } else {
                std::fs::remove_file(&self.local_path).context("failed to delete local file")?;
            }
        }

        let path_str = self
            .local_path
            .to_str()
            .context("failed to convert path to string")?;
        inventory
            .batch_delete_by_path(vec![path_str])
            .context("failed to delete from inventory")?;
        Ok(())
    }

    /// Commits the pending metadata using non-Windows full-sync semantics.
    ///
    /// For folders, this creates a real local directory immediately and then
    /// stores metadata in the inventory. For files, this only ensures the parent
    /// directory exists and stores metadata; it intentionally does **not** create
    /// a placeholder file. The normal download queue must later write the full
    /// file contents to disk.
    pub fn commit(&mut self, inventory: Arc<InventoryDb>) -> Result<()> {
        let file_meta = self
            .file_meta
            .as_ref()
            .ok_or_else(|| anyhow::anyhow!("File metadata is not set"))?;

        if file_meta.is_folder {
            std::fs::create_dir_all(&self.local_path)
                .context("failed to create local directory")?;
        } else if let Some(parent) = self.local_path.parent() {
            std::fs::create_dir_all(parent).context("failed to create parent directory")?;
        }

        let mut entry = MetadataEntry::from(file_meta);
        match LocalFileInfo::from_path(&self.local_path) {
            Ok(fresh) if fresh.exists => {
                let mtime_ms = fresh.last_modified.and_then(|time| {
                    time.duration_since(std::time::UNIX_EPOCH)
                        .ok()
                        .map(|duration| duration.as_millis() as i64)
                });
                if let (Some(mtime_ms), Some(size)) = (mtime_ms, fresh.file_size) {
                    entry = entry.with_local_snapshot(mtime_ms, size as i64);
                }
            }
            _ => {}
        }

        inventory
            .upsert(&entry)
            .context("failed to upsert inventory")?;
        self.local_file_info =
            LocalFileInfo::from_path(&self.local_path).unwrap_or_else(|_| LocalFileInfo::missing());
        Ok(())
    }

    /// Converts a remote API file response into local inventory metadata.
    ///
    /// This method does not create any local file or directory by itself. It only
    /// prepares the metadata that `commit` will apply using non-Windows full-sync
    /// behavior.
    pub fn with_remote_file(mut self, file_info: &FileResponse) -> Self {
        let created_at = DateTime::parse_from_rfc3339(&file_info.created_at)
            .ok()
            .map(|dt| dt.timestamp())
            .unwrap_or_default();

        let updated_at = DateTime::parse_from_rfc3339(&file_info.updated_at)
            .ok()
            .map(|dt| dt.timestamp())
            .unwrap_or_default();

        self.file_meta = Some(FileMetadata {
            id: 0,
            drive_id: self.drive_id,
            local_path: self.local_path.to_string_lossy().to_string(),
            is_folder: file_info.file_type == file_type::FOLDER,
            created_at,
            updated_at,
            size: file_info.size,
            etag: file_info
                .primary_entity
                .as_ref()
                .unwrap_or(&String::new())
                .clone(),
            permissions: file_info
                .permission
                .as_ref()
                .unwrap_or(&String::new())
                .clone(),
            shared: file_info.shared.unwrap_or(false),
            metadata: file_info.metadata.clone().unwrap_or_default(),
            props: None,
            conflict_state: None,
            local_updated_at: None,
            local_size: None,
        });
        self
    }

    /// No-op compatibility hook for Windows placeholder sync error state.
    ///
    /// Windows can surface sync error state through shell/provider metadata.
    /// Non-Windows full sync currently has no equivalent per-file shell state,
    /// so this method succeeds without changing the filesystem or inventory.
    pub fn update_sync_error_state(&mut self, _sync_error: bool) -> Result<()> {
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use cloudreve_api::models::explorer::file_type;
    use tempfile::tempdir;

    #[test]
    fn remote_file_commit_updates_inventory_without_creating_placeholder_file() {
        let temp = tempdir().unwrap();
        let sync_root = temp.path().join("sync");
        let local_path = sync_root.join("remote.txt");
        let inventory = Arc::new(InventoryDb::with_path(temp.path().join("meta.db")).unwrap());
        let drive_id = Uuid::new_v4();
        let remote = FileResponse {
            file_type: file_type::FILE,
            id: "file-id".to_string(),
            name: "remote.txt".to_string(),
            created_at: "2026-01-01T00:00:00Z".to_string(),
            updated_at: "2026-01-01T00:00:00Z".to_string(),
            size: 42,
            path: "cloudreve://remote.txt".to_string(),
            primary_entity: Some("etag".to_string()),
            ..Default::default()
        };

        CrPlaceholder::new(local_path.clone(), sync_root, drive_id)
            .with_remote_file(&remote)
            .commit(inventory.clone())
            .unwrap();

        assert!(!local_path.exists());
        let stored = inventory
            .query_by_path(local_path.to_str().unwrap())
            .unwrap()
            .unwrap();
        assert_eq!(stored.etag, "etag");
        assert!(!stored.is_folder);
    }

    #[test]
    fn remote_folder_commit_creates_local_directory() {
        let temp = tempdir().unwrap();
        let sync_root = temp.path().join("sync");
        let local_path = sync_root.join("folder");
        let inventory = Arc::new(InventoryDb::with_path(temp.path().join("meta.db")).unwrap());
        let drive_id = Uuid::new_v4();
        let remote = FileResponse {
            file_type: file_type::FOLDER,
            id: "folder-id".to_string(),
            name: "folder".to_string(),
            created_at: "2026-01-01T00:00:00Z".to_string(),
            updated_at: "2026-01-01T00:00:00Z".to_string(),
            path: "cloudreve://folder".to_string(),
            primary_entity: Some("folder-etag".to_string()),
            ..Default::default()
        };

        CrPlaceholder::new(local_path.clone(), sync_root, drive_id)
            .with_remote_file(&remote)
            .commit(inventory)
            .unwrap();

        assert!(local_path.is_dir());
    }
}
