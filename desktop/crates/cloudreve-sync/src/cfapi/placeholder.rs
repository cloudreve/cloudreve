use anyhow::{Result};
use std::{
    fmt::Debug,
    fs::File,
    mem::{self, MaybeUninit},
    ops::{Bound, Range, RangeBounds},
    os::windows::io::{AsRawHandle, FromRawHandle, IntoRawHandle, RawHandle},
    path::Path,
    ptr,
    time::{Duration, SystemTime},
};
use widestring::U16CString;
use windows::{
    Win32::{
        Foundation::{
            BOOL, CloseHandle, E_HANDLE, ERROR_FILE_NOT_FOUND, ERROR_NOT_A_CLOUD_FILE,
            ERROR_PATH_NOT_FOUND, FILETIME, HANDLE, INVALID_HANDLE_VALUE,
        },
        Storage::{
            CloudFilters::{
                self, CF_CONVERT_FLAGS, CF_FILE_RANGE, CF_FS_METADATA, CF_OPEN_FILE_FLAGS,
                CF_PIN_STATE, CF_PLACEHOLDER_RANGE_INFO_CLASS, CF_PLACEHOLDER_STANDARD_INFO,
                CF_PLACEHOLDER_STATE, CF_SET_PIN_FLAGS, CF_UPDATE_FLAGS, CfCloseHandle,
                CfConvertToPlaceholder, CfDehydratePlaceholder, CfGetPlaceholderInfo,
                CfGetPlaceholderRangeInfo,
                CfGetPlaceholderStateFromFindData, CfGetWin32HandleFromProtectedHandle,
                CfHydratePlaceholder, CfOpenFileWithOplock, CfReferenceProtectedHandle,
                CfReleaseProtectedHandle, CfRevertPlaceholder, CfSetInSyncState, CfSetPinState,
                CfUpdatePlaceholder,
            },
            FileSystem::{
                CreateFileW, FILE_ATTRIBUTE_DIRECTORY, FILE_ATTRIBUTE_PINNED, FILE_ATTRIBUTE_UNPINNED, FILE_FLAGS_AND_ATTRIBUTES, FILE_SHARE_MODE, FILE_SHARE_READ, FIND_FIRST_EX_FLAGS, FindClose, FindExInfoBasic, FindExSearchNameMatch, FindFirstFileExW, OPEN_EXISTING, WIN32_FIND_DATAW
            },
        },
    },
    core::{self, Error as WindowsError, PCWSTR},
};

use crate::cfapi::{metadata::Metadata, usn::Usn};

/// The type of handle that the placeholder file/directory owns.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum PlaceholderHandleType {
    /// A handle that was opened with [CfOpenFileWithOplock].
    CfApi,
    /// A handle that was opened with [CreateFileW][windows::Win32::Storage::FileSystem::CreateFileW] etc.
    Win32,
}

/// An owned handle to a placeholder file/directory.
///
/// This closes the handle on drop.
#[derive(Debug)]
pub struct OwnedPlaceholderHandle {
    handle_type: PlaceholderHandleType,
    handle: HANDLE,
}

impl OwnedPlaceholderHandle {
    /// Create a new [OwnedPlaceholderHandle] from a handle returned by [CfOpenFileWithOplock].
    ///
    /// # Safety
    ///
    /// The handle must be valid and owned by the caller.
    pub unsafe fn from_cfapi(handle: HANDLE) -> Self {
        Self {
            handle_type: PlaceholderHandleType::CfApi,
            handle,
        }
    }

    /// Create a new [OwnedPlaceholderHandle] from a handle returned by
    /// [CreateFile][windows::Win32::Storage::FileSystem::CreateFileW] etc.
    ///
    /// # Safety
    ///
    /// The handle must be valid and owned by the caller.
    pub unsafe fn from_win32(handle: HANDLE) -> Self {
        Self {
            handle_type: PlaceholderHandleType::Win32,
            handle,
        }
    }

    pub const fn handle(&self) -> HANDLE {
        self.handle
    }

    pub const fn handle_type(&self) -> PlaceholderHandleType {
        self.handle_type
    }
}

impl Drop for OwnedPlaceholderHandle {
    fn drop(&mut self) {
        match self.handle_type {
            PlaceholderHandleType::CfApi => unsafe {
                tracing::trace!(target: "cfapi::placeholder", "Drop cf handle: {:?}", self.handle);
                CfCloseHandle(self.handle)
            },
            PlaceholderHandleType::Win32 => unsafe {
                tracing::trace!(target: "cfapi::placeholder", "Drop win32 handle: {:?}", self.handle);
                _ = CloseHandle(self.handle);
            },
        }
    }
}

/// Safety: Windows handles are safe to send across threads
unsafe impl Send for OwnedPlaceholderHandle {}
/// Safety: Windows handles can be safely shared between threads (read operations are thread-safe)
unsafe impl Sync for OwnedPlaceholderHandle {}

/// Holds a Win32 handle from the protected handle.
///
/// The reference count will increase when the [ArcWin32Handle] is cloned
/// and decrease when the [ArcWin32Handle] is dropped.
pub struct ArcWin32Handle {
    win32_handle: HANDLE,
    protected_handle: HANDLE,
}

impl ArcWin32Handle {
    /// Win32 handle from the protected handle.
    pub fn handle(&self) -> HANDLE {
        self.win32_handle
    }
}

impl Clone for ArcWin32Handle {
    fn clone(&self) -> Self {
        if self.protected_handle != INVALID_HANDLE_VALUE {
            unsafe { CfReferenceProtectedHandle(self.protected_handle) };
        }

        Self {
            win32_handle: self.win32_handle,
            protected_handle: self.protected_handle,
        }
    }
}

impl AsRawHandle for ArcWin32Handle {
    fn as_raw_handle(&self) -> RawHandle {
        unsafe { mem::transmute(self.win32_handle) }
    }
}

impl Drop for ArcWin32Handle {
    fn drop(&mut self) {
        if self.protected_handle != INVALID_HANDLE_VALUE {
            tracing::trace!(target: "cfapi::placeholder", win32_handle = ?self.win32_handle, protected_handle = ?self.protected_handle, "Drop cf protected handle");
            unsafe { CfReleaseProtectedHandle(self.protected_handle) };
        }
    }
}

