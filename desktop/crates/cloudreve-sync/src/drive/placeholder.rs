use crate::{
    cfapi::{
        metadata::Metadata,
        placeholder::{ConvertOptions, LocalFileInfo, OpenOptions, UpdateOptions},
        placeholder_file::PlaceholderFile,
    },
    drive::utils::notify_shell_change,
    inventory::{FileMetadata, InventoryDb, MetadataEntry},
};
use anyhow::{Context, Result};
use chrono::DateTime;
use cloudreve_api::models::explorer::{FileResponse, file_type};
use nt_time::FileTime;
use std::{
    ffi::OsString,
    path::PathBuf,
    sync::Arc,
};
use uuid::Uuid;
use widestring::U16CString;
use windows::{
    Win32::{
        Foundation::E_FAIL,
        Storage::EnhancedStorage::PKEY_LastSyncError,
        System::Variant::VT_UI4,
        UI::Shell::{
            IShellItem2,
            PropertiesSystem::{
                GPS_EXTRINSICPROPERTIESONLY, GPS_READWRITE, IPropertyStore,
            },
            SHCNE_CREATE, SHCNE_DELETE, SHCNE_MKDIR,
            SHCreateItemFromParsingName,
        },
    },
    core::PCWSTR,
};
use windows_core::PROPVARIANT;

pub struct CrPlaceholder {
    pub local_file_info: LocalFileInfo,

    local_path: PathBuf,
    sync_root: PathBuf,
    drive_id: Uuid,
    file_meta: Option<FileMetadata>,
    options: u32,
}

enum CrPlaceholderOptions {
    InvalidateAllRange = 1 << 0,
    MarkNoChildren = 1 << 1,
}

impl CrPlaceholder {
    pub fn new(local_path: impl Into<PathBuf>, sync_root: PathBuf, drive_id: Uuid) -> Self {
        let local_path = local_path.into();
        Self {
            local_path: local_path.clone(),
            sync_root,
            drive_id,
            file_meta: None,
            options: 0,
            local_file_info: LocalFileInfo::from_path(&local_path.clone())
                .unwrap_or(LocalFileInfo::missing()),
        }
    }

    pub fn with_invalidate_all_range(mut self, enable: bool) -> Self {
        if enable {
            self.options |= CrPlaceholderOptions::InvalidateAllRange as u32;
        } else {
            self.options &= !(CrPlaceholderOptions::InvalidateAllRange as u32);
        }
        self
    }

    pub fn with_mark_no_children(mut self, enable: bool) -> Self {
        if enable {
            self.options |= CrPlaceholderOptions::MarkNoChildren as u32;
        } else {
            self.options &= !(CrPlaceholderOptions::MarkNoChildren as u32);
        }
        self
    }

    pub fn with_file_meta(mut self, file_meta: FileMetadata) -> Self {
        self.file_meta = Some(file_meta);
        self
    }

    pub fn delete_placeholder(&self, inventory: Arc<InventoryDb>) -> Result<()> {
        // Delete local file/folder if it exists
        if self.local_file_info.exists {
            if self.local_path.is_dir() {
                std::fs::remove_dir_all(&self.local_path)
                    .context("failed to delete local directory")?;
            } else {
                std::fs::remove_file(&self.local_path).context("failed to delete local file")?;
            }
        }

        // Remove from inventory
        let path_str = self
            .local_path
            .to_str()
            .context("failed to convert path to string")?;
        inventory
            .batch_delete_by_path(vec![path_str])
            .context("failed to delete from inventory")?;

        // Notify shell change
        notify_shell_change(&self.local_path, SHCNE_DELETE)
            .context("failed to notify shell change")?;

        Ok(())
    }

