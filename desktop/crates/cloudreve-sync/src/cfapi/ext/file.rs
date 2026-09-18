use std::{
    fs::File,
    mem,
    ops::{Bound, RangeBounds},
    os::windows::{io::AsRawHandle, prelude::RawHandle},
};

use widestring::U16CStr;
use windows::{
    Win32::{
        Foundation::HANDLE,
        Storage::CloudFilters::{
            self, CF_SYNC_PROVIDER_STATUS, CF_SYNC_ROOT_STANDARD_INFO, CfDehydratePlaceholder,
        },
    },
    core,
};

use crate::cfapi::sealed::Sealed;

/// An API extension to [File][std::fs::File].
pub trait FileExt: AsRawHandle + Sealed {
    /// Dehydrates a placeholder file.
    fn dehydrate<T: RangeBounds<u64>>(&self, range: T) -> core::Result<()> {
        dehydrate(self.as_raw_handle(), range, false)
    }

    /// Dehydrates a placeholder file as a system process running in the background. Otherwise, it
    /// is called on behalf of a logged-in user.
    fn background_dehydrate<T: RangeBounds<u64>>(&self, range: T) -> core::Result<()> {
        dehydrate(self.as_raw_handle(), range, true)
    }

    /// Returns whether or not the handle is inside of a sync root.
    fn in_sync_root() -> core::Result<bool> {
        // TODO: this should use the uwp apis
        todo!()
    }
}

// TODO: is `CfDehydratePlaceholder` deprecated?
// https://docs.microsoft.com/en-us/answers/questions/723805/what-is-the-behavior-of-file-ranges-in-different-p.html
fn dehydrate<T: RangeBounds<u64>>(
    handle: RawHandle,
    range: T,
    background: bool,
) -> core::Result<()> {
    unsafe {
        CfDehydratePlaceholder(
            HANDLE(handle),
            match range.start_bound() {
                Bound::Included(x) => *x as i64,
                Bound::Excluded(x) => x.saturating_add(1) as i64,
                Bound::Unbounded => 0,
            },
            match range.end_bound() {
                Bound::Included(x) => *x as i64,
                Bound::Excluded(x) => x.saturating_sub(1) as i64,
                // This behavior is documented in CfDehydratePlaceholder
                Bound::Unbounded => -1,
            },
            if background {
                CloudFilters::CF_DEHYDRATE_FLAG_NONE
            } else {
                CloudFilters::CF_DEHYDRATE_FLAG_BACKGROUND
            },
            None,
        )
    }
}

impl FileExt for File {}

impl Sealed for File {}

/// Information about a sync root.
#[derive(Debug)]
pub struct SyncRootInfo {
    data: Vec<u8>,
    info: *const CF_SYNC_ROOT_STANDARD_INFO,
}

// TODO: most of the returns only have setters, no getters
impl SyncRootInfo {
    /// The file ID of the sync root.
    pub fn file_id(&self) -> u64 {
        unsafe { &*self.info }.SyncRootFileId as u64
    }

    // /// The hydration policy of the sync root.
    // pub fn hydration_policy(&self) -> HydrationType {
    //     unsafe { &*self.info }.HydrationPolicy.Primary.into()
    // }

    // /// The hydration type of the sync root.
    // pub fn hydration_type(&self) -> HydrationPolicy {
    //     unsafe { &*self.info }.HydrationPolicy.Modifier.into()
    // }

    // /// The population type of the sync root.
    // pub fn population_type(&self) -> PopulationType {
    //     unsafe { &*self.info }.PopulationPolicy.Primary.into()
    // }

    // /// The attributes supported by the sync root.
    // pub fn supported_attributes(&self) -> SupportedAttributes {
    //     unsafe { &*self.info }.InSyncPolicy.into()
    // }

    /// Whether or not hardlinks are allowed by the sync root.
    pub fn hardlinks_allowed(&self) -> bool {
        unsafe { &*self.info }.HardLinkPolicy == CloudFilters::CF_HARDLINK_POLICY_ALLOWED
    }

    /// The status of the sync provider.
    pub fn status(&self) -> ProviderStatus {
        unsafe { &*self.info }.ProviderStatus.into()
    }

    /// The name of the sync provider.
    pub fn provider_name(&self) -> &U16CStr {
        U16CStr::from_slice_truncate(unsafe { &*self.info }.ProviderName.as_slice()).unwrap()
    }

