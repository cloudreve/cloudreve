use std::fs;

use nt_time::FileTime;
use windows::Win32::Storage::{
    CloudFilters::CF_FS_METADATA,
    FileSystem::{FILE_ATTRIBUTE_DIRECTORY, FILE_ATTRIBUTE_NORMAL, FILE_BASIC_INFO},
};

use crate::cfapi::sealed;

/// The metadata for placeholder.
#[derive(Debug, Clone, Copy, Default)]
pub struct Metadata(pub(crate) CF_FS_METADATA);

impl Metadata {
    /// The default [Metadata] with `FILE_ATTRIBUTE_NORMAL` attribute.
    pub fn file() -> Self {
        Self(CF_FS_METADATA {
            BasicInfo: FILE_BASIC_INFO {
                FileAttributes: FILE_ATTRIBUTE_NORMAL.0,
                ..Default::default()
            },
            ..Default::default()
        })
    }

    /// The default [Metadata] with `FILE_ATTRIBUTE_DIRECTORY` attribute.
    pub fn directory() -> Self {
        Self(CF_FS_METADATA {
            BasicInfo: FILE_BASIC_INFO {
                FileAttributes: FILE_ATTRIBUTE_DIRECTORY.0,
                ..Default::default()
            },
            ..Default::default()
        })
    }

    /// The time the file/directory was created.
    pub fn created(mut self, time: FileTime) -> Self {
        self.0.BasicInfo.CreationTime = time.try_into().unwrap();
        self
    }

    /// The time the file/directory was last accessed.
    pub fn accessed(mut self, time: FileTime) -> Self {
        self.0.BasicInfo.LastAccessTime = time.try_into().unwrap();
        self
    }

    /// The time the file/directory content was last written.
    pub fn written(mut self, time: FileTime) -> Self {
        self.0.BasicInfo.LastWriteTime = time.try_into().unwrap();
        self
    }

    /// The time the file/directory content or metadata was changed.
    pub fn changed(mut self, time: FileTime) -> Self {
        self.0.BasicInfo.ChangeTime = time.try_into().unwrap();
        self
    }

    /// The size of the file's content.
    pub fn size(mut self, size: u64) -> Self {
        self.0.FileSize = size as i64;
        self
    }

    /// File attributes.
    pub fn attributes(mut self, attributes: u32) -> Self {
        self.0.BasicInfo.FileAttributes |= attributes;
        self
    }
}

pub trait MetadataExt: sealed::Sealed {
    /// The time the file was changed in
    /// [FILETIME](https://learn.microsoft.com/en-us/windows/win32/api/minwinbase/ns-minwinbase-filetime) format.
    fn change_time(self, time: i64) -> Self;

    /// The time the file was last accessed in
    /// [FILETIME](https://learn.microsoft.com/en-us/windows/win32/api/minwinbase/ns-minwinbase-filetime) format.
    fn last_access_time(self, time: i64) -> Self;

    /// The time the file was last written to in
    /// [FILETIME](https://learn.microsoft.com/en-us/windows/win32/api/minwinbase/ns-minwinbase-filetime) format.
    fn last_write_time(self, time: i64) -> Self;

    /// The time the file was created in
    /// [FILETIME](https://learn.microsoft.com/en-us/windows/win32/api/minwinbase/ns-minwinbase-filetime) format.
    fn creation_time(self, time: i64) -> Self;
}

impl MetadataExt for Metadata {
    fn change_time(mut self, time: i64) -> Self {
        self.0.BasicInfo.ChangeTime = time;
        self
    }

    fn last_access_time(mut self, time: i64) -> Self {
        self.0.BasicInfo.LastAccessTime = time;
        self
    }

    fn last_write_time(mut self, time: i64) -> Self {
        self.0.BasicInfo.LastWriteTime = time;
        self
    }

    fn creation_time(mut self, time: i64) -> Self {
        self.0.BasicInfo.CreationTime = time;
        self
    }
}

impl sealed::Sealed for Metadata {}

impl From<fs::Metadata> for Metadata {
    fn from(metadata: fs::Metadata) -> Self {
        use std::os::windows::fs::MetadataExt;
        Self(CF_FS_METADATA {
            BasicInfo: FILE_BASIC_INFO {
                CreationTime: metadata.creation_time() as i64,
                LastAccessTime: metadata.last_access_time() as i64,
                LastWriteTime: metadata.last_write_time() as i64,
                ChangeTime: metadata.last_write_time() as i64,
                FileAttributes: metadata.file_attributes(),
            },
            FileSize: metadata.file_size() as i64,
        })
    }
}
