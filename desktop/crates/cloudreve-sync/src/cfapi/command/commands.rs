use std::{ops::Range, ptr};

use windows::{
    Win32::{
        Foundation,
        Storage::CloudFilters::{
            self, CF_OPERATION_PARAMETERS_0, CF_OPERATION_PARAMETERS_0_0,
            CF_OPERATION_PARAMETERS_0_1, CF_OPERATION_PARAMETERS_0_2, CF_OPERATION_PARAMETERS_0_3,
            CF_OPERATION_PARAMETERS_0_4, CF_OPERATION_PARAMETERS_0_5, CF_OPERATION_PARAMETERS_0_6,
            CF_OPERATION_PARAMETERS_0_7, CF_OPERATION_TYPE,
        },
    },
    core,
};

use crate::cfapi::{
    command::executor::{Command, Fallible, execute},
    error::CloudErrorKind,
    filter::{RawConnectionKey, RawTransferKey},
    metadata::Metadata,
    placeholder_file::PlaceholderFile,
};

/// Read data from a placeholder file.
#[derive(Debug)]
pub struct Read<'a> {
    /// The buffer of data to read into.
    pub buffer: &'a mut [u8],
    /// The starting offset of from where to read.
    pub position: u64,
}

impl Command for Read<'_> {
    const OPERATION: CF_OPERATION_TYPE = CloudFilters::CF_OPERATION_TYPE_RETRIEVE_DATA;

    type Result = u64;
    type Field = CF_OPERATION_PARAMETERS_0_5;

    unsafe fn result(info: CF_OPERATION_PARAMETERS_0) -> Self::Result { unsafe {
        info.RetrieveData.ReturnedLength as u64
    }}

    fn build(&self) -> CF_OPERATION_PARAMETERS_0 {
        CF_OPERATION_PARAMETERS_0 {
            RetrieveData: CF_OPERATION_PARAMETERS_0_5 {
                Flags: CloudFilters::CF_OPERATION_RETRIEVE_DATA_FLAG_NONE,
                Buffer: self.buffer.as_ptr() as *mut _,
                Offset: self.position as i64,
                Length: self.buffer.len() as i64,
                ReturnedLength: 0,
            },
        }
    }
}

/// Write data to a placeholder file.
#[derive(Debug)]
pub struct Write<'a> {
    /// The buffer of data to write into the file.
    pub buffer: &'a [u8],
    /// The starting offset of from where to write.
    pub position: u64,
}

impl Command for Write<'_> {
    const OPERATION: CF_OPERATION_TYPE = CloudFilters::CF_OPERATION_TYPE_TRANSFER_DATA;

    type Result = ();
    type Field = CF_OPERATION_PARAMETERS_0_6;

    unsafe fn result(_info: CF_OPERATION_PARAMETERS_0) -> Self::Result {}

    fn build(&self) -> CF_OPERATION_PARAMETERS_0 {
        CF_OPERATION_PARAMETERS_0 {
            TransferData: CF_OPERATION_PARAMETERS_0_6 {
                // TODO: add flag for disable_on_demand_population
                Flags: CloudFilters::CF_OPERATION_TRANSFER_DATA_FLAG_NONE,
                CompletionStatus: Foundation::STATUS_SUCCESS,
                Buffer: self.buffer.as_ptr() as *mut _,
                Offset: self.position as i64,
                Length: self.buffer.len() as i64,
            },
        }
    }
}

impl Fallible for Write<'_> {
    fn fail(
        connection_key: RawConnectionKey,
        transfer_key: RawTransferKey,
        error_kind: CloudErrorKind,
    ) -> core::Result<Self::Result> {
        execute::<Self>(
            CF_OPERATION_PARAMETERS_0 {
                TransferData: CF_OPERATION_PARAMETERS_0_6 {
                    Flags: CloudFilters::CF_OPERATION_TRANSFER_DATA_FLAG_NONE,
                    CompletionStatus: error_kind.into(),
                    // TODO: SAME HERE AS BELOW?
                    Buffer: [0; 1].as_mut_ptr() as *mut _,
                    Offset: 0,
                    // TODO: DOES THIS HAVE TO BE DEFINED?
                    Length: 0,
                },
            },
            connection_key,
            transfer_key,
        )
    }
}

/// Update various properties on a placeholder.
#[derive(Debug)]
#[allow(dead_code)]
pub struct Update<'a> {
    /// Whether or not to mark the placeholder as "synced."
    pub mark_in_sync: bool,
    /// Optional metadata to update.
    pub metadata: Option<Metadata>,
    /// Optional file blob to update.
    pub blob: &'a [u8],
}

