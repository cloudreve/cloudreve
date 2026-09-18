use std::{
    path::Path,
    sync::{
        Arc, Weak,
    },
};

use widestring::U16CString;
use windows::{
    Win32::Storage::CloudFilters::{self, CF_CONNECT_FLAGS, CfConnectSyncRoot},
    core::{self, PCWSTR},
};

use crate::cfapi::{
    filter::{self, AsyncBridge, Filter, SyncFilter},
    root::connect::Connection,
    utility::LocalBoxFuture,
};

/// A builder to create a new connection for the sync root at the specified path.
#[derive(Debug, Clone, Copy)]
pub struct Session(CF_CONNECT_FLAGS);

impl Session {
    /// Create a new [Session].
    pub fn new() -> Self {
        Self::default()
    }

    /// The [Session::block_implicit_hydration] flag will prevent
    /// implicit placeholder hydrations from invoking
    /// [SyncFilter::fetch_data][crate::filter::SyncFilter::fetch_data]. This could occur when an
    /// anti-virus is scanning file system activity on files within the sync root.
    ///
    /// A call to the [Placeholder::hydrate][crate::placeholder::Placeholder::hydrate] trait will not be blocked by this flag.
    pub fn block_implicit_hydration(mut self) -> Self {
        self.0 |= CloudFilters::CF_CONNECT_FLAG_BLOCK_SELF_IMPLICIT_HYDRATION;
        self
    }

    /// Initiates a connection to the sync root with the given [SyncFilter].
    pub fn connect<P, F>(self, path: P, filter: F) -> core::Result<Connection<F>>
    where
        P: AsRef<Path>,
        F: SyncFilter + 'static,
    {
        let filter = Arc::new(filter);
        let callbacks = filter::callbacks::<F>();
        tracing::trace!(target: "cfapi::root::session", "CfConnectSyncRoot enter");
        let key = unsafe {
            CfConnectSyncRoot(
                PCWSTR(
                    U16CString::from_os_str(path.as_ref())
                        .expect("not contains nul")
                        .as_ptr(),
                ),
                callbacks.as_ptr(),
                // create a weak arc so that it could be upgraded when it's being used and when the
                // connection is closed, the filter could be freed
                Some(Weak::into_raw(Arc::downgrade(&filter)) as *const _),
                // This is enabled by default to remove the Option requirement around various fields of the
                // [Request][crate::Request] struct
                self.0
                    | CloudFilters::CF_CONNECT_FLAG_REQUIRE_FULL_FILE_PATH
                    | CloudFilters::CF_CONNECT_FLAG_REQUIRE_PROCESS_INFO,
            )
        }?;

        Ok(Connection::new(key.0, callbacks, filter))
    }

    /// Initiates a connection to the sync root with the given [Filter].
    pub fn connect_async<P, F, B>(
        self,
        path: P,
        filter: F,
        block_on: B,
    ) -> core::Result<Connection<AsyncBridge<F, B>>>
    where
        P: AsRef<Path>,
        F: Filter + 'static,
        B: Fn(LocalBoxFuture<'_, ()>) + Send + Sync + 'static,
    {
        self.connect(path, AsyncBridge::new(filter, block_on))
    }
}

impl Default for Session {
    fn default() -> Self {
        Self(CloudFilters::CF_CONNECT_FLAG_NONE)
    }
}