/// Safety: reference counted by syscall
unsafe impl Send for ArcWin32Handle {}
/// Safety: reference counted by syscall
unsafe impl Sync for ArcWin32Handle {}

/// Configuration for retry behavior when opening placeholder files.
#[derive(Debug, Clone)]
pub struct RetryConfig {
    /// Maximum number of retry attempts (default: 5)
    pub max_retries: u32,
    /// Base delay in milliseconds for exponential backoff (default: 500)
    pub base_delay_ms: u64,
}

impl Default for RetryConfig {
    fn default() -> Self {
        Self {
            max_retries: 5,
            base_delay_ms: 500,
        }
    }
}

impl RetryConfig {
    /// Create a new retry configuration with the specified parameters.
    pub fn new(max_retries: u32, base_delay_ms: u64) -> Self {
        Self {
            max_retries,
            base_delay_ms,
        }
    }

    /// Calculate the delay for a given attempt number (0-indexed).
    /// Uses exponential backoff: base_delay * 2^attempt
    fn delay_for_attempt(&self, attempt: u32) -> std::time::Duration {
        let delay_ms = self.base_delay_ms * (1 << attempt);
        std::time::Duration::from_millis(delay_ms)
    }
}

/// Options for opening a placeholder file/directory.
pub struct OpenOptions {
    flags: CF_OPEN_FILE_FLAGS,
    share_mode: FILE_SHARE_MODE,
}

impl OpenOptions {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn exclusive(mut self) -> Self {
        self.flags |= CloudFilters::CF_OPEN_FILE_FLAG_EXCLUSIVE;
        self
    }

    pub fn write_access(mut self) -> Self {
        self.flags |= CloudFilters::CF_OPEN_FILE_FLAG_WRITE_ACCESS;
        self
    }

    pub fn delete_access(mut self) -> Self {
        self.flags |= CloudFilters::CF_OPEN_FILE_FLAG_DELETE_ACCESS;
        self
    }

    pub fn foreground(mut self) -> Self {
        self.flags |= CloudFilters::CF_OPEN_FILE_FLAG_FOREGROUND;
        self
    }

    pub fn open_win32(self, path: impl AsRef<Path>) -> core::Result<Placeholder> {
        let u16_path = U16CString::from_os_str(path.as_ref()).unwrap();
        let handle = unsafe {
            CreateFileW(
                PCWSTR(u16_path.as_ptr()),
                0,
                self.share_mode,
                None,
                OPEN_EXISTING,
                FILE_FLAGS_AND_ATTRIBUTES(0),
                None,
            )
        }?;
        Ok(Placeholder {
            handle: unsafe { OwnedPlaceholderHandle::from_win32(handle) },
        })
    }

    /// Open the placeholder file/directory using `CfOpenFileWithOplock`.
    pub fn open(self, path: impl AsRef<Path>) -> core::Result<Placeholder> {
        let u16_path = U16CString::from_os_str(path.as_ref()).unwrap();
        let handle = unsafe { CfOpenFileWithOplock(PCWSTR(u16_path.as_ptr()), self.flags) }?;
        Ok(Placeholder {
            handle: unsafe { OwnedPlaceholderHandle::from_cfapi(handle) },
        })
    }

    /// Open the placeholder file/directory using `CreateFileW` with exponential backoff retry.
    ///
    /// This method will retry the open operation with exponential backoff if it fails,
    /// which is useful when files may be temporarily locked by other processes.
    ///
    /// Uses default retry configuration (5 retries, 500ms base delay).
    pub async fn open_win32_with_retry(self, path: impl AsRef<Path>) -> core::Result<Placeholder> {
        self.open_win32_with_retry_config(path, RetryConfig::default())
            .await
    }

    /// Open the placeholder file/directory using `CreateFileW` with configurable retry.
    ///
    /// This method will retry the open operation with exponential backoff if it fails,
    /// which is useful when files may be temporarily locked by other processes.
    pub async fn open_win32_with_retry_config(
        self,
        path: impl AsRef<Path>,
        config: RetryConfig,
    ) -> core::Result<Placeholder> {
        let path = path.as_ref();
        let u16_path = U16CString::from_os_str(path).unwrap();

        for attempt in 0..=config.max_retries {
            if attempt > 0 {
                let delay = config.delay_for_attempt(attempt - 1);
                tracing::debug!(
                    target: "cfapi::placeholder",
                    path = %path.display(),
                    attempt = attempt,
                    delay_ms = delay.as_millis(),
                    "Retrying open_win32 after delay"
                );
                tokio::time::sleep(delay).await;
            }

            let result = unsafe {
                CreateFileW(
                    PCWSTR(u16_path.as_ptr()),
                    0,
                    self.share_mode,
                    None,
                    OPEN_EXISTING,
                    FILE_FLAGS_AND_ATTRIBUTES(0),
                    None,
                )
            };

            match result {
                Ok(handle) => {
                    if attempt > 0 {
                        tracing::debug!(
                            target: "cfapi::placeholder",
                            path = %path.display(),
                            attempt = attempt,
                            "open_win32 succeeded after retry"
                        );
                    }
                    return Ok(Placeholder {
                        handle: unsafe { OwnedPlaceholderHandle::from_win32(handle) },
                    });
                }
                Err(e) => {
                    if attempt == config.max_retries {
                        tracing::warn!(
                            target: "cfapi::placeholder",
                            path = %path.display(),
                            error = %e,
                            attempts = config.max_retries + 1,
                            "open_win32 failed after all retries"
                        );
                        return Err(e);
                    }
                    tracing::trace!(
                        target: "cfapi::placeholder",
                        path = %path.display(),
                        error = %e,
                        attempt = attempt,
                        "open_win32 attempt failed, will retry"
                    );
                }
            }
        }

        // This should never be reached due to the return in the last iteration
        unreachable!()
    }

    /// Open the placeholder file/directory using `CfOpenFileWithOplock` with exponential backoff retry.
    ///
    /// This method will retry the open operation with exponential backoff if it fails,
    /// which is useful when files may be temporarily locked by other processes.
    ///
    /// Uses default retry configuration (5 retries, 500ms base delay).
    pub async fn open_with_retry(self, path: impl AsRef<Path>) -> core::Result<Placeholder> {
        self.open_with_retry_config(path, RetryConfig::default())
            .await
    }

