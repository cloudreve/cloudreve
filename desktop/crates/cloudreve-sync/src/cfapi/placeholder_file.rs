use std::{path::Path, ptr, slice};

use widestring::U16CString;
use windows::{
    Win32::{
        Foundation,
        Storage::CloudFilters::{self, CF_PLACEHOLDER_CREATE_INFO, CfCreatePlaceholders},
    },
    core::{self, PCWSTR},
};

use crate::cfapi::{metadata::Metadata, sealed, usn::Usn};

/// A builder for creating new placeholder files/directories.
#[derive(Debug)]
pub struct PlaceholderFile(CF_PLACEHOLDER_CREATE_INFO);

impl PlaceholderFile {
    /// Creates a new [PlaceholderFile].
    pub fn new(relative_path: impl AsRef<Path>) -> Self {
        Self(CF_PLACEHOLDER_CREATE_INFO {
            RelativeFileName: PCWSTR(
                U16CString::from_os_str(relative_path.as_ref())
                    .unwrap()
                    .into_raw(),
            ),
            Flags: CloudFilters::CF_PLACEHOLDER_CREATE_FLAG_NONE,
            Result: Foundation::S_FALSE,
            ..Default::default()
        })
    }

    /// Marks this [PlaceholderFile] as having no child placeholders on
    /// creation.
    ///
    /// Only applicable to placeholder directories.
    pub fn has_no_children(mut self) -> Self {
        self.0.Flags |= CloudFilters::CF_PLACEHOLDER_CREATE_FLAG_DISABLE_ON_DEMAND_POPULATION;
        self
    }

    /// Marks a placeholder as in sync.
    ///
    /// See also
    /// [SetInSyncState](https://learn.microsoft.com/en-us/windows/win32/api/cfapi/nf-cfapi-cfsetinsyncstate),
    /// [What does "In-Sync" Mean?](https://www.userfilesystem.com/programming/faq/#nav_whatdoesin-syncmean)
    pub fn mark_in_sync(mut self) -> Self {
        self.0.Flags |= CloudFilters::CF_PLACEHOLDER_CREATE_FLAG_MARK_IN_SYNC;
        self
    }

    /// Whether or not to overwrite an existing placeholder.
    pub fn overwrite(mut self) -> Self {
        self.0.Flags |= CloudFilters::CF_PLACEHOLDER_CREATE_FLAG_SUPERSEDE;
        self
    }

    /// Blocks this placeholder file from being dehydrated.
    ///
    /// This flag does not work on directories.
    pub fn block_dehydration(mut self) -> Self {
        self.0.Flags |= CloudFilters::CF_PLACEHOLDER_CREATE_FLAG_ALWAYS_FULL;
        self
    }

    /// The metadata for the [PlaceholderFile].
    pub fn metadata(mut self, metadata: Metadata) -> Self {
        self.0.FsMetadata = metadata.0;
        self
    }

    /// A buffer of bytes stored with the file that could be accessed through a
    /// [Request::file_blob][crate::filter::Request::file_blob] or
    /// [Placeholder::info][crate::placeholder::Placeholder::info].
    ///
    /// The buffer must not exceed
    /// [4KiB](https://microsoft.github.io/windows-docs-rs/doc/windows/Win32/Storage/CloudFilters/constant.CF_PLACEHOLDER_MAX_FILE_IDENTITY_LENGTH.html).
    pub fn blob(mut self, blob: Vec<u8>) -> Self {
        assert!(
            blob.len() <= CloudFilters::CF_PLACEHOLDER_MAX_FILE_IDENTITY_LENGTH as usize,
            "blob size must not exceed {} bytes, got {} bytes",
            CloudFilters::CF_PLACEHOLDER_MAX_FILE_IDENTITY_LENGTH,
            blob.len()
        );

        if blob.is_empty() {
            self.0.FileIdentity = ptr::null();
            self.0.FileIdentityLength = 0;
            return self;
        }

        let leaked_blob = Box::leak(blob.into_boxed_slice());
        self.0.FileIdentity = leaked_blob.as_ptr() as *const _;
        self.0.FileIdentityLength = leaked_blob.len() as _;

        self
    }

    pub fn result(&self) -> core::Result<Usn> {
        self.0.Result.ok().map(|_| self.0.CreateUsn as _)
    }

    /// Creates a placeholder file/directory on the file system.
    ///
    /// The value returned is the final [Usn] after the placeholder is created.
    ///
    /// It is recommended to use this function over
    /// [Placeholder][crate::placeholder::Placeholder::convert_to_placeholder] for efficiency purposes. If you
    /// need to create multiple placeholders, consider using [BatchCreate::create].
    ///
    /// If you need to create placeholders from the
    /// [SyncFilter::fetch_placeholders][crate::filter::SyncFilter::fetch_placeholders] callback,
    /// do not use this method. Instead, use
    /// [FetchPlaceholders::pass_with_placeholder][crate::filter::ticket::FetchPlaceholders::pass_with_placeholder].
    pub fn create<P: AsRef<Path>>(self, parent: impl AsRef<Path>) -> core::Result<Usn> {
        unsafe {
            CfCreatePlaceholders(
                PCWSTR(U16CString::from_os_str(parent.as_ref()).unwrap().as_ptr()),
                &mut [self.0],
                CloudFilters::CF_CREATE_FLAG_NONE,
                None,
            )?;
        }

        self.result()
    }
}

impl Drop for PlaceholderFile {
    fn drop(&mut self) {
        // Safety: `self.0.RelativeFileName.0` is a valid pointer to a valid UTF-16 string
        drop(unsafe { U16CString::from_ptr_str(self.0.RelativeFileName.0) });

        if !self.0.FileIdentity.is_null() {
            // Safety: `self.0.FileIdentity` is a valid pointer to a valid slice
            drop(unsafe {
                Box::from_raw(slice::from_raw_parts_mut(
                    self.0.FileIdentity as *mut u8,
                    self.0.FileIdentityLength as _,
                ))
            });
        }
    }
}

/// Creates multiple placeholder file/directories within the given path.
pub trait BatchCreate: sealed::Sealed {
    fn create<P: AsRef<Path>>(&mut self, path: P) -> core::Result<()>;
}

impl BatchCreate for [PlaceholderFile] {
    fn create<P: AsRef<Path>>(&mut self, path: P) -> core::Result<()> {
        unsafe {
            CfCreatePlaceholders(
                PCWSTR(U16CString::from_os_str(path.as_ref()).unwrap().as_ptr()),
                slice::from_raw_parts_mut(self.as_mut_ptr() as *mut _, self.len()),
                CloudFilters::CF_CREATE_FLAG_NONE,
                None,
            )
        }
    }
}

impl sealed::Sealed for [PlaceholderFile] {}