impl Command for Update<'_> {
    const OPERATION: CF_OPERATION_TYPE = CloudFilters::CF_OPERATION_TYPE_RESTART_HYDRATION;

    type Result = ();
    type Field = CF_OPERATION_PARAMETERS_0_4;

    unsafe fn result(_info: CF_OPERATION_PARAMETERS_0) -> Self::Result {}

    fn build(&self) -> CF_OPERATION_PARAMETERS_0 {
        CF_OPERATION_PARAMETERS_0 {
            RestartHydration: CF_OPERATION_PARAMETERS_0_4 {
                Flags: if self.mark_in_sync {
                    CloudFilters::CF_OPERATION_RESTART_HYDRATION_FLAG_MARK_IN_SYNC
                } else {
                    CloudFilters::CF_OPERATION_RESTART_HYDRATION_FLAG_NONE
                },
                FsMetadata: self
                    .metadata
                    .as_ref()
                    .map_or(ptr::null(), |metadata| &metadata.0 as *const _),
                FileIdentity: match self.blob.is_empty() {
                    true => ptr::null(),
                    false => self.blob.as_ptr() as *const _,
                },
                FileIdentityLength: self.blob.len() as _,
            },
        }
    }
}

/// Create placeholder files/directories.
#[derive(Debug)]
pub struct CreatePlaceholders<'a> {
    /// The placeholders to create.
    pub placeholders: &'a mut [PlaceholderFile],
    /// The total amount of placeholders that are a child of the current directory.
    pub total: u64,
}

impl Command for CreatePlaceholders<'_> {
    const OPERATION: CF_OPERATION_TYPE = CloudFilters::CF_OPERATION_TYPE_TRANSFER_PLACEHOLDERS;

    type Result = ();
    type Field = CF_OPERATION_PARAMETERS_0_7;

    unsafe fn result(_info: CF_OPERATION_PARAMETERS_0) -> Self::Result {}

    fn build(&self) -> CF_OPERATION_PARAMETERS_0 {
        CF_OPERATION_PARAMETERS_0 {
            TransferPlaceholders: CF_OPERATION_PARAMETERS_0_7 {
                // TODO: this flag tells the system there are no more placeholders in this directory (when that can be untrue)
                //       in the future, implement streaming
                Flags: CloudFilters::CF_OPERATION_TRANSFER_PLACEHOLDERS_FLAG_DISABLE_ON_DEMAND_POPULATION,
                CompletionStatus: Foundation::STATUS_SUCCESS,
                PlaceholderTotalCount: self.total as i64,
                PlaceholderArray: match self.placeholders.is_empty() {
                    // If the slice is empty there will be an invalid memory access error, not with null pointers, however.
                    true => ptr::null_mut(),
                    false => self.placeholders.as_ptr() as *mut _,
                },
                PlaceholderCount: self.placeholders.len() as u32,
                EntriesProcessed: 0,
            },
        }
    }
}

impl Fallible for CreatePlaceholders<'_> {
    fn fail(
        connection_key: RawConnectionKey,
        transfer_key: RawTransferKey,
        error_kind: CloudErrorKind,
    ) -> core::Result<Self::Result> {
        execute::<Self>(
            CF_OPERATION_PARAMETERS_0 {
                TransferPlaceholders: CF_OPERATION_PARAMETERS_0_7 {
                    Flags: CloudFilters::CF_OPERATION_TRANSFER_PLACEHOLDERS_FLAG_NONE,
                    CompletionStatus: error_kind.into(),
                    PlaceholderTotalCount: 0,
                    // TODO: DOES THIS HAVE TO BE A VALID ARRAY?
                    PlaceholderArray: ptr::null_mut(),
                    PlaceholderCount: 0,
                    EntriesProcessed: 0,
                },
            },
            connection_key,
            transfer_key,
        )
    }
}

/// Validate the data range in the placeholder file is valid.
#[derive(Debug)]
pub struct Validate {
    /// The range of data to validate as "good."
    pub range: Range<u64>,
}

impl Command for Validate {
    const OPERATION: CF_OPERATION_TYPE = CloudFilters::CF_OPERATION_TYPE_ACK_DATA;

    type Result = ();
    type Field = CF_OPERATION_PARAMETERS_0_0;

    unsafe fn result(_info: CF_OPERATION_PARAMETERS_0) -> Self::Result {}

    fn build(&self) -> CF_OPERATION_PARAMETERS_0 {
        CF_OPERATION_PARAMETERS_0 {
            AckData: CF_OPERATION_PARAMETERS_0_0 {
                Flags: CloudFilters::CF_OPERATION_ACK_DATA_FLAG_NONE,
                CompletionStatus: Foundation::STATUS_SUCCESS,
                Offset: self.range.start as i64,
                Length: self.range.end as i64,
            },
        }
    }
}

impl Fallible for Validate {
    fn fail(
        connection_key: RawConnectionKey,
        transfer_key: RawTransferKey,
        error_kind: CloudErrorKind,
    ) -> core::Result<Self::Result> {
        execute::<Self>(
            CF_OPERATION_PARAMETERS_0 {
                AckData: CF_OPERATION_PARAMETERS_0_0 {
                    Flags: CloudFilters::CF_OPERATION_ACK_DATA_FLAG_NONE,
                    CompletionStatus: error_kind.into(),
                    Offset: 0,
                    Length: 0,
                },
            },
            connection_key,
            transfer_key,
        )
    }
}

