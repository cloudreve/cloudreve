use std::{mem, ptr};

use memoffset::offset_of;
use windows::{
    Win32::Storage::CloudFilters::{
        self, CF_CONNECTION_KEY, CF_OPERATION_INFO, CF_OPERATION_PARAMETERS,
        CF_OPERATION_PARAMETERS_0, CF_OPERATION_TYPE, CfExecute,
    },
    core,
};

use crate::cfapi::{
    error::CloudErrorKind,
    filter::{RawConnectionKey, RawTransferKey},
};

/// Allows a command to fail with a specified error.
pub trait Fallible: Command {
    fn fail(
        connection_key: RawConnectionKey,
        transfer_key: RawTransferKey,
        error_kind: CloudErrorKind,
    ) -> core::Result<Self::Result>;
}

/// A Cloud Filter command used to execute various functions.
pub trait Command: Sized {
    /// The [CF_OPERATION_TYPE][windows::Win32::Storage::CloudFilters::CF_OPERATION_TYPE]
    /// corresponding to the command.
    const OPERATION: CF_OPERATION_TYPE;
    /// The result of the command, if applicable.
    type Result;
    /// The union field of
    /// [CF_OPERATION_PARAMETERS_0][windows::Win32::Storage::CloudFilters::CF_OPERATION_PARAMETERS_0].
    type Field;

    /// Gathers and returns the result of the command.
    ///
    /// # Safety
    /// It is expected to index the union parameter, hence `unsafe`.
    unsafe fn result(info: CF_OPERATION_PARAMETERS_0) -> Self::Result;

    /// Creates and returns the union.
    fn build(&self) -> CF_OPERATION_PARAMETERS_0;

    /// Executes the command to the platform.
    fn execute(
        &self,
        connection_key: RawConnectionKey,
        transfer_key: RawTransferKey,
    ) -> core::Result<Self::Result> {
        execute::<Self>(self.build(), connection_key, transfer_key)
    }
}

pub fn execute<C: Command>(
    info: CF_OPERATION_PARAMETERS_0,
    connection_key: RawConnectionKey,
    transfer_key: RawTransferKey,
) -> core::Result<C::Result> {
    unsafe {
        CfExecute(
            &CF_OPERATION_INFO {
                StructSize: mem::size_of::<CF_OPERATION_INFO>() as u32,
                Type: C::OPERATION,
                ConnectionKey: CF_CONNECTION_KEY(connection_key),
                TransferKey: transfer_key,
                CorrelationVector: ptr::null(),
                SyncStatus: ptr::null(),
                // https://docs.microsoft.com/en-us/answers/questions/749979/what-is-a-requestkey-cfapi.html
                RequestKey: CloudFilters::CF_REQUEST_KEY_DEFAULT as i64,
            } as *const _,
            &mut CF_OPERATION_PARAMETERS {
                ParamSize: (mem::size_of::<C::Field>()
                    + offset_of!(CF_OPERATION_PARAMETERS, Anonymous))
                    as u32,
                Anonymous: info,
            } as *mut _,
        )
        .and(Ok(C::result(info)))
    }
}