    /// Open the placeholder file/directory using `CfOpenFileWithOplock` with configurable retry.
    ///
    /// This method will retry the open operation with exponential backoff if it fails,
    /// which is useful when files may be temporarily locked by other processes.
    pub async fn open_with_retry_config(
        self,
        path: impl AsRef<Path>,
        config: RetryConfig,
    ) -> core::Result<Placeholder> {
        let path = path.as_ref();
        let u16_path = U16CString::from_os_str(path).unwrap();

        for attempt in 0..=config.max_retries {
            if attempt > 0 {
                let delay = config.delay_for_attempt(attempt - 1);
                tracing::debug!(
                    target: "cfapi::placeholder",
                    path = %path.display(),
                    attempt = attempt,
                    delay_ms = delay.as_millis(),
                    "Retrying open after delay"
                );
                tokio::time::sleep(delay).await;
            }

            let result =
                unsafe { CfOpenFileWithOplock(PCWSTR(u16_path.as_ptr()), self.flags) };

            match result {
                Ok(handle) => {
                    if attempt > 0 {
                        tracing::debug!(
                            target: "cfapi::placeholder",
                            path = %path.display(),
                            attempt = attempt,
                            "open succeeded after retry"
                        );
                    }
                    return Ok(Placeholder {
                        handle: unsafe { OwnedPlaceholderHandle::from_cfapi(handle) },
                    });
                }
                Err(e) => {
                    if attempt == config.max_retries {
                        tracing::warn!(
                            target: "cfapi::placeholder",
                            path = %path.display(),
                            error = %e,
                            attempts = config.max_retries + 1,
                            "open failed after all retries"
                        );
                        return Err(e);
                    }
                    tracing::trace!(
                        target: "cfapi::placeholder",
                        path = %path.display(),
                        error = %e,
                        attempt = attempt,
                        "open attempt failed, will retry"
                    );
                }
            }
        }

        // This should never be reached due to the return in the last iteration
        unreachable!()
    }
}

impl Default for OpenOptions {
    fn default() -> Self {
        Self {
            flags: CloudFilters::CF_OPEN_FILE_FLAG_NONE,
            share_mode: FILE_SHARE_MODE(FILE_SHARE_READ.0),
        }
    }
}

/// The pin state of a placeholder.
///
/// [Read more
/// here](https://docs.microsoft.com/en-us/windows/win32/api/cfapi/ne-cfapi-cf_pin_state#remarks)
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum PinState {
    /// The platform could decide freely.
    Unspecified,
    /// [SyncFilter::fetch_data][crate::filter::SyncFilter::fetch_data] will be called to hydrate the rest
    /// of the placeholder's data. Any dehydration requests will fail automatically.
    Pinned,
    /// [SyncFilter::dehydrate][crate::filter::SyncFilter::dehydrate] will be called to dehydrate the rest
    /// of the placeholder's data.
    Unpinned,
    /// The placeholder will never sync to the cloud.
    Excluded,
    /// The placeholder will inherit the parent placeholder's pin state.
    Inherit,
}

impl From<PinState> for CF_PIN_STATE {
    fn from(state: PinState) -> Self {
        match state {
            PinState::Unspecified => CloudFilters::CF_PIN_STATE_UNSPECIFIED,
            PinState::Pinned => CloudFilters::CF_PIN_STATE_PINNED,
            PinState::Unpinned => CloudFilters::CF_PIN_STATE_UNPINNED,
            PinState::Excluded => CloudFilters::CF_PIN_STATE_EXCLUDED,
            PinState::Inherit => CloudFilters::CF_PIN_STATE_INHERIT,
        }
    }
}

impl From<CF_PIN_STATE> for PinState {
    fn from(state: CF_PIN_STATE) -> Self {
        match state {
            CloudFilters::CF_PIN_STATE_UNSPECIFIED => PinState::Unspecified,
            CloudFilters::CF_PIN_STATE_PINNED => PinState::Pinned,
            CloudFilters::CF_PIN_STATE_UNPINNED => PinState::Unpinned,
            CloudFilters::CF_PIN_STATE_EXCLUDED => PinState::Excluded,
            CloudFilters::CF_PIN_STATE_INHERIT => PinState::Inherit,
            _ => unreachable!(),
        }
    }
}

/// The placeholder pin flags.
#[derive(Debug, Clone, Copy)]
pub struct PinOptions(CF_SET_PIN_FLAGS);

impl PinOptions {
    /// Applies the pin state to all descendants of the placeholder (if the placeholder is a
    /// directory).
    pub fn recurse(&mut self) -> &mut Self {
        self.0 |= CloudFilters::CF_SET_PIN_FLAG_RECURSE;
        self
    }

    /// Applies the pin state to all descendants of the placeholder excluding the current one (if
    /// the placeholder is a directory).
    pub fn recurse_children(&mut self) -> &mut Self {
        self.0 |= CloudFilters::CF_SET_PIN_FLAG_RECURSE_ONLY;
        self
    }

    /// Stop applying the pin state when the first error is encountered. Otherwise, skip over it
    /// and keep applying.
    pub fn stop_on_error(&mut self) -> &mut Self {
        self.0 |= CloudFilters::CF_SET_PIN_FLAG_RECURSE_STOP_ON_ERROR;
        self
    }
}

impl Default for PinOptions {
    fn default() -> Self {
        Self(CloudFilters::CF_SET_PIN_FLAG_NONE)
    }
}

/// File to placeholder file conversion parameters.
#[derive(Debug, Clone)]
pub struct ConvertOptions {
    flags: CF_CONVERT_FLAGS,
    blob: Vec<u8>,
}

impl ConvertOptions {
    /// Marks a placeholder as in sync.
    ///
    /// See also
    /// [SetInSyncState](https://learn.microsoft.com/en-us/windows/win32/api/cfapi/nf-cfapi-cfsetinsyncstate),
    /// [What does "In-Sync" Mean?](https://www.userfilesystem.com/programming/faq/#nav_whatdoesin-syncmean)
    pub fn mark_in_sync(mut self) -> Self {
        self.flags |= CloudFilters::CF_CONVERT_FLAG_MARK_IN_SYNC;
        self
    }