/// Confirm dehydration of the placeholder file and optionally update its blob.
#[derive(Debug)]
pub struct Dehydrate<'a> {
    /// Optional file blob to update.
    pub blob: &'a [u8],
}

impl Command for Dehydrate<'_> {
    const OPERATION: CF_OPERATION_TYPE = CloudFilters::CF_OPERATION_TYPE_ACK_DEHYDRATE;

    type Result = ();
    type Field = CF_OPERATION_PARAMETERS_0_1;

    unsafe fn result(_info: CF_OPERATION_PARAMETERS_0) -> Self::Result {}

    fn build(&self) -> CF_OPERATION_PARAMETERS_0 {
        CF_OPERATION_PARAMETERS_0 {
            AckDehydrate: CF_OPERATION_PARAMETERS_0_1 {
                Flags: CloudFilters::CF_OPERATION_ACK_DEHYDRATE_FLAG_NONE,
                CompletionStatus: Foundation::STATUS_SUCCESS,
                FileIdentity: match self.blob.is_empty() {
                    true => ptr::null(),
                    false => self.blob.as_ptr() as *const _,
                },
                FileIdentityLength: self.blob.len() as u32,
            },
        }
    }
}

impl Fallible for Dehydrate<'_> {
    fn fail(
        connection_key: RawConnectionKey,
        transfer_key: RawTransferKey,
        error_kind: CloudErrorKind,
    ) -> core::Result<Self::Result> {
        execute::<Self>(
            CF_OPERATION_PARAMETERS_0 {
                AckDehydrate: CF_OPERATION_PARAMETERS_0_1 {
                    Flags: CloudFilters::CF_OPERATION_ACK_DEHYDRATE_FLAG_NONE,
                    CompletionStatus: error_kind.into(),
                    FileIdentity: ptr::null(),
                    FileIdentityLength: 0,
                },
            },
            connection_key,
            transfer_key,
        )
    }
}

/// Confirm deletion of the placeholder.
#[derive(Debug)]
pub struct Delete;

impl Command for Delete {
    const OPERATION: CF_OPERATION_TYPE = CloudFilters::CF_OPERATION_TYPE_ACK_DELETE;

    type Result = ();
    type Field = CF_OPERATION_PARAMETERS_0_2;

    unsafe fn result(_info: CF_OPERATION_PARAMETERS_0) -> Self::Result {}

    fn build(&self) -> CF_OPERATION_PARAMETERS_0 {
        CF_OPERATION_PARAMETERS_0 {
            AckDelete: CF_OPERATION_PARAMETERS_0_2 {
                Flags: CloudFilters::CF_OPERATION_ACK_DELETE_FLAG_NONE,
                CompletionStatus: Foundation::STATUS_SUCCESS,
            },
        }
    }
}

impl Fallible for Delete {
    fn fail(
        connection_key: RawConnectionKey,
        transfer_key: RawTransferKey,
        error_kind: CloudErrorKind,
    ) -> core::Result<Self::Result> {
        execute::<Self>(
            CF_OPERATION_PARAMETERS_0 {
                AckDelete: CF_OPERATION_PARAMETERS_0_2 {
                    Flags: CloudFilters::CF_OPERATION_ACK_DELETE_FLAG_NONE,
                    CompletionStatus: error_kind.into(),
                },
            },
            connection_key,
            transfer_key,
        )
    }
}

/// Confirm rename/move of the placeholder.
#[derive(Debug)]
pub struct Rename;

impl Command for Rename {
    const OPERATION: CF_OPERATION_TYPE = CloudFilters::CF_OPERATION_TYPE_ACK_RENAME;

    type Result = ();
    type Field = CF_OPERATION_PARAMETERS_0_3;

    unsafe fn result(_info: CF_OPERATION_PARAMETERS_0) -> Self::Result {}

    fn build(&self) -> CF_OPERATION_PARAMETERS_0 {
        CF_OPERATION_PARAMETERS_0 {
            AckRename: CF_OPERATION_PARAMETERS_0_3 {
                Flags: CloudFilters::CF_OPERATION_ACK_RENAME_FLAG_NONE,
                CompletionStatus: Foundation::STATUS_SUCCESS,
            },
        }
    }
}

impl Fallible for Rename {
    fn fail(
        connection_key: RawConnectionKey,
        transfer_key: RawTransferKey,
        error_kind: CloudErrorKind,
    ) -> core::Result<Self::Result> {
        execute::<Self>(
            CF_OPERATION_PARAMETERS_0 {
                AckRename: CF_OPERATION_PARAMETERS_0_3 {
                    Flags: CloudFilters::CF_OPERATION_ACK_RENAME_FLAG_NONE,
                    CompletionStatus: error_kind.into(),
                },
            },
            connection_key,
            transfer_key,
        )
    }
}
