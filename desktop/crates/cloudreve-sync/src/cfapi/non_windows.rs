use std::{ops::RangeBounds, path::Path, time::SystemTime};

use anyhow::{Result, anyhow};

pub mod filter {
    pub mod ticket {
        #[derive(Debug)]
        pub struct FetchData;

        impl FetchData {
            pub fn write_at(&self, _data: &[u8], _offset: u64) -> anyhow::Result<()> {
                Err(anyhow::anyhow!(
                    "placeholder hydration is not supported on this platform"
                ))
            }

            pub fn report_progress(&self, _total: u64, _completed: u64) -> anyhow::Result<()> {
                Ok(())
            }
        }
    }
}

pub mod utility {
    pub trait WriteAt {}
}

pub mod root {
    use anyhow::{Result, anyhow};
    use serde::{Deserialize, Serialize};
    use std::{ffi::OsString, marker::PhantomData, path::Path};

    #[derive(Debug)]
    pub struct Connection<T>(PhantomData<T>);

    impl<T> Connection<T> {
        pub fn disconnect(&self) -> Result<()> {
            Ok(())
        }
    }

    #[derive(Debug, Clone, Serialize, Deserialize)]
    pub struct SyncRootId(String);

    impl SyncRootId {
        pub fn to_os_string(&self) -> OsString {
            OsString::from(self.0.clone())
        }

        pub fn is_registered(&self) -> Result<bool> {
            Ok(false)
        }

        pub fn register(&self, _info: SyncRootInfo) -> Result<()> {
            Err(anyhow!(
                "sync root registration is not supported on this platform"
            ))
        }

        pub fn unregister(&self) -> Result<()> {
            Ok(())
        }

        pub fn index(&self) -> Result<()> {
            Ok(())
        }
    }

    #[derive(Debug, Clone, Copy)]
    pub enum HydrationType {
        Full,
    }

    #[derive(Debug, Clone, Copy)]
    pub enum PopulationType {
        Full,
    }

    pub struct SecurityId;

    impl SecurityId {
        pub fn current_user() -> Result<Self> {
            Ok(Self)
        }
    }

    pub struct Session;

    impl Session {
        pub fn new() -> Self {
            Self
        }

        pub fn connect<T>(&self, _path: &Path, _callback: T) -> Result<Connection<T>> {
            Err(anyhow!(
                "placeholder sync roots are not supported on this platform"
            ))
        }
    }

    #[derive(Default)]
    pub struct SyncRootInfo;

    impl SyncRootInfo {
        pub fn set_display_name(&mut self, _name: String) {}
        pub fn set_hydration_type(&mut self, _hydration_type: HydrationType) {}
        pub fn set_population_type(&mut self, _population_type: PopulationType) {}
        pub fn set_icon(&mut self, _icon: String) {}
        pub fn set_version(&mut self, _version: &str) {}
        pub fn set_recycle_bin_uri(&mut self, _uri: String) -> Result<()> {
            Ok(())
        }
        pub fn set_path(&mut self, _path: &Path) -> Result<()> {
            Ok(())
        }
        pub fn add_custom_state(&mut self, _label: &str, _state: u32) -> Result<()> {
            Ok(())
        }
    }

    pub struct SyncRootIdBuilder {
        provider_name: String,
        account_name: Option<String>,
    }

    impl SyncRootIdBuilder {
        pub fn new(provider_name: String) -> Self {
            Self {
                provider_name,
                account_name: None,
            }
        }

        pub fn user_security_id(self, _security_id: SecurityId) -> Self {
            self
        }

        pub fn account_name(mut self, account_name: &str) -> Self {
            self.account_name = Some(account_name.to_string());
            self
        }

        pub fn build(self) -> SyncRootId {
            SyncRootId(format!(
                "{}!{}",
                self.provider_name,
                self.account_name.unwrap_or_default()
            ))
        }
    }
}

pub mod metadata {
    use nt_time::FileTime;

    #[derive(Clone, Default)]
    pub struct Metadata;

    impl Metadata {
        pub fn directory() -> Self {
            Self
        }

        pub fn file() -> Self {
            Self
        }

        pub fn size(self, _size: u64) -> Self {
            self
        }

        pub fn changed(self, _time: FileTime) -> Self {
            self
        }

        pub fn written(self, _time: FileTime) -> Self {
            self
        }

        pub fn created(self, _time: FileTime) -> Self {
            self
        }
    }
}

pub mod placeholder_file {
    use std::path::{Path, PathBuf};

    use anyhow::{Result, anyhow};

    use super::metadata::Metadata;

    pub struct PlaceholderFile {
        path: PathBuf,
    }

    impl PlaceholderFile {
        pub fn new(path: impl Into<PathBuf>) -> Self {
            Self { path: path.into() }
        }