    /// Dehydrate the placeholder after conversion.
    ///
    /// This flag is only applicable to files.
    pub fn dehydrate(mut self) -> Self {
        self.flags |= CloudFilters::CF_CONVERT_FLAG_DEHYDRATE;
        self
    }

    /// Marks the placeholder as "partially full," such that
    /// [SyncFilter::fetch_placeholders][crate::filter::SyncFilter::fetch_placeholders]
    /// will be invoked when this directory is next accessed so that the remaining placeholders are inserted.
    ///
    /// Only applicable to placeholder directories.
    pub fn has_children(mut self) -> Self {
        self.flags |= CloudFilters::CF_CONVERT_FLAG_ENABLE_ON_DEMAND_POPULATION;
        self
    }

    /// Blocks this placeholder from being dehydrated.
    ///
    /// This flag does not work on directories.
    pub fn block_dehydration(mut self) -> Self {
        self.flags |= CloudFilters::CF_CONVERT_FLAG_ALWAYS_FULL;
        self
    }

    /// Forces the conversion of a non-cloud placeholder file to a cloud placeholder file.
    ///
    /// Placeholder files are built into the NTFS file system and thus, a placeholder not associated
    /// with the sync root is possible.
    pub fn force(mut self) -> Self {
        self.flags |= CloudFilters::CF_CONVERT_FLAG_FORCE_CONVERT_TO_CLOUD_FILE;
        self
    }

    /// A buffer of bytes stored with the file that could be accessed through a
    /// [Request::file_blob][crate::filter::Request::file_blob] or [Placeholder::info].
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
        self.blob = blob;
        self
    }
}

impl Default for ConvertOptions {
    fn default() -> Self {
        Self {
            flags: CloudFilters::CF_CONVERT_FLAG_NONE,
            blob: Vec::new(),
        }
    }
}

#[allow(dead_code)]
fn should_suppress_placeholder_error(err: &WindowsError) -> bool {
    let code = err.code();
    code == ERROR_FILE_NOT_FOUND.to_hresult() || code == ERROR_PATH_NOT_FOUND.to_hresult()
}

#[derive(Debug, Clone)]
pub struct PlaceholderState {
    state: CF_PLACEHOLDER_STATE,
}

impl PlaceholderState {
    pub fn from_find_data(fdata: *const WIN32_FIND_DATAW) -> Self {
        Self {
            state: unsafe { CfGetPlaceholderStateFromFindData(fdata as *const _ as *const _) },
        }
    }

    // The file or directory whose FileAttributes and ReparseTag examined by the API is a placeholder.
    pub fn is_placeholder(&self) -> bool {
        (self.state & CloudFilters::CF_PLACEHOLDER_STATE_PLACEHOLDER).0 != 0
    }

    // The directory is both a placeholder directory as well as the sync root.
    pub fn sync_root(&self) -> bool {
        (self.state & CloudFilters::CF_PLACEHOLDER_STATE_SYNC_ROOT).0 != 0
    }

    pub fn essential_prop_present(&self) -> bool {
        (self.state & CloudFilters::CF_PLACEHOLDER_STATE_ESSENTIAL_PROP_PRESENT).0 != 0
    }

    // The file or directory must be a placeholder and its content in sync with the cloud.
    pub fn in_sync(&self) -> bool {
        (self.state & CloudFilters::CF_PLACEHOLDER_STATE_IN_SYNC).0 != 0
    }

    pub fn partial(&self) -> bool {
        (self.state & CloudFilters::CF_PLACEHOLDER_STATE_PARTIAL).0 != 0
    }

    // The file or directory must be a placeholder and its content is not fully present locally.
    pub fn partial_on_disk(&self) -> bool {
        (self.state & CloudFilters::CF_PLACEHOLDER_STATE_PARTIALLY_ON_DISK).0 != 0
    }
}

#[derive(Debug, Clone)]
pub struct LocalFileInfo {
    pub exists: bool,
    pub is_directory: bool,
    pub file_size: Option<u64>,
    pub last_modified: Option<SystemTime>,
    pub placeholder_state: Option<PlaceholderState>,
    pub pin_state: PinState,
}

impl LocalFileInfo {
    pub fn missing() -> Self {
        Self {
            exists: false,
            is_directory: false,
            file_size: None,
            last_modified: None,
            placeholder_state: None,
            pin_state: PinState::Unspecified,
        }
    }

    pub fn from_path(path: &Path) -> Result<Self> {
        let mut find_data = unsafe { std::mem::zeroed::<WIN32_FIND_DATAW>() };
        let u16_path = U16CString::from_os_str(path).unwrap();
        let handle = match unsafe {
            FindFirstFileExW(
                PCWSTR::from_raw(u16_path.as_ptr()),
                FindExInfoBasic,
                &mut find_data as *mut _ as *mut _,
                FindExSearchNameMatch,
                None,
                FIND_FIRST_EX_FLAGS(0),
            )
        } {
            Ok(handle) => handle,
            Err(err) => {
                tracing::debug!(target: "cfapi::placeholder", "error getting file info from path {}: {:?}", path.display(), err);
                return Ok(Self::missing());
            }
        };

        // Close the handle after use
        unsafe { let _ = FindClose(handle); };

        let pin_state = if find_data.dwFileAttributes & FILE_ATTRIBUTE_PINNED.0 != 0 {
            PinState::Pinned
        } else if find_data.dwFileAttributes & FILE_ATTRIBUTE_UNPINNED.0 != 0 {
            PinState::Unpinned
        } else {
            PinState::Unspecified
        };
        let is_directory = (find_data.dwFileAttributes & FILE_ATTRIBUTE_DIRECTORY.0) != 0;
        let file_size = (!is_directory)
            .then_some(((find_data.nFileSizeHigh as u64) << 32) | find_data.nFileSizeLow as u64);
        let last_modified = filetime_to_system_time(find_data.ftLastWriteTime);
        let placeholder_state = PlaceholderState::from_find_data(&find_data);

        Ok(Self {
            exists: true,
            is_directory,
            file_size,
            last_modified,
            placeholder_state: Some(placeholder_state),
            pin_state: pin_state,
        })
    }

    pub fn in_sync(&self) -> bool {
        self.placeholder_state
            .as_ref()
            .map(|state| state.in_sync())
            .unwrap_or(false)
    }

