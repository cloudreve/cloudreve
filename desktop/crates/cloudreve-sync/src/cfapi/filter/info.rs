use std::{ffi::OsString, fmt::Debug, ops::Range, path::PathBuf};

use nt_time::FileTime;
use widestring::U16CStr;
use windows::Win32::Storage::CloudFilters::{
    self, CF_CALLBACK_DEHYDRATION_REASON, CF_CALLBACK_PARAMETERS_0_0, CF_CALLBACK_PARAMETERS_0_1,
    CF_CALLBACK_PARAMETERS_0_2, CF_CALLBACK_PARAMETERS_0_3, CF_CALLBACK_PARAMETERS_0_4,
    CF_CALLBACK_PARAMETERS_0_5, CF_CALLBACK_PARAMETERS_0_6, CF_CALLBACK_PARAMETERS_0_7,
    CF_CALLBACK_PARAMETERS_0_8, CF_CALLBACK_PARAMETERS_0_9, CF_CALLBACK_PARAMETERS_0_10,
    CF_CALLBACK_PARAMETERS_0_11,
};

/// Information for the [SyncFilter::fetch_data][crate::filter::SyncFilter::fetch_data] callback.
pub struct FetchData(pub(crate) CF_CALLBACK_PARAMETERS_0_6);

impl FetchData {
    /// Whether or not the callback was called from an interrupted hydration.
    pub fn interrupted_hydration(&self) -> bool {
        (self.0.Flags & CloudFilters::CF_CALLBACK_FETCH_DATA_FLAG_RECOVERY).0 != 0
    }

    /// Whether or not the callback was called from an explicit hydration via
    /// [Placeholder::hydrate][crate::placeholder::Placeholder::hydrate].
    pub fn explicit_hydration(&self) -> bool {
        (self.0.Flags & CloudFilters::CF_CALLBACK_FETCH_DATA_FLAG_EXPLICIT_HYDRATION).0 != 0
    }

    /// The amount of bytes that must be written to the placeholder.
    pub fn required_file_range(&self) -> Range<u64> {
        (self.0.RequiredFileOffset as u64)
            ..(self.0.RequiredFileOffset + self.0.RequiredLength) as u64
    }

    /// The amount of bytes that must be written to the placeholder.
    ///
    /// If the sync provider prefer to give data in larger chunks, use this range instead.
    ///
    /// [Discussion](https://docs.microsoft.com/en-us/answers/questions/748214/what-is-fetchdataoptionalfileoffset-cfapi.html).
    pub fn optional_file_range(&self) -> Range<u64> {
        (self.0.OptionalFileOffset as u64)
            ..(self.0.OptionalFileOffset + self.0.OptionalLength) as u64
    }

    /// The last time the file was dehydrated.
    pub fn last_dehydration_time(&self) -> FileTime {
        self.0.LastDehydrationTime.try_into().unwrap()
    }

    /// The reason the file was last dehydrated.
    pub fn last_dehydration_reason(&self) -> Option<DehydrationReason> {
        DehydrationReason::from_win32(self.0.LastDehydrationReason)
    }
}

impl Debug for FetchData {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("FetchData")
            .field("interrupted_hydration", &self.interrupted_hydration())
            .field("required_file_range", &self.required_file_range())
            .field("optional_file_range", &self.optional_file_range())
            .field("last_dehydration_time", &self.last_dehydration_time())
            .field("last_dehydration_reason", &self.last_dehydration_reason())
            .finish()
    }
}

/// Information for the [SyncFilter::cancel_fetch_data][crate::filter::SyncFilter::cancel_fetch_data] callback.
pub struct CancelFetchData(pub(crate) CF_CALLBACK_PARAMETERS_0_0);

impl CancelFetchData {
    /// Whether or not the callback failed as a result of the 60 second timeout.
    pub fn timeout(&self) -> bool {
        (self.0.Flags & CloudFilters::CF_CALLBACK_CANCEL_FLAG_IO_TIMEOUT).0 != 0
    }