    // Commit changes to file system and inventory
    pub fn commit(&mut self, inventory: Arc<InventoryDb>) -> Result<()> {
        if self.file_meta.is_none() {
            return Err(anyhow::anyhow!("File metadata is not set"));
        }

        let file_meta = self.file_meta.as_ref().unwrap();

        if self.local_file_info.exists {
            if !self.local_file_info.is_placeholder() {
                let primary_entity = OsString::from(file_meta.etag.clone());
                let blob = primary_entity.into_encoded_bytes();
                // Upgrade to placeholder
                let mut local_handle = match self.local_file_info.is_directory {
                    true => OpenOptions::new()
                        .open(&self.local_path)
                        .context("failed to open local directory")?,
                    false => OpenOptions::new()
                        .open_win32(&self.local_path)
                        .context("failed to open local file")?,
                };
                tracing::info!(
                    target: "drive::placeholder",
                    local_path = %self.local_path.display(),
                    "Converting to placeholder"
                );
                local_handle
                    .convert_to_placeholder(
                        ConvertOptions::default().mark_in_sync().blob(blob),
                        None,
                    )
                    .context("failed to convert to placeholder")?;
            }

            // Update file metadata
            let mut upload_options = UpdateOptions::default().mark_in_sync().metadata(
                Metadata::default()
                    .size(file_meta.size as u64)
                    .changed(FileTime::from_unix_time(file_meta.updated_at)?)
                    .written(FileTime::from_unix_time(file_meta.updated_at)?)
                    .created(FileTime::from_unix_time(file_meta.created_at)?),
            );

            let dehydrate_requested =
                self.options & CrPlaceholderOptions::InvalidateAllRange as u32 != 0;
            let mut local_handle = if dehydrate_requested {
                OpenOptions::new()
                    .write_access()
                    .exclusive()
                    .open(&self.local_path)
                    .context("failed to open local placeholder for dehydration")?
            } else {
                match self.local_file_info.is_directory {
                    true => OpenOptions::new()
                        .open(&self.local_path)
                        .context("failed to open local placeholder directory")?,
                    false => OpenOptions::new()
                        .open_win32(&self.local_path)
                        .context("failed to open local placeholder file")?,
                }
            };
            if dehydrate_requested {
                tracing::debug!(target: "drive::placeholder", local_path = %self.local_path.display(), "Invalidating all range");
                upload_options = upload_options.dehydrate();
            }
            if self.options & CrPlaceholderOptions::MarkNoChildren as u32 != 0 {
                tracing::debug!(target: "drive::placeholder", local_path = %self.local_path.display(), "Marking no children");
                upload_options = upload_options.has_no_children();
            }
            local_handle
                .update(upload_options, None)
                .context("failed to invalidate all range")?;
        } else {
            // Create placeholder file/directory
            let relative_path = self
                .local_path
                .strip_prefix(&self.sync_root)
                .context("failed to get relative path")?;
            tracing::trace!(target: "drive::placeholder", relative_path = %relative_path.to_string_lossy(), "Relative path");
            let primary_entity = OsString::from(file_meta.etag.clone());
            let placeholder = PlaceholderFile::new(
                self.local_path
                    .file_name()
                    .context("failed to get file name")?,
            )
            .metadata(
                match file_meta.is_folder {
                    true => Metadata::directory(),
                    false => Metadata::file(),
                }
                .size(file_meta.size as u64)
                .changed(FileTime::from_unix_time(file_meta.updated_at)?)
                .written(FileTime::from_unix_time(file_meta.updated_at)?)
                .created(FileTime::from_unix_time(file_meta.created_at)?),
            )
            .mark_in_sync()
            .overwrite()
            .blob(primary_entity.into_encoded_bytes());
            let parent_path: &std::path::Path = self
                .local_path
                .parent()
                .ok_or(anyhow::anyhow!("failed to get parent path"))?;
            placeholder
                .create::<&std::path::Path>(parent_path)
                .context("failed to create placeholder")?;
        }

        // Upser inventory
        inventory
            .upsert(&MetadataEntry::from(file_meta))
            .context("failed to upsert inventory")?;

        // Notify shell change
        notify_shell_change(
            &self.local_path,
            if file_meta.is_folder {
                SHCNE_CREATE
            } else {
                SHCNE_MKDIR
            },
        )
        .context("failed to notify shell change")?;

        Ok(())
    }

