use std::ops::Range;

use windows::{
    Win32::Storage::CloudFilters::{CF_CONNECTION_KEY, CfReportProviderProgress},
    core,
};

use crate::cfapi::{
    command::{self, Command},
    filter::{RawConnectionKey, RawTransferKey},
    placeholder_file::PlaceholderFile,
    sealed, utility,
};

/// A ticket for the [SyncFilter::fetch_data][crate::filter::SyncFilter::fetch_data] callback.
#[derive(Debug)]
pub struct FetchData {
    connection_key: RawConnectionKey,
    transfer_key: RawTransferKey,
}

impl FetchData {
    /// Create a new [FetchData].
    pub(crate) fn new(connection_key: RawConnectionKey, transfer_key: RawTransferKey) -> Self {
        Self {
            connection_key,
            transfer_key,
        }
    }

    /// Displays a progress bar next to the file in the file explorer to show the progress of the
    /// current operation. In addition, the standard Windows file progress dialog will open
    /// displaying the speed and progress based on the values set. During background hydrations,
    /// an interactive toast will appear notifying the user of an operation with a progress bar.
    pub fn report_progress(&self, total: u64, completed: u64) -> core::Result<()> {
        unsafe {
            CfReportProviderProgress(
                CF_CONNECTION_KEY(self.connection_key),
                self.transfer_key,
                total as i64,
                completed as i64,
            )
        }?;

        Ok(())
    }

    // TODO: response command::Update
}

impl utility::ReadAt for FetchData {
    /// Read data at an offset from a placeholder file.
    ///
    /// This method is equivalent to calling `CfExecute` with `CF_OPERATION_TYPE_RETRIEVE_DATA`.
    fn read_at(&self, buf: &mut [u8], offset: u64) -> core::Result<u64> {
        command::Read {
            buffer: buf,
            position: offset,
        }
        .execute(self.connection_key, self.transfer_key)
    }
}

impl utility::WriteAt for FetchData {
    /// Write data at an offset to a placeholder file.
    ///
    /// The buffer passed must be 4KiB in length or end on the logical file size. Unfortunately,
    /// this is a restriction of the operating system.
    ///
    /// This method is equivalent to calling `CfExecute` with `CF_OPERATION_TYPE_TRANSFER_DATA`.
    fn write_at(&self, buf: &[u8], offset: u64) -> core::Result<()> {
        command::Write {
            buffer: buf,
            position: offset,
        }
        .execute(self.connection_key, self.transfer_key)
    }
}

impl sealed::Sealed for FetchData {}

/// A ticket for the [SyncFilter::validate_data][crate::filter::SyncFilter::validate_data] callback.
#[derive(Debug)]
pub struct ValidateData {
    connection_key: RawConnectionKey,
    transfer_key: RawTransferKey,
}

impl ValidateData {
    /// Create a new [ValidateData].
    pub(crate) fn new(connection_key: RawConnectionKey, transfer_key: RawTransferKey) -> Self {
        Self {
            connection_key,
            transfer_key,
        }
    }

    /// Validates the data range in the placeholder file is valid.
    ///
    /// This method is equivalent to calling `CfExecute` with `CF_OPERATION_TYPE_ACK_DATA`.
    // TODO: make this generic over a RangeBounds
    // if the range specified is past the current file length, will it consider that range to be validated?
    // https://docs.microsoft.com/en-us/answers/questions/750302/if-the-ackdata-field-of-cf-operation-parameters-is.html
    pub fn pass(&self, range: Range<u64>) -> core::Result<()> {
        command::Validate { range }.execute(self.connection_key, self.transfer_key)
    }

    // TODO: response command::Update
}

impl utility::ReadAt for ValidateData {
    /// Read data at an offset from a placeholder file.
    ///
    /// This method is equivalent to calling `CfExecute` with `CF_OPERATION_TYPE_RETRIEVE_DATA`.
    ///
    /// The bytes returned will ALWAYS be the length of the buffer passed in. The operating
    /// system provides these guarantees.
    fn read_at(&self, buf: &mut [u8], offset: u64) -> core::Result<u64> {
        command::Read {
            buffer: buf,
            position: offset,
        }
        .execute(self.connection_key, self.transfer_key)
    }
}

impl sealed::Sealed for ValidateData {}

/// A ticket for the [SyncFilter::fetch_placeholders][crate::filter::SyncFilter::fetch_placeholders] callback.
#[derive(Debug)]
pub struct FetchPlaceholders {
    connection_key: RawConnectionKey,
    transfer_key: RawTransferKey,
}

impl FetchPlaceholders {
    /// Create a new [FetchPlaceholders].
    pub(crate) fn new(connection_key: RawConnectionKey, transfer_key: RawTransferKey) -> Self {
        Self {
            connection_key,
            transfer_key,
        }
    }

    /// Creates a list of placeholder files/directorys on the file system.
    ///
    /// The value returned is the final [Usn][crate::usn::Usn] (and if they succeeded) after each placeholder is created.
    pub fn pass_with_placeholder(&self, placeholders: &mut [PlaceholderFile]) -> core::Result<()> {
        command::CreatePlaceholders {
            total: placeholders.len() as _,
            placeholders,
        }
        .execute(self.connection_key, self.transfer_key)
    }
}

/// A ticket for the [SyncFilter::dehydrate][crate::filter::SyncFilter::dehydrate] callback.
#[derive(Debug)]
pub struct Dehydrate {
    connection_key: RawConnectionKey,
    transfer_key: RawTransferKey,
}

impl Dehydrate {
    /// Create a new [Dehydrate].
    pub(crate) fn new(connection_key: RawConnectionKey, transfer_key: RawTransferKey) -> Self {
        Self {
            connection_key,
            transfer_key,
        }
    }

    /// Confirms dehydration of the file.
    pub fn pass(&self) -> core::Result<()> {
        command::Dehydrate { blob: &[] }.execute(self.connection_key, self.transfer_key)
    }

    /// Confirms dehydration of the file and updates its file blob.
    pub fn pass_with_blob(&self, blob: &[u8]) -> core::Result<()> {
        command::Dehydrate { blob }.execute(self.connection_key, self.transfer_key)
    }
}

/// A ticket for the [SyncFilter::delete][crate::filter::SyncFilter::delete] callback.
#[derive(Debug)]
pub struct Delete {
    connection_key: RawConnectionKey,
    transfer_key: RawTransferKey,
}

impl Delete {
    /// Create a new [Delete].
    pub(crate) fn new(connection_key: RawConnectionKey, transfer_key: RawTransferKey) -> Self {
        Self {
            connection_key,
            transfer_key,
        }
    }

    /// Confirms deletion of the file.
    pub fn pass(&self) -> core::Result<()> {
        command::Delete.execute(self.connection_key, self.transfer_key)
    }
}

/// A ticket for the [SyncFilter::rename][crate::filter::SyncFilter::rename] callback.
#[derive(Debug)]
pub struct Rename {
    connection_key: RawConnectionKey,
    transfer_key: RawTransferKey,
}

impl Rename {
    /// Create a new [Rename].
    pub(crate) fn new(connection_key: RawConnectionKey, transfer_key: RawTransferKey) -> Self {
        Self {
            connection_key,
            transfer_key,
        }
    }

    /// Confirms the rename/move of a file.
    pub fn pass(&self) -> core::Result<()> {
        command::Rename.execute(self.connection_key, self.transfer_key)
    }
}