    /// The user has cancelled the request manually.
    ///
    /// A user could cancel a request through a download toast?
    pub fn user_cancelled(&self) -> bool {
        (self.0.Flags & CloudFilters::CF_CALLBACK_CANCEL_FLAG_IO_ABORTED).0 != 0
    }

    /// The range of the file data that is no longer required.
    pub fn file_range(&self) -> Range<u64> {
        let range = unsafe { self.0.Anonymous.FetchData };
        (range.FileOffset as u64)..(range.FileOffset + range.Length) as u64
    }
}

impl Debug for CancelFetchData {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("CancelFetchData")
            .field("timeout", &self.timeout())
            .field("user_cancelled", &self.user_cancelled())
            .field("file_range", &self.file_range())
            .finish()
    }
}

/// Information for the [SyncFilter::validate_data][crate::filter::SyncFilter::validate_data] callback.
pub struct ValidateData(pub(crate) CF_CALLBACK_PARAMETERS_0_11);

impl ValidateData {
    /// Whether or not the callback failed as a result of the 60 second timeout.
    pub fn explicit_hydration(&self) -> bool {
        (self.0.Flags & CloudFilters::CF_CALLBACK_VALIDATE_DATA_FLAG_EXPLICIT_HYDRATION).0 != 0
    }

    /// The range of data to validate.
    pub fn file_range(&self) -> Range<u64> {
        (self.0.RequiredFileOffset as u64)
            ..(self.0.RequiredFileOffset + self.0.RequiredLength) as u64
    }
}

impl Debug for ValidateData {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("ValidateData")
            .field("explicit_hydration", &self.explicit_hydration())
            .field("file_range", &self.file_range())
            .finish()
    }
}

/// Information for the [SyncFilter::fetch_placeholders][crate::filter::SyncFilter::fetch_placeholders]
/// callback.
pub struct FetchPlaceholders(pub(crate) CF_CALLBACK_PARAMETERS_0_7);

impl FetchPlaceholders {
    /// A glob pattern specifying the files that should be fetched.
    ///
    /// This field is completely optional and does not have to be respected.
    pub fn pattern(&self) -> String {
        unsafe { U16CStr::from_ptr_str(self.0.Pattern.0) }.to_string_lossy()
    }
}

impl Debug for FetchPlaceholders {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("FetchPlaceholders")
            .field("pattern", &self.pattern())
            .finish()
    }
}

/// Information for the
/// [SyncFilter::cancel_fetch_placeholders][super::SyncFilter::cancel_fetch_placeholders] callback.
pub struct CancelFetchPlaceholders(pub(crate) CF_CALLBACK_PARAMETERS_0_0);

impl CancelFetchPlaceholders {
    /// Whether or not the callback failed as a result of the 60 second timeout.
    ///
    // Read more [here][crate::filter::Request::reset_timeout].
    pub fn timeout(&self) -> bool {
        (self.0.Flags & CloudFilters::CF_CALLBACK_CANCEL_FLAG_IO_TIMEOUT).0 != 0
    }

    /// The user has cancelled the request manually.
    ///
    /// A user could cancel a request through a download toast?
    pub fn user_cancelled(&self) -> bool {
        (self.0.Flags & CloudFilters::CF_CALLBACK_CANCEL_FLAG_IO_ABORTED).0 != 0
    }
}

impl Debug for CancelFetchPlaceholders {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("CancelFetchPlaceholders")
            .field("timeout", &self.timeout())
            .field("user_cancelled", &self.user_cancelled())
            .finish()
    }
}

/// Information for the [SyncFilter::opened][super::SyncFilter::opened] callback.
pub struct Opened(pub(crate) CF_CALLBACK_PARAMETERS_0_8);