    pub fn is_placeholder(&self) -> bool {
        self.placeholder_state
            .as_ref()
            .map(|state: &PlaceholderState| state.is_placeholder())
            .unwrap_or(false)
    }

    pub fn partial_on_disk(&self) -> bool {
        self.placeholder_state
            .as_ref()
            .map(|state: &PlaceholderState| state.partial_on_disk())
            .unwrap_or(false)
    }

    pub fn pinned(&self) -> PinState {
        self.pin_state.clone()
    }

    pub fn is_directory(&self) -> bool {
        self.is_directory
    }

    pub fn is_folder_populated(&self) -> bool {
        self.is_placeholder() && self.is_directory && !self.partial_on_disk()
    }
}

fn filetime_to_system_time(filetime: FILETIME) -> Option<SystemTime> {
    const TICKS_PER_SECOND: u64 = 10_000_000;
    const SECONDS_BETWEEN_EPOCHS: u64 = 11_644_473_600;

    if filetime.dwLowDateTime == 0 && filetime.dwHighDateTime == 0 {
        return None;
    }

    let ticks = ((filetime.dwHighDateTime as u64) << 32) | (filetime.dwLowDateTime as u64);
    if ticks < SECONDS_BETWEEN_EPOCHS * TICKS_PER_SECOND {
        return None;
    }

    let secs_since_windows_epoch = ticks / TICKS_PER_SECOND;
    let subsecs_ticks = ticks % TICKS_PER_SECOND;
    let secs_since_unix_epoch = secs_since_windows_epoch - SECONDS_BETWEEN_EPOCHS;
    let nanos = subsecs_ticks * 100;

    Some(SystemTime::UNIX_EPOCH + Duration::new(secs_since_unix_epoch, nanos as u32))
}

#[derive(Clone)]
pub struct PlaceholderInfo {
    data: Vec<u8>,
    info: *const CF_PLACEHOLDER_STANDARD_INFO,
    // Additional file info from GetFileInformationByHandleEx
}

impl PlaceholderInfo {
    pub fn on_disk_data_size(&self) -> i64 {
        unsafe { &*self.info }.OnDiskDataSize
    }

    pub fn validated_data_size(&self) -> i64 {
        unsafe { &*self.info }.ValidatedDataSize
    }

    pub fn modified_data_size(&self) -> i64 {
        unsafe { &*self.info }.ModifiedDataSize
    }

    pub fn properties_size(&self) -> i64 {
        unsafe { &*self.info }.PropertiesSize
    }

    pub fn pin_state(&self) -> PinState {
        unsafe { &*self.info }.PinState.into()
    }

    pub fn is_in_sync(&self) -> bool {
        unsafe { &*self.info }.InSyncState == CloudFilters::CF_IN_SYNC_STATE_IN_SYNC
    }

    pub fn file_id(&self) -> i64 {
        unsafe { &*self.info }.FileId
    }

    pub fn sync_root_file_id(&self) -> i64 {
        unsafe { &*self.info }.SyncRootFileId
    }

    pub fn blob(&self) -> &[u8] {
        &self.data[mem::size_of::<CF_PLACEHOLDER_STANDARD_INFO>()..]
    }
}

impl std::fmt::Debug for PlaceholderInfo {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("PlaceholderInfo")
            .field("on_disk_data_size", &self.on_disk_data_size())
            .field("validated_data_size", &self.validated_data_size())
            .field("modified_data_size", &self.modified_data_size())
            .field("properties_size", &self.properties_size())
            .field("pin_state", &self.pin_state())
            .field("is_in_sync", &self.is_in_sync())
            .field("file_id", &self.file_id())
            .field("sync_root_file_id", &self.sync_root_file_id())
            .finish()
    }
}

/// Placeholder update parameters.
#[derive(Debug, Clone)]
pub struct UpdateOptions<'a> {
    metadata: Option<Metadata>,
    dehydrate_ranges: Vec<CF_FILE_RANGE>,
    flags: CF_UPDATE_FLAGS,
    blob: &'a [u8],
}

impl<'a> UpdateOptions<'a> {
    /// [Metadata] contains file system metadata about the placeholder to be updated.
    ///
    /// File size will be truncates to 0 if not specified, otherwise to the specified size.
    pub fn metadata(mut self, metadata: Metadata) -> Self {
        self.flags &= !(CloudFilters::CF_UPDATE_FLAG_PASSTHROUGH_FS_METADATA);
        self.metadata = Some(metadata);
        self
    }

    /// Fields in [Metadata] will be updated.
    pub fn metadata_all(mut self, metadata: Metadata) -> Self {
        self.flags |= CloudFilters::CF_UPDATE_FLAG_PASSTHROUGH_FS_METADATA;
        self.metadata = Some(metadata);
        self
    }

    /// Extended ranges to be dehydrated.
    ///
    /// All the offsets and lengths should be `PAGE_SIZE` aligned.
    /// Passing a single range with Offset `0` and Length `CF_EOF` will invalidate the entire file.
    /// This has the same effect as passing the flag `CF_UPDATE_FLAG_DEHYDRATE` instead
    pub fn dehydrate_ranges(mut self, ranges: impl IntoIterator<Item = Range<u64>>) -> Self {
        self.dehydrate_ranges
            .extend(ranges.into_iter().map(|r| CF_FILE_RANGE {
                StartingOffset: r.start as _,
                Length: (r.end - r.start) as _,
            }));
        self
    }

    /// The update will fail if the `IN_SYNC` attribute is not currently set on the placeholder.
    pub fn update_if_in_sync(mut self) -> Self {
        self.flags |= CloudFilters::CF_UPDATE_FLAG_VERIFY_IN_SYNC;
        self
    }

    /// Marks a placeholder as in sync.
    ///
    /// See also
    /// [SetInSyncState](https://learn.microsoft.com/en-us/windows/win32/api/cfapi/nf-cfapi-cfsetinsyncstate),
    /// [What does "In-Sync" Mean?](https://www.userfilesystem.com/programming/faq/#nav_whatdoesin-syncmean)
    pub fn mark_in_sync(mut self) -> Self {
        self.flags |= CloudFilters::CF_UPDATE_FLAG_MARK_IN_SYNC;
        self
    }

