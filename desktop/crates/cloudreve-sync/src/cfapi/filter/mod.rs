/// This module contains the structs that are used to pass information into the callbacks of the
/// [SyncFilter][crate::filter::SyncFilter] trait and the [Filter][crate::filter::Filter] trait.
pub mod info;

/// This module contains the structs that are used to represent the operation that the engine
/// needs to perform. They are used as parameters to the methods of the [SyncFilter][crate::filter::SyncFilter]
/// trait and the [Filter][crate::filter::Filter] trait.
pub mod ticket;

pub use async_filter::{AsyncBridge, Filter};
pub(crate) use proxy::{Callbacks, callbacks};
pub use request::{Process, Request};
pub(crate) use request::{RawConnectionKey, RawTransferKey};
pub use sync_filter::SyncFilter;

mod async_filter;
mod proxy;
mod request;
mod sync_filter;