    pub fn with_remote_file(mut self, file_info: &FileResponse) -> Self {
        // Parse RFC3339 time strings from Golang
        let created_at = DateTime::parse_from_rfc3339(&file_info.created_at)
            .ok()
            .map(|dt| dt.timestamp())
            .unwrap_or_default();

        let updated_at = DateTime::parse_from_rfc3339(&file_info.updated_at)
            .ok()
            .map(|dt| dt.timestamp())
            .unwrap_or_default();

        self.file_meta = Some(FileMetadata {
            drive_id: self.drive_id,
            local_path: self.local_path.to_string_lossy().to_string(),
            is_folder: file_info.file_type == file_type::FOLDER,
            created_at,
            updated_at,
            size: file_info.size,
            etag: file_info.primary_entity.clone().unwrap_or_default(),
            id: 0,
            metadata: file_info.metadata.clone().unwrap_or_default(),
            props: None,
            permissions: file_info.permission.clone().unwrap_or_default(),
            shared: file_info.shared.unwrap_or(false),
            conflict_state: None,
        });
        self
    }

    /// Updates the sync error state for a file or folder in Windows Explorer.
    ///
    /// This function sets or clears the `PKEY_LastSyncError` shell property,
    /// which controls the sync error overlay icon displayed in Explorer.
    ///
    /// # Arguments
    ///
    /// * `path` - The path to the file or folder
    /// * `set_error` - If true, sets the error state (shows error overlay);
    ///                 if false, clears the error state
    ///
    /// # Example
    ///
    /// ```no_run
    /// use std::path::Path;
    ///
    /// // Set error state on a file
    /// update_sync_error_state(Path::new("C:\\MyFolder\\file.txt"), true)?;
    ///
    /// // Clear error state
    /// update_sync_error_state(Path::new("C:\\MyFolder\\file.txt"), false)?;
    /// ```
    pub fn update_sync_error_state(&self, set_error: bool) -> Result<()> {
        if !self.local_file_info.is_placeholder() {
            // Skip non-placeholder file
            return Ok(());
        }
        let path_wide = U16CString::from_os_str(&self.local_path)
            .context("failed to convert path to wide string")?;

        unsafe {
            // Create a Shell Item from the file path
            let item: IShellItem2 = SHCreateItemFromParsingName(PCWSTR(path_wide.as_ptr()), None)
                .context("failed to create shell item from path")?;

            // Get the Property Store with read/write access for extrinsic properties
            let flags = GPS_READWRITE | GPS_EXTRINSICPROPERTIESONLY;
            let property_store: IPropertyStore = item
                .GetPropertyStore(flags)
                .context("failed to get property store")?;

            // Prepare the PROPVARIANT to set or clear the error state
            let prop_var = if set_error {
                // Set error state: VT_UI4 with E_FAIL value
                let mut pv = PROPVARIANT::default().as_raw().clone();
                pv.Anonymous.Anonymous.vt = VT_UI4.0;
                pv.Anonymous.Anonymous.Anonymous.ulVal = E_FAIL.0 as u32;
                PROPVARIANT::from_raw(pv)
            } else {
                let pv = PROPVARIANT::default().as_raw().clone();
                PROPVARIANT::from_raw(pv)
            };

            // Set the PKEY_LastSyncError property
            property_store
                .SetValue(&PKEY_LastSyncError, &prop_var)
                .context("failed to set PKEY_LastSyncError value")?;

            // Commit the changes
            property_store
                .Commit()
                .context("failed to commit property store changes")?;
        }

        tracing::debug!(
            target: "drive::placeholder",
            path = %self.local_path.display(),
            set_error,
            "Updated sync error state"
        );

        Ok(())
    }
}
