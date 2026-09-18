use std::{future::Future, pin::Pin};

use windows::core::{self, HSTRING};

use crate::cfapi::sealed;

pub use nt_time::FileTime;

// TODO: add something to convert an Option<T> to a *const T and *mut T

pub(crate) trait ToHString
where
    Self: AsRef<[u16]>,
{
    /// Converts a 16-bit buffer to a Windows reference-counted [HSTRING][windows::core::HSTRING].
    ///
    /// # Panics
    ///
    /// Panics if [HeapAlloc](https://docs.microsoft.com/en-us/windows/win32/api/heapapi/nf-heapapi-heapalloc) fails.
    fn to_hstring(&self) -> HSTRING {
        HSTRING::from_wide(self.as_ref()).unwrap()
    }
}

impl<T: AsRef<[u16]>> ToHString for T {}

/// A trait for types that can read data from a file-like object at a specific offset.
///
/// This is a low-level interface that is used by Cloud Filter to implement the
/// [CfExecute](https://docs.microsoft.com/en-us/windows/win32/api/cfapi/nf-cfapi-cfexecute)
/// function's `CF_OPERATION_TYPE_RETRIEVE_DATA` operation.
///
/// You should not need to implement this trait yourself, but rather use the
/// [utility::ReadAt](crate::utility::ReadAt) trait as a bound for your argument.
pub trait ReadAt: sealed::Sealed {
    fn read_at(&self, buf: &mut [u8], offset: u64) -> core::Result<u64>;
}

/// A trait for types that can write data to a file-like object at a specific offset.
///
/// This is a low-level interface that is used by Cloud Filter to implement the
/// [CfExecute](https://docs.microsoft.com/en-us/windows/win32/api/cfapi/nf-cfapi-cfexecute)
/// function's `CF_OPERATION_TYPE_TRANSFER_DATA` operation.
///
/// You should not need to implement this trait yourself, but rather use the
/// [utility::WriteAt](crate::utility::WriteAt) trait as a bound for your argument.
pub trait WriteAt: sealed::Sealed {
    fn write_at(&self, buf: &[u8], offset: u64) -> core::Result<()>;
}

pub(crate) type LocalBoxFuture<'a, T> = Pin<Box<dyn Future<Output = T> + 'a>>;