    /// Marks a placeholder as not in sync. `Sync Pending` will be shown in explorer.
    ///
    /// See also
    /// [SetInSyncState](https://learn.microsoft.com/en-us/windows/win32/api/cfapi/nf-cfapi-cfsetinsyncstate),
    /// [What does "In-Sync" Mean?](https://www.userfilesystem.com/programming/faq/#nav_whatdoesin-syncmean)
    pub fn mark_not_in_sync(mut self) -> Self {
        self.flags |= CloudFilters::CF_UPDATE_FLAG_CLEAR_IN_SYNC;
        self
    }

    /// The platform dehydrates the file after updating the placeholder successfully.
    pub fn dehydrate(mut self) -> Self {
        self.flags |= CloudFilters::CF_UPDATE_FLAG_DEHYDRATE;
        self
    }

    /// Disables on-demand population for directories.
    pub fn has_no_children(mut self) -> Self {
        self.flags |= CloudFilters::CF_UPDATE_FLAG_DISABLE_ON_DEMAND_POPULATION;
        self
    }

    /// Enable on-demand population for directories.
    pub fn has_children(mut self) -> Self {
        self.flags |= CloudFilters::CF_UPDATE_FLAG_ENABLE_ON_DEMAND_POPULATION;
        self
    }

    /// Remove the file identity from the placeholder.
    /// [UpdateOptions::blob()](crate::placeholder::UpdateOptions::blob) will be ignored.
    pub fn remove_blob(mut self) -> Self {
        self.flags |= CloudFilters::CF_UPDATE_FLAG_REMOVE_FILE_IDENTITY;
        self
    }

    /// The platform removes all existing extrinsic properties on the placeholder.
    pub fn remove_properties(mut self) -> Self {
        self.flags |= CloudFilters::CF_UPDATE_FLAG_REMOVE_PROPERTY;
        self
    }

    pub fn blob(mut self, blob: &'a [u8]) -> Self {
        assert!(
            blob.len() <= CloudFilters::CF_PLACEHOLDER_MAX_FILE_IDENTITY_LENGTH as usize,
            "blob size must not exceed {} bytes, got {} bytes",
            CloudFilters::CF_PLACEHOLDER_MAX_FILE_IDENTITY_LENGTH,
            blob.len()
        );
        self.blob = blob;
        self
    }
}

impl Default for UpdateOptions<'_> {
    fn default() -> Self {
        Self {
            metadata: None,
            dehydrate_ranges: Vec::new(),
            flags: CloudFilters::CF_UPDATE_FLAG_NONE,
            blob: &[],
        }
    }
}

/// The type of data to read from a placeholder.
#[derive(Debug, Copy, Clone, PartialEq, Eq)]
pub enum ReadType {
    /// Any data that is saved to the disk.
    Any,
    /// Data that has been synced to the cloud.
    Validated,
    /// Data that has not synced to the cloud.
    Modified,
}

impl From<ReadType> for CF_PLACEHOLDER_RANGE_INFO_CLASS {
    fn from(read_type: ReadType) -> Self {
        match read_type {
            ReadType::Any => CloudFilters::CF_PLACEHOLDER_RANGE_INFO_ONDISK,
            ReadType::Validated => CloudFilters::CF_PLACEHOLDER_RANGE_INFO_VALIDATED,
            ReadType::Modified => CloudFilters::CF_PLACEHOLDER_RANGE_INFO_MODIFIED,
        }
    }
}

// #[derive(Clone, Copy)]
// pub struct PlaceholderState(CF_PLACEHOLDER_STATE);

// impl PlaceholderState {
//     /// The placeholder is both a directory as well as the sync root.
//     pub fn sync_root(&self) -> bool {
//         (self.0 & CloudFilters::CF_PLACEHOLDER_STATE_SYNC_ROOT).0 != 0
//     }

//     /// There exists an essential property in the property store of the file or directory.
//     pub fn essential_prop_present(&self) -> bool {
//         (self.0 & CloudFilters::CF_PLACEHOLDER_STATE_ESSENTIAL_PROP_PRESENT).0 != 0
//     }

//     /// The placeholder is in sync.
//     pub fn in_sync(&self) -> bool {
//         (self.0 & CloudFilters::CF_PLACEHOLDER_STATE_IN_SYNC).0 != 0
//     }

//     /// The placeholder content is not ready to be consumed by the user application,
//     /// though it may or may not be fully present locally.
//     ///
//     /// An example is a placeholder file whose content has been fully downloaded to the local disk,
//     /// but is yet to be validated by a sync provider that
//     /// has registered the sync root with the hydration modifier
//     /// [HydrationPolicy::require_validation][crate::root::HydrationPolicy::require_validation].
//     pub fn partial(&self) -> bool {
//         (self.0 & CloudFilters::CF_PLACEHOLDER_STATE_PARTIAL).0 != 0
//     }

//     /// The placeholder content is not fully present locally.
//     ///
//     /// When this is set, [PlaceholderState::partial] also be `true`.
//     pub fn partial_on_disk(&self) -> bool {
//         (self.0 & CloudFilters::CF_PLACEHOLDER_STATE_PARTIALLY_ON_DISK).0 != 0
//     }
// }

// impl Debug for PlaceholderState {
//     fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
//         f.debug_struct("PlaceholderState")
//             .field("sync_root", &self.sync_root())
//             .field("essential_prop_present", &self.essential_prop_present())
//             .field("in_sync", &self.in_sync())
//             .field("partial", &self.partial())
//             .field("partial_on_disk", &self.partial_on_disk())
//             .finish()
//     }
// }

/// A struct to perform various operations on a placeholder(or regular) file/directory.
#[derive(Debug)]
pub struct Placeholder {
    handle: OwnedPlaceholderHandle,
}

/// Safety: Placeholder only contains OwnedPlaceholderHandle which is Send
unsafe impl Send for Placeholder {}
/// Safety: Placeholder only contains OwnedPlaceholderHandle which is Sync
unsafe impl Sync for Placeholder {}

impl Placeholder {
    /// Create a placeholder from a raw handle.
    ///
    /// # Safety
    ///
    /// The passed handle must be a valid protected handle or win32 handle.
    pub unsafe fn from_raw_handle(handle: OwnedPlaceholderHandle) -> Self {
        Self { handle }
    }

    /// Open options for opening [Placeholder]s.
    pub fn options() -> OpenOptions {
        OpenOptions::default()
    }