        pub fn metadata(self, _metadata: Metadata) -> Self {
            self
        }

        pub fn mark_in_sync(self) -> Self {
            self
        }

        pub fn overwrite(self) -> Self {
            self
        }

        pub fn blob(self, _blob: Vec<u8>) -> Self {
            self
        }

        pub fn create<P: AsRef<Path>>(&self, _parent: P) -> Result<()> {
            Err(anyhow!(
                "placeholder creation is not supported on this platform: {}",
                self.path.display()
            ))
        }
    }
}

pub mod placeholder {
    use super::*;

    #[derive(Debug, Clone, Copy, PartialEq, Eq)]
    pub enum PinState {
        Unspecified,
        Pinned,
        Unpinned,
    }

    #[derive(Debug, Clone)]
    pub struct LocalFileInfo {
        pub exists: bool,
        pub is_directory: bool,
        pub file_size: Option<u64>,
        pub last_modified: Option<SystemTime>,
    }

    impl LocalFileInfo {
        pub fn from_path(path: impl AsRef<Path>) -> Result<Self> {
            match std::fs::metadata(path.as_ref()) {
                Ok(metadata) => Ok(Self {
                    exists: true,
                    is_directory: metadata.is_dir(),
                    file_size: Some(metadata.len()),
                    last_modified: metadata.modified().ok(),
                }),
                Err(err) if err.kind() == std::io::ErrorKind::NotFound => Ok(Self::missing()),
                Err(err) => Err(err.into()),
            }
        }

        pub fn missing() -> Self {
            Self {
                exists: false,
                is_directory: false,
                file_size: None,
                last_modified: None,
            }
        }

        pub fn is_placeholder(&self) -> bool {
            false
        }

        pub fn in_sync(&self) -> bool {
            false
        }

        pub fn is_directory(&self) -> bool {
            self.is_directory
        }

        pub fn partial_on_disk(&self) -> bool {
            false
        }

        pub fn is_folder_populated(&self) -> bool {
            self.exists && self.is_directory
        }

        pub fn pinned(&self) -> PinState {
            PinState::Pinned
        }

        pub fn sync_error_state(&self) -> bool {
            false
        }
    }

    #[derive(Default)]
    pub struct ConvertOptions;

    impl ConvertOptions {
        pub fn mark_in_sync(self) -> Self {
            self
        }

        pub fn blob(self, _blob: Vec<u8>) -> Self {
            self
        }
    }

    #[derive(Default)]
    pub struct UpdateOptions;

    impl UpdateOptions {
        pub fn mark_in_sync() -> Self {
            Self
        }

        pub fn metadata(self, _metadata: crate::cfapi::metadata::Metadata) -> Self {
            self
        }

        pub fn dehydrate(self) -> Self {
            self
        }

        pub fn has_no_children(self) -> Self {
            self
        }
    }

    pub struct OpenOptions;

    impl OpenOptions {
        pub fn new() -> Self {
            Self
        }

        pub fn write_access(self) -> Self {
            self
        }

        pub fn exclusive(self) -> Self {
            self
        }

        pub fn open(self, _path: impl AsRef<Path>) -> Result<PlaceholderHandle> {
            Ok(PlaceholderHandle)
        }

        pub fn open_win32(self, _path: impl AsRef<Path>) -> Result<PlaceholderHandle> {
            Ok(PlaceholderHandle)
        }

        pub async fn open_with_retry(self, _path: impl AsRef<Path>) -> Result<PlaceholderHandle> {
            Ok(PlaceholderHandle)
        }

        pub async fn open_win32_with_retry(
            self,
            _path: impl AsRef<Path>,
        ) -> Result<PlaceholderHandle> {
            Ok(PlaceholderHandle)
        }
    }

    pub struct PlaceholderHandle;

    impl PlaceholderHandle {
        pub fn convert_to_placeholder(
            &mut self,
            _options: ConvertOptions,
            _progress: Option<()>,
        ) -> Result<()> {
            Err(anyhow!(
                "placeholder conversion is not supported on this platform"
            ))
        }

        pub fn update(&mut self, _options: UpdateOptions, _progress: Option<()>) -> Result<()> {
            Ok(())
        }

        pub fn mark_in_sync(&mut self, _in_sync: bool, _progress: Option<()>) -> Result<()> {
            Ok(())
        }

        pub fn set_pin_state(&mut self, _pin_state: PinState, _progress: Option<()>) -> Result<()> {
            Ok(())
        }

        pub fn hydrate<R: RangeBounds<u64>>(&mut self, _range: R) -> Result<()> {
            Ok(())
        }

        pub fn dehydrate<R: RangeBounds<u64>>(&mut self, _range: R) -> Result<()> {
            Ok(())
        }
    }
}