impl Opened {
    /// The placeholder metadata is corrupt.
    pub fn metadata_corrupt(&self) -> bool {
        (self.0.Flags & CloudFilters::CF_CALLBACK_OPEN_COMPLETION_FLAG_PLACEHOLDER_UNKNOWN).0 != 0
    }

    /// The placeholder metadata is not supported.
    pub fn metadata_unsupported(&self) -> bool {
        (self.0.Flags & CloudFilters::CF_CALLBACK_OPEN_COMPLETION_FLAG_PLACEHOLDER_UNSUPPORTED).0
            != 0
    }
}

impl Debug for Opened {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("Opened")
            .field("metadata_corrupt", &self.metadata_corrupt())
            .field("metadata_unsupported", &self.metadata_unsupported())
            .finish()
    }
}

/// Information for the [SyncFilter::closed][super::SyncFilter::closed] callback.
pub struct Closed(pub(crate) CF_CALLBACK_PARAMETERS_0_1);

impl Closed {
    /// Whether or not the placeholder was deleted as a result of the close.
    pub fn deleted(&self) -> bool {
        (self.0.Flags & CloudFilters::CF_CALLBACK_CLOSE_COMPLETION_FLAG_DELETED).0 != 0
    }
}

impl Debug for Closed {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("Closed")
            .field("deleted", &self.deleted())
            .finish()
    }
}

/// Information for the [SyncFilter::dehydrate][super::SyncFilter::dehydrate] callback.
pub struct Dehydrate(pub(crate) CF_CALLBACK_PARAMETERS_0_3);

impl Dehydrate {
    /// Whether or not the callback was called from a system background service.
    pub fn background(&self) -> bool {
        (self.0.Flags & CloudFilters::CF_CALLBACK_DEHYDRATE_FLAG_BACKGROUND).0 != 0
    }

    /// The reason the file is being dehydrated.
    pub fn reason(&self) -> Option<DehydrationReason> {
        DehydrationReason::from_win32(self.0.Reason)
    }
}

impl Debug for Dehydrate {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("Dehydrate")
            .field("background", &self.background())
            .field("reason", &self.reason())
            .finish()
    }
}

/// Information for the [SyncFilter::dehydrated][super::SyncFilter::dehydrated] callback.
pub struct Dehydrated(pub(crate) CF_CALLBACK_PARAMETERS_0_2);

impl Dehydrated {
    /// Whether or not the callback was called from a system background service.
    pub fn background(&self) -> bool {
        (self.0.Flags & CloudFilters::CF_CALLBACK_DEHYDRATE_COMPLETION_FLAG_BACKGROUND).0 != 0
    }

    /// Whether or not the placeholder was already hydrated.
    pub fn already_hydrated(&self) -> bool {
        (self.0.Flags & CloudFilters::CF_CALLBACK_DEHYDRATE_COMPLETION_FLAG_DEHYDRATED).0 != 0
    }

    /// The reason the file is being dehydrated.
    pub fn reason(&self) -> Option<DehydrationReason> {
        DehydrationReason::from_win32(self.0.Reason)
    }
}

impl Debug for Dehydrated {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("Dehydrated")
            .field("background", &self.background())
            .field("already_hydrated", &self.already_hydrated())
            .field("reason", &self.reason())
            .finish()
    }
}

/// Information for the [SyncFilter::delete][super::SyncFilter::delete] callback.
pub struct Delete(pub(crate) CF_CALLBACK_PARAMETERS_0_5);

impl Delete {
    /// Whether or not the placeholder being deleted is a directory.
    pub fn is_directory(&self) -> bool {
        (self.0.Flags & CloudFilters::CF_CALLBACK_DELETE_FLAG_IS_DIRECTORY).0 != 0
    }

    // TODO: missing docs
    /// The placeholder is being undeleted.
    pub fn is_undelete(&self) -> bool {
        (self.0.Flags & CloudFilters::CF_CALLBACK_DELETE_FLAG_IS_UNDELETE).0 != 0
    }
}