    /// Open the placeholder file/directory with `CF_OPEN_FILE_FLAG_NONE`.
    pub fn open(path: impl AsRef<Path>) -> core::Result<Self> {
        OpenOptions::new().open(path)
    }

    /// Marks a placeholder as in sync or not.
    ///
    /// If the passed [Usn] is outdated, the call will fail,
    /// otherwise the [Usn] will be updated.
    ///
    /// See also
    /// [SetInSyncState](https://learn.microsoft.com/en-us/windows/win32/api/cfapi/nf-cfapi-cfsetinsyncstate),
    /// [What does "In-Sync" Mean?](https://www.userfilesystem.com/programming/faq/#nav_whatdoesin-syncmean)
    pub fn mark_in_sync<'a>(
        &mut self,
        in_sync: bool,
        usn: impl Into<Option<&'a mut Usn>>,
    ) -> core::Result<&mut Self> {
        unsafe {
            CfSetInSyncState(
                self.handle.handle,
                match in_sync {
                    true => CloudFilters::CF_IN_SYNC_STATE_IN_SYNC,
                    false => CloudFilters::CF_IN_SYNC_STATE_NOT_IN_SYNC,
                },
                CloudFilters::CF_SET_IN_SYNC_FLAG_NONE,
                usn.into().map(|x| ptr::read(x) as *mut _),
            )
        }?;

        Ok(self)
    }

    /// Sets the pin state of the placeholder.
    ///
    /// See also
    /// [CfSetPinState](https://learn.microsoft.com/en-us/windows/win32/api/cfapi/nf-cfapi-cfsetpinstate),
    /// [What does "Pinned" Mean?](https://www.userfilesystem.com/programming/faq/#nav_howdoesthealwayskeeponthisdevicemenuworks)
    pub fn mark_pin(&mut self, state: PinState, options: PinOptions) -> core::Result<&mut Self> {
        unsafe { CfSetPinState(self.handle.handle, state.into(), options.0, None) }?;
        Ok(self)
    }

    /// Converts a file to a placeholder file.
    ///
    /// If the passed [Usn] is outdated, the call will fail,
    /// otherwise the [Usn] will be updated.
    ///
    /// See also [CfConvertToPlaceholder](https://learn.microsoft.com/en-us/windows/win32/api/cfapi/nf-cfapi-cfconverttoplaceholder).
    pub fn convert_to_placeholder<'a>(
        &mut self,
        options: ConvertOptions,
        usn: impl Into<Option<&'a mut Usn>>,
    ) -> core::Result<&mut Self> {
        unsafe {
            CfConvertToPlaceholder(
                self.handle.handle,
                (!options.blob.is_empty()).then_some(options.blob.as_ptr() as *const _),
                options.blob.len() as _,
                options.flags,
                usn.into().map(|x| ptr::read(x) as *mut _),
                None,
            )
        }?;

        Ok(self)
    }

    /// Gets various characteristics of the placeholder.
    ///
    /// Returns [None] if the handle not points to a placeholder.
    ///
    /// If the placeholder blob size is known, use [fixed_size_info](Self::fixed_size_info) instead.
    pub fn info(&self) -> core::Result<Option<PlaceholderInfo>> {
        let mut info_size = 0;
        let mut data = vec![0u8; mem::size_of::<CF_PLACEHOLDER_STANDARD_INFO>() + 4096];
        let r = unsafe {
            CfGetPlaceholderInfo(
                self.handle.handle,
                CloudFilters::CF_PLACEHOLDER_INFO_STANDARD,
                data.as_mut_ptr() as *mut _,
                data.len() as _,
                Some(&mut info_size),
            )
        };

        match r {
            Ok(()) => {
                unsafe { data.set_len(info_size as _) };
                data.shrink_to_fit();

                Ok(Some(PlaceholderInfo {
                    info: &unsafe {
                        data[..=mem::size_of::<CF_PLACEHOLDER_STANDARD_INFO>()]
                            .align_to::<CF_PLACEHOLDER_STANDARD_INFO>()
                    }
                    .1[0] as *const _,
                    data,
                }))
            }
            Err(e) if e.code() == ERROR_NOT_A_CLOUD_FILE.to_hresult() => Ok(None),
            Err(e) => Err(e),
        }
    }

    /// Gets various characteristics of the placeholder.
    ///
    /// If the `blob_size` not matches the actual size of the blob,
    /// the call will returns `HRESULT_FROM_WIN32(ERROR_MORE_DATA)`.
    /// Returns [None] if the handle not points to a placeholder.
    pub fn fixed_size_info(&self, blob_size: usize) -> core::Result<Option<PlaceholderInfo>> {
        let mut data = vec![0; mem::size_of::<CF_PLACEHOLDER_STANDARD_INFO>() + blob_size];

        let r = unsafe {
            CfGetPlaceholderInfo(
                self.handle.handle,
                CloudFilters::CF_PLACEHOLDER_INFO_STANDARD,
                data.as_mut_ptr() as *mut _,
                data.len() as u32,
                None,
            )
        };

        match r {
            Ok(()) => Ok(Some(PlaceholderInfo {
                info: &unsafe {
                    data[..=mem::size_of::<CF_PLACEHOLDER_STANDARD_INFO>()]
                        .align_to::<CF_PLACEHOLDER_STANDARD_INFO>()
                }
                .1[0] as *const _,
                data,
            })),
            Err(e) if e.code() == ERROR_NOT_A_CLOUD_FILE.to_hresult() => Ok(None),
            Err(e) => Err(e),
        }
    }

    /// Updates various characteristics of a placeholder.
    ///
    /// See also [CfUpdatePlaceholder](https://learn.microsoft.com/en-us/windows/win32/api/cfapi/nf-cfapi-cfupdateplaceholder).
    pub fn update<'a>(
        &mut self,
        options: UpdateOptions,
        usn: impl Into<Option<&'a mut Usn>>,
    ) -> core::Result<&mut Self> {
        unsafe {
            CfUpdatePlaceholder(
                self.handle.handle,
                options
                    .metadata
                    .as_ref()
                    .map(|x| &x.0 as *const CF_FS_METADATA),
                (!options.blob.is_empty()).then_some(options.blob.as_ptr() as *const _),
                options.blob.len() as _,
                (options.dehydrate_ranges.is_empty()).then_some(&options.dehydrate_ranges),
                options.flags,
                usn.into().map(|u| u as *mut _),
                None,
            )
        }?;

        Ok(self)
    }

    /// Retrieves data from a placeholder.
    pub fn retrieve_data(
        &self,
        read_type: ReadType,
        offset: u64,
        buffer: &mut [u8],
    ) -> core::Result<u32> {
        let mut length = MaybeUninit::zeroed();
        unsafe {
            CfGetPlaceholderRangeInfo(
                self.handle.handle,
                read_type.into(),
                offset as i64,
                buffer.len() as i64,
                buffer as *mut _ as *mut _,
                buffer.len() as u32,
                Some(length.as_mut_ptr()),
            )
            .map(|_| length.assume_init())
        }
    }

    // FIXME: This function is not work at all, the CF_PLACEHOLDER_STATE always be 0 or 1
    // pub fn state(&self) -> core::Result<Option<PlaceholderState>> {
    //     let mut info = MaybeUninit::<FILE_ATTRIBUTE_TAG_INFO>::zeroed();
    //     let win32_handle = self.win32_handle()?;
    //     let state = unsafe {
    //         GetFileInformationByHandleEx(
    //             win32_handle.win32_handle,
    //             FileSystem::FileAttributeTagInfo,
    //             info.as_mut_ptr() as *mut _,
    //             mem::size_of::<FILE_ATTRIBUTE_TAG_INFO>() as u32,
    //         )
    //         .ok()
    //         .inspect_err(|e| println!("GetFileInformationByHandleEx: {e:#?}"))?;

    //         CfGetPlaceholderStateFromFileInfo(
    //             info.assume_init_ref() as *const _ as *const _,
    //             FileSystem::FileAttributeTagInfo,
    //         )
    //     };

    //     match state {
    //         CloudFilters::CF_PLACEHOLDER_STATE_INVALID => Err(core::Error::from_win32()),
    //         CloudFilters::CF_PLACEHOLDER_STATE_NO_STATES => Ok(None),
    //         s => Ok(Some(PlaceholderState(s))),
    //     }
    // }

    /// Returns the Win32 handle from protected handle.
    ///
    /// Returns `Err(E_HANDLE)` if the [OwnedPlaceholderHandle::handle_type] is not [PlaceholderHandleType::CfApi].
    pub fn win32_handle(&self) -> core::Result<ArcWin32Handle> {
        let (handle, win32_handle) = match self.handle.handle_type {
            PlaceholderHandleType::CfApi => {
                let win32_handle = unsafe {
                    CfReferenceProtectedHandle(self.handle.handle).ok()?;
                    CfGetWin32HandleFromProtectedHandle(self.handle.handle)
                };
                BOOL::from(!win32_handle.is_invalid()).ok()?;
                (self.handle.handle, win32_handle)
            }
            PlaceholderHandleType::Win32 => Err(core::Error::from(E_HANDLE))?,
        };

        Ok(ArcWin32Handle {
            win32_handle,
            protected_handle: handle,
        })
    }

    /// Returns the owned placeholder handle.
    pub fn inner_handle(&self) -> &OwnedPlaceholderHandle {
        &self.handle
    }

    /// Hydrates a placeholder file by ensuring that the specified byte range is present on-disk
    /// in the placeholder. This is valid for files only.
    ///
    /// # Panics
    ///
    /// Panics if the start bound is greater than [i64::MAX] or
    /// the end bound sub start bound is greater than [i64::MAX].
    ///
    /// See also [CfHydratePlaceholder](https://learn.microsoft.com/en-us/windows/win32/api/cfapi/nf-cfapi-cfhydrateplaceholder)
    /// and [discussion](https://docs.microsoft.com/en-us/windows/win32/api/cfapi/nf-cfapi-cfhydrateplaceholder#remarks).
    pub fn hydrate(&mut self, range: impl RangeBounds<u64>) -> core::Result<()> {
        unsafe {
            CfHydratePlaceholder(
                self.handle.handle,
                match range.start_bound() {
                    Bound::Included(x) => (*x).try_into().unwrap(),
                    Bound::Excluded(x) => (x + 1).try_into().unwrap(),
                    Bound::Unbounded => 0,
                },
                match range.end_bound() {
                    Bound::Included(x) => (*x).try_into().unwrap(),
                    Bound::Excluded(x) => (x - 1).try_into().unwrap(),
                    Bound::Unbounded => -1,
                },
                CloudFilters::CF_HYDRATE_FLAG_NONE,
                None,
            )
        }
    }

    pub fn dehydrate(&mut self, range: impl RangeBounds<u64>) -> core::Result<()> {
        unsafe {
            CfDehydratePlaceholder(
                self.handle.handle,
                match range.start_bound() {
                    Bound::Included(x) => (*x).try_into().unwrap(),
                    Bound::Excluded(x) => (x + 1).try_into().unwrap(),
                    Bound::Unbounded => 0,
                },
                match range.end_bound() {
                    Bound::Included(x) => (*x).try_into().unwrap(),
                    Bound::Excluded(x) => (x - 1).try_into().unwrap(),
                    Bound::Unbounded => -1,
                },
                CloudFilters::CF_DEHYDRATE_FLAG_NONE,
                None,
            )
        }
    }
}

impl From<File> for Placeholder {
    fn from(file: File) -> Self {
        Self {
            handle: unsafe {
                OwnedPlaceholderHandle::from_win32(HANDLE(file.into_raw_handle() as _))
            },
        }
    }
}

impl TryFrom<Placeholder> for File {
    type Error = core::Error;

    #[allow(clippy::missing_transmute_annotations)]
    fn try_from(placeholder: Placeholder) -> core::Result<Self> {
        match placeholder.handle.handle_type {
            PlaceholderHandleType::Win32 => {
                let file =
                    unsafe { File::from_raw_handle(mem::transmute(placeholder.handle.handle)) };
                Ok(file)
            }
            PlaceholderHandleType::CfApi => unsafe {
                CfRevertPlaceholder(
                    placeholder.handle.handle,
                    CloudFilters::CF_REVERT_FLAG_NONE,
                    None,
                )
            }
            .map(|_| unsafe { File::from_raw_handle(mem::transmute(placeholder.handle.handle)) }),
        }
    }
}
