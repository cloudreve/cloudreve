use windows::Win32::Foundation::{self, NTSTATUS};

/// [SyncFilter][crate::filter::SyncFilter] trait callback result type.
pub type CResult<T> = std::result::Result<T, CloudErrorKind>;

/// Predefined error types provided by the operating system.
#[derive(Debug, Clone, Copy)]
pub enum CloudErrorKind {
    /// Access to the cloud file is denied.
    AccessDenied,
    /// The cloud sync root is already connected with another cloud sync provider.
    AlreadyConnected,
    /// The cloud sync provider failed user authentication.
    AuthenticationFailed,
    /// The operation is reserved for a connected cloud sync provider.
    ConnectedProviderOnly,
    /// Dehydration of the cloud file is disallowed by the cloud sync provider.
    DehydrationDisallowed,
    /// The cloud operation cannot be performed on a file with incompatible hardlinks.
    IncompatibleHardlinks,
    /// The cloud sync provider failed to perform the operation due to low system resources.
    InsufficientResources,
    /// The cloud operation is invalid.
    InvalidRequest,
    /// The operation cannot be performed on cloud files in use.
    InUse,
    /// The cloud file metadata is corrupt and unreadable.
    MetadataCorrupt,
    /// The cloud file metadata is too large.
    MetadataTooLarge,
    /// The cloud sync provider failed to perform the operation due to network being unavailable.
    NetworkUnavailable,
    /// The file is not in sync with the cloud.
    NotInSync,
    /// The operation is not supported by the cloud sync provider.
    NotSupported,
    /// The operation is only supported on files under a cloud sync root.
    NotUnderSyncRoot,
    /// The operation cannot be performed on pinned cloud files.
    Pinned,
    /// The cloud file property is possibly corrupt. The on-disk checksum does not match the
    /// computed checksum.
    PropertyBlobChecksumMismatch,
    /// The cloud file property is too large.
    PropertyBlobTooLarge,
    /// The cloud file's property store is corrupt.
    PropertyCorrupt,
    /// The operation failed due to a conflicting cloud file property lock.
    PropertyLockConflict,
    /// The version of the cloud file property store is not supported.
    PropertyVersionNotSupported,
    /// The cloud file provider is not running.
    ProviderNotRunning,
    /// The cloud file provider exited unexpectedly.
    ProviderTerminated,
    /// The cloud operation is not supported on a read-only volume.
    ReadOnlyVolume,
    /// The cloud operation was aborted.
    RequestAborted,
    /// The cloud operation was canceled by user.
    RequestCancelled,
    /// The cloud operation was not completed before the time-out period expired.
    RequestTimeout,
    /// The cloud sync root metadata is corrupted.
    SyncRootMetadataCorrupt,
    /// The maximum number of cloud file properties has been reached.
    TooManyPropertyBlobs,
    /// The cloud operation was unsuccessful.
    Unsuccessful,
    /// The cloud sync provider failed to validate the downloaded data.
    ValidationFailed,
}

impl From<CloudErrorKind> for NTSTATUS {
    fn from(error: CloudErrorKind) -> Self {
        match error {
            CloudErrorKind::AccessDenied => Foundation::STATUS_CLOUD_FILE_ACCESS_DENIED,
            CloudErrorKind::AlreadyConnected => Foundation::STATUS_CLOUD_FILE_ALREADY_CONNECTED,
            CloudErrorKind::AuthenticationFailed => {
                Foundation::STATUS_CLOUD_FILE_AUTHENTICATION_FAILED
            }
            CloudErrorKind::ConnectedProviderOnly => {
                Foundation::STATUS_CLOUD_FILE_CONNECTED_PROVIDER_ONLY
            }
            CloudErrorKind::DehydrationDisallowed => {
                Foundation::STATUS_CLOUD_FILE_DEHYDRATION_DISALLOWED
            }
            CloudErrorKind::IncompatibleHardlinks => {
                Foundation::STATUS_CLOUD_FILE_INCOMPATIBLE_HARDLINKS
            }
            CloudErrorKind::InsufficientResources => {
                Foundation::STATUS_CLOUD_FILE_INSUFFICIENT_RESOURCES
            }
            CloudErrorKind::InvalidRequest => Foundation::STATUS_CLOUD_FILE_INVALID_REQUEST,
            CloudErrorKind::InUse => Foundation::STATUS_CLOUD_FILE_IN_USE,
            CloudErrorKind::MetadataCorrupt => Foundation::STATUS_CLOUD_FILE_METADATA_CORRUPT,
            CloudErrorKind::MetadataTooLarge => Foundation::STATUS_CLOUD_FILE_METADATA_TOO_LARGE,
            CloudErrorKind::NetworkUnavailable => Foundation::STATUS_CLOUD_FILE_NETWORK_UNAVAILABLE,
            CloudErrorKind::NotInSync => Foundation::STATUS_CLOUD_FILE_NOT_IN_SYNC,
            CloudErrorKind::NotSupported => Foundation::STATUS_CLOUD_FILE_NOT_SUPPORTED,
            CloudErrorKind::NotUnderSyncRoot => Foundation::STATUS_CLOUD_FILE_NOT_UNDER_SYNC_ROOT,
            CloudErrorKind::Pinned => Foundation::STATUS_CLOUD_FILE_PINNED,
            CloudErrorKind::PropertyBlobChecksumMismatch => {
                Foundation::STATUS_CLOUD_FILE_PROPERTY_BLOB_CHECKSUM_MISMATCH
            }
            CloudErrorKind::PropertyBlobTooLarge => {
                Foundation::STATUS_CLOUD_FILE_PROPERTY_BLOB_TOO_LARGE
            }
            CloudErrorKind::PropertyCorrupt => Foundation::STATUS_CLOUD_FILE_PROPERTY_CORRUPT,
            CloudErrorKind::PropertyLockConflict => {
                Foundation::STATUS_CLOUD_FILE_PROPERTY_LOCK_CONFLICT
            }
            CloudErrorKind::PropertyVersionNotSupported => {
                Foundation::STATUS_CLOUD_FILE_PROPERTY_VERSION_NOT_SUPPORTED
            }
            CloudErrorKind::ProviderNotRunning => {
                Foundation::STATUS_CLOUD_FILE_PROVIDER_NOT_RUNNING
            }
            CloudErrorKind::ProviderTerminated => Foundation::STATUS_CLOUD_FILE_PROVIDER_TERMINATED,
            CloudErrorKind::ReadOnlyVolume => Foundation::STATUS_CLOUD_FILE_READ_ONLY_VOLUME,
            CloudErrorKind::RequestAborted => Foundation::STATUS_CLOUD_FILE_REQUEST_ABORTED,
            CloudErrorKind::RequestCancelled => Foundation::STATUS_CLOUD_FILE_REQUEST_CANCELED,
            CloudErrorKind::RequestTimeout => Foundation::STATUS_CLOUD_FILE_REQUEST_TIMEOUT,
            CloudErrorKind::SyncRootMetadataCorrupt => {
                Foundation::STATUS_CLOUD_FILE_SYNC_ROOT_METADATA_CORRUPT
            }
            CloudErrorKind::TooManyPropertyBlobs => {
                Foundation::STATUS_CLOUD_FILE_TOO_MANY_PROPERTY_BLOBS
            }
            CloudErrorKind::Unsuccessful => Foundation::STATUS_CLOUD_FILE_UNSUCCESSFUL,
            CloudErrorKind::ValidationFailed => Foundation::STATUS_CLOUD_FILE_VALIDATION_FAILED,
        }
    }
}
