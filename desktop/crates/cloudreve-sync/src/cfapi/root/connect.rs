use std::sync::Arc;
use anyhow::Result;
use windows::Win32::Storage::CloudFilters::{CF_CONNECTION_KEY, CfDisconnectSyncRoot};

use crate::cfapi::filter::{Callbacks, RawConnectionKey};

/// A handle to the current session for a given sync root.
///
/// [Connection] will disconnect when dropped. Note that this
/// does **NOT** mean the sync root will be unregistered. To do so, call
/// [SyncRootId::unregister][crate::root::SyncRootId::unregister].
#[derive(Debug, Clone)]
pub struct Connection<F> {
    connection_key: RawConnectionKey,

    _callbacks: Callbacks,
    filter: Arc<F>,
}

// this struct could house many more windows api functions, although they all seem to do nothing
// according to the threads on microsoft q&a
impl<T> Connection<T> {
    pub(crate) fn new(
        connection_key: RawConnectionKey,
        callbacks: Callbacks,
        filter: Arc<T>,
    ) -> Self {
        Self {
            connection_key,
            _callbacks: callbacks,
            filter,
        }
    }

    /// A raw connection key used to identify the connection.
    pub fn connection_key(&self) -> RawConnectionKey {
        self.connection_key
    }

    /// A reference to the inner [SyncFilter][crate::filter::SyncFilter] struct.
    pub fn filter(&self) -> &T {
        &self.filter
    }

    pub fn disconnect(&self) -> Result<()> {
        unsafe { CfDisconnectSyncRoot(CF_CONNECTION_KEY(self.connection_key)) }?;
        Ok(())
    }
}