    /// The version of the sync provider.
    pub fn version(&self) -> &U16CStr {
        U16CStr::from_slice_truncate(unsafe { &*self.info }.ProviderVersion.as_slice()).unwrap()
    }

    /// The register blob associated with the sync root.
    pub fn blob(&self) -> &[u8] {
        &self.data[(mem::size_of::<CF_SYNC_ROOT_STANDARD_INFO>() + 1)..]
    }
}

/// Sync provider status.
#[derive(Debug, Clone, Copy)]
pub enum ProviderStatus {
    /// The sync provider is disconnected.
    Disconnected,
    /// The sync provider is idle.
    Idle,
    /// The sync provider is populating a namespace.
    PopulateNamespace,
    /// The sync provider is populating placeholder metadata.
    PopulateMetadata,
    /// The sync provider is incrementally syncing placeholder content.
    PopulateContent,
    /// The sync provider is incrementally syncing placeholder content.
    SyncIncremental,
    /// The sync provider has fully synced placeholder data.
    SyncFull,
    /// The sync provider has lost connectivity.
    ConnectivityLost,
    // TODO: if setting the sync status is added.
    // ClearFlags,
    /// The sync provider has been terminated.
    Terminated,
    /// The sync provider had an error.
    Error,
}

impl From<CF_SYNC_PROVIDER_STATUS> for ProviderStatus {
    fn from(status: CF_SYNC_PROVIDER_STATUS) -> Self {
        match status {
            CloudFilters::CF_PROVIDER_STATUS_DISCONNECTED => Self::Disconnected,
            CloudFilters::CF_PROVIDER_STATUS_IDLE => Self::Idle,
            CloudFilters::CF_PROVIDER_STATUS_POPULATE_NAMESPACE => Self::PopulateNamespace,
            CloudFilters::CF_PROVIDER_STATUS_POPULATE_METADATA => Self::PopulateContent,
            CloudFilters::CF_PROVIDER_STATUS_POPULATE_CONTENT => Self::PopulateContent,
            CloudFilters::CF_PROVIDER_STATUS_SYNC_INCREMENTAL => Self::SyncIncremental,
            CloudFilters::CF_PROVIDER_STATUS_SYNC_FULL => Self::SyncFull,
            CloudFilters::CF_PROVIDER_STATUS_CONNECTIVITY_LOST => Self::ConnectivityLost,
            // CloudFilters::CF_PROVIDER_STATUS_CLEAR_FLAGS => Self::ClearFlags,
            CloudFilters::CF_PROVIDER_STATUS_TERMINATED => Self::Terminated,
            CloudFilters::CF_PROVIDER_STATUS_ERROR => Self::Error,
            _ => unreachable!(),
        }
    }
}

impl From<ProviderStatus> for CF_SYNC_PROVIDER_STATUS {
    fn from(status: ProviderStatus) -> Self {
        match status {
            ProviderStatus::Disconnected => CloudFilters::CF_PROVIDER_STATUS_DISCONNECTED,
            ProviderStatus::Idle => CloudFilters::CF_PROVIDER_STATUS_IDLE,
            ProviderStatus::PopulateNamespace => {
                CloudFilters::CF_PROVIDER_STATUS_POPULATE_NAMESPACE
            }
            ProviderStatus::PopulateMetadata => CloudFilters::CF_PROVIDER_STATUS_POPULATE_METADATA,
            ProviderStatus::PopulateContent => CloudFilters::CF_PROVIDER_STATUS_POPULATE_CONTENT,
            ProviderStatus::SyncIncremental => CloudFilters::CF_PROVIDER_STATUS_SYNC_INCREMENTAL,
            ProviderStatus::SyncFull => CloudFilters::CF_PROVIDER_STATUS_SYNC_FULL,
            ProviderStatus::ConnectivityLost => CloudFilters::CF_PROVIDER_STATUS_CONNECTIVITY_LOST,
            // ProviderStatus::ClearFlags => CloudFilters::CF_PROVIDER_STATUS_CLEAR_FLAGS,
            ProviderStatus::Terminated => CloudFilters::CF_PROVIDER_STATUS_TERMINATED,
            ProviderStatus::Error => CloudFilters::CF_PROVIDER_STATUS_ERROR,
        }
    }
}