impl Debug for Delete {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("Delete")
            .field("is_directory", &self.is_directory())
            .field("is_undelete", &self.is_undelete())
            .finish()
    }
}

/// Information for the [SyncFilter::deleted][crate::filter::SyncFilter::deleted] callback.
#[derive(Debug)]
#[allow(dead_code)]
pub struct Deleted(pub(crate) CF_CALLBACK_PARAMETERS_0_4);

/// Information for the [SyncFilter::rename][crate::filter::SyncFilter::rename] callback.
pub struct Rename(pub(crate) CF_CALLBACK_PARAMETERS_0_10, pub(crate) OsString);

impl Rename {
    /// Whether or not the placeholder being renamed is a directory.
    pub fn is_directory(&self) -> bool {
        (self.0.Flags & CloudFilters::CF_CALLBACK_RENAME_FLAG_IS_DIRECTORY).0 != 0
    }

    /// Whether or not the placeholder was originally in the sync root.
    pub fn source_in_scope(&self) -> bool {
        (self.0.Flags & CloudFilters::CF_CALLBACK_RENAME_FLAG_SOURCE_IN_SCOPE).0 != 0
    }

    /// Whether or not the placeholder is being moved inside the sync root.
    pub fn target_in_scope(&self) -> bool {
        (self.0.Flags & CloudFilters::CF_CALLBACK_RENAME_FLAG_TARGET_IN_SCOPE).0 != 0
    }

    /// The full path the placeholder is being moved to.
    pub fn target_path(&self) -> PathBuf {
        let mut path = PathBuf::from(&self.1);
        path.push(unsafe { U16CStr::from_ptr_str(self.0.TargetPath.0) }.to_os_string());
        path
    }
}

impl Debug for Rename {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("Rename")
            .field("is_directory", &self.is_directory())
            .field("source_in_scope", &self.source_in_scope())
            .field("target_in_scope", &self.target_in_scope())
            .field("target_path", &self.target_path())
            .finish()
    }
}

/// Information for the [SyncFilter::renamed][crate::filter::SyncFilter::renamed] callback.
pub struct Renamed(pub(crate) CF_CALLBACK_PARAMETERS_0_9, pub(crate) OsString);

impl Renamed {
    /// The full path the placeholder has been moved from.
    pub fn source_path(&self) -> PathBuf {
        let mut path = PathBuf::from(&self.1);
        path.push(unsafe { U16CStr::from_ptr_str(self.0.SourcePath.0) }.to_os_string());
        path
    }
}

impl Debug for Renamed {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("Renamed")
            .field("source_path", &self.source_path())
            .finish()
    }
}

/// The reason a placeholder has been dehydrated.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum DehydrationReason {
    /// The user manually dehydrated the placeholder.
    UserManually,
    /// The operating system automatically dehydrated the placeholder due to low disk space on the
    /// volume.
    LowSpace,
    /// The operating system automatically dehydrated the placeholder due to low activity.
    ///
    /// This is based on the Windows Storage Sense settings.
    Inactive,
    /// The operating system automatically dehydrated this file to make room for an operating
    /// system upgrade.
    OsUpgrade,
}

impl DehydrationReason {
    fn from_win32(reason: CF_CALLBACK_DEHYDRATION_REASON) -> Option<DehydrationReason> {
        match reason {
            CloudFilters::CF_CALLBACK_DEHYDRATION_REASON_USER_MANUAL => Some(Self::UserManually),
            CloudFilters::CF_CALLBACK_DEHYDRATION_REASON_SYSTEM_LOW_SPACE => Some(Self::LowSpace),
            CloudFilters::CF_CALLBACK_DEHYDRATION_REASON_SYSTEM_INACTIVITY => Some(Self::Inactive),
            CloudFilters::CF_CALLBACK_DEHYDRATION_REASON_SYSTEM_OS_UPGRADE => Some(Self::OsUpgrade),
            _ => None,
        }
    }
}
