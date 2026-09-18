use std::{future::Future, mem::MaybeUninit, ops::Deref, path::PathBuf};

use crate::cfapi::{
    error::{CResult, CloudErrorKind},
    filter::Request,
    utility::LocalBoxFuture,
};

use super::{SyncFilter, info, ticket};

/// Async core functions for implementing a Sync Engine.
///
/// [Send] and [Sync] are required as the callback could be invoked from an arbitrary thread, [read
/// here](https://docs.microsoft.com/en-us/windows/win32/api/cfapi/ne-cfapi-cf_callback_type#remarks).
pub trait Filter: Send + Sync {
    /// A placeholder hydration has been requested. This means that the placeholder should be
    /// populated with its corresponding data on the remote.
    fn fetch_data(
        &self,
        _request: Request,
        _ticket: ticket::FetchData,
        _info: info::FetchData,
    ) -> impl Future<Output = CResult<()>>;

    /// A placeholder hydration request has been cancelled.
    fn cancel_fetch_data(
        &self,
        _request: Request,
        _info: info::CancelFetchData,
    ) -> impl Future<Output = ()> {
        async {}
    }

    /// Followed by a successful call to [Filter::fetch_data][super::Filter::fetch_data], this callback should verify the integrity of
    /// the data persisted in the placeholder.
    ///
    /// **You** are responsible for validating the data in the placeholder. To approve
    /// the request, use the ticket provided.
    ///
    /// Note that this callback is only called if [HydrationPolicy::ValidationRequired][crate::root::HydrationPolicy::ValidationRequired]
    /// is specified.
    fn validate_data(
        &self,
        _request: Request,
        _ticket: ticket::ValidateData,
        _info: info::ValidateData,
    ) -> impl Future<Output = CResult<()>> {
        async { Err(CloudErrorKind::NotSupported) }
    }

    /// A directory population has been requested. The behavior of this callback is dependent on
    /// the [PopulationType][crate::root::PopulationType] variant specified during registration.
    fn fetch_placeholders(
        &self,
        _request: Request,
        _ticket: ticket::FetchPlaceholders,
        _info: info::FetchPlaceholders,
    ) -> impl Future<Output = CResult<()>> {
        async { Err(CloudErrorKind::NotSupported) }
    }

    /// A directory population request has been cancelled.
    fn cancel_fetch_placeholders(
        &self,
        _request: Request,
        _info: info::CancelFetchPlaceholders,
    ) -> impl Future<Output = ()> {
        async {}
    }

    /// A placeholder file handle has been opened for read, write, and/or delete
    /// access.
    fn opened(&self, _request: Request, _info: info::Opened) -> impl Future<Output = ()> {
        async {}
    }

    /// A placeholder file handle that has been previously opened with read, write,
    /// and/or delete access has been closed.
    fn closed(&self, _request: Request, _info: info::Closed) -> impl Future<Output = ()> {
        async {}
    }

    /// A placeholder dehydration has been requested. This means that all of the data persisted in
    /// the file will be __completely__ discarded.
    ///
    /// The operating system will handle dehydrating placeholder files automatically. However, it
    /// is up to **you** to approve this. Use the ticket to approve the request.
    fn dehydrate(
        &self,
        _request: Request,
        _ticket: ticket::Dehydrate,
        _info: info::Dehydrate,
    ) -> impl Future<Output = CResult<()>> {
        async { Err(CloudErrorKind::NotSupported) }
    }

    /// A placeholder dehydration request has been cancelled.
    fn dehydrated(&self, _request: Request, _info: info::Dehydrated) -> impl Future<Output = ()> {
        async {}
    }

    /// A placeholder file is about to be deleted.
    ///
    /// The operating system will handle deleting placeholder files automatically. However, it is
    /// up to **you** to approve this. Use the ticket to approve the request.
    fn delete(
        &self,
        _request: Request,
        _ticket: ticket::Delete,
        _info: info::Delete,
    ) -> impl Future<Output = CResult<()>> {
        async { Err(CloudErrorKind::NotSupported) }
    }

    /// A placeholder file has been deleted.
    fn deleted(&self, _request: Request, _info: info::Deleted) -> impl Future<Output = ()> {
        async {}
    }

    /// A placeholder file is about to be renamed or moved.
    ///
    /// The operating system will handle moving and renaming placeholder files automatically.
    /// However, it is up to **you** to approve this. Use the ticket to approve the
    /// request.
    ///
    /// When the operation is completed, the [Filter::renamed] callback will be called.
    fn rename(
        &self,
        _request: Request,
        _ticket: ticket::Rename,
        _info: info::Rename,
    ) -> impl Future<Output = CResult<()>> {
        async { Err(CloudErrorKind::NotSupported) }
    }

    /// A placeholder file has been renamed or moved.
    fn renamed(&self, _request: Request, _info: info::Renamed) -> impl Future<Output = ()> {
        async {}
    }

    /// Placeholder for changed attributes under the sync root.
    ///
    /// This callback is implemented using [ReadDirectoryChangesW](https://learn.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-readdirectorychangesw)
    /// so it is not provided by the `Cloud Filter APIs`.
    ///
    /// This callback is used to detect when a user pins or unpins a placeholder file, etc.
    ///
    /// See also [Cloud Files API Frequently Asked Questions](https://www.userfilesystem.com/programming/faq/).
    fn state_changed(&self, _changes: Vec<PathBuf>) -> impl Future<Output = ()> {
        async {}
    }
}

/// Adapts a [Filter] to the [SyncFilter] trait.
pub struct AsyncBridge<F, B> {
    filter: F,
    block_on: B,
}

impl<F, B> AsyncBridge<F, B>
where
    F: Filter,
    B: Fn(LocalBoxFuture<'_, ()>) + Send + Sync,
{
    pub(crate) fn new(filter: F, block_on: B) -> Self {
        Self { filter, block_on }
    }
}

impl<F, B> SyncFilter for AsyncBridge<F, B>
where
    F: Filter,
    B: Fn(LocalBoxFuture<'_, ()>) + Send + Sync,
{
    fn fetch_data(
        &self,
        request: Request,
        ticket: ticket::FetchData,
        info: info::FetchData,
    ) -> CResult<()> {
        tracing::trace!(target: "cfapi::filter::async_filter", "FetchData enter");
        let mut ret = MaybeUninit::zeroed();
        (self.block_on)(Box::pin(async {
            ret.write(self.filter.fetch_data(request, ticket, info).await);
        }));

        unsafe { ret.assume_init() }
    }

    fn cancel_fetch_data(&self, request: Request, info: info::CancelFetchData) {
        tracing::trace!(target: "cfapi::filter::async_filter", "CancelFetchData enter");
        (self.block_on)(Box::pin(self.filter.cancel_fetch_data(request, info)))
    }

    fn validate_data(
        &self,
        request: Request,
        ticket: ticket::ValidateData,
        info: info::ValidateData,
    ) -> CResult<()> {
        tracing::trace!(target: "cfapi::filter::async_filter", "ValidateData enter");
        let mut ret = MaybeUninit::zeroed();
        (self.block_on)(Box::pin(async {
            ret.write(self.filter.validate_data(request, ticket, info).await);
        }));

        unsafe { ret.assume_init() }
    }

    fn fetch_placeholders(
        &self,
        request: Request,
        ticket: ticket::FetchPlaceholders,
        info: info::FetchPlaceholders,
    ) -> CResult<()> {
        tracing::trace!(target: "cfapi::filter::async_filter", "FetchPlaceholders enter");
        let mut ret = MaybeUninit::zeroed();
        (self.block_on)(Box::pin(async {
            ret.write(self.filter.fetch_placeholders(request, ticket, info).await);
        }));

        unsafe { ret.assume_init() }
    }

    fn cancel_fetch_placeholders(&self, request: Request, info: info::CancelFetchPlaceholders) {
        tracing::trace!(target: "cfapi::filter::async_filter", "CancelFetchPlaceholders enter");
        (self.block_on)(Box::pin(
            self.filter.cancel_fetch_placeholders(request, info),
        ))
    }

    fn opened(&self, request: Request, info: info::Opened) {
        tracing::trace!(target: "cfapi::filter::async_filter", "Opened enter");
        (self.block_on)(Box::pin(self.filter.opened(request, info)))
    }

    fn closed(&self, request: Request, info: info::Closed) {
        tracing::trace!(target: "cfapi::filter::async_filter", "Closed enter");
        (self.block_on)(Box::pin(self.filter.closed(request, info)))
    }

    fn dehydrate(
        &self,
        request: Request,
        ticket: ticket::Dehydrate,
        info: info::Dehydrate,
    ) -> CResult<()> {
        tracing::trace!(target: "cfapi::filter::async_filter", "Dehydrate enter");
        let mut ret = MaybeUninit::zeroed();
        (self.block_on)(Box::pin(async {
            ret.write(self.filter.dehydrate(request, ticket, info).await);
        }));

        unsafe { ret.assume_init() }
    }

    fn dehydrated(&self, request: Request, info: info::Dehydrated) {
        tracing::trace!(target: "cfapi::filter::async_filter", "Dehydrated enter");
        (self.block_on)(Box::pin(self.filter.dehydrated(request, info)))
    }

    fn delete(&self, request: Request, ticket: ticket::Delete, info: info::Delete) -> CResult<()> {
        tracing::trace!(target: "cfapi::filter::async_filter", "Delete enter");
        let mut ret = MaybeUninit::zeroed();
        (self.block_on)(Box::pin(async {
            ret.write(self.filter.delete(request, ticket, info).await);
        }));

        unsafe { ret.assume_init() }
    }

    fn deleted(&self, request: Request, info: info::Deleted) {
        tracing::trace!(target: "cfapi::filter::async_filter", "Deleted enter");
        (self.block_on)(Box::pin(self.filter.deleted(request, info)))
    }

    fn rename(&self, request: Request, ticket: ticket::Rename, info: info::Rename) -> CResult<()> {
        tracing::trace!(target: "cfapi::filter::async_filter", "Rename enter");
        let mut ret = MaybeUninit::zeroed();
        (self.block_on)(Box::pin(async {
            ret.write(self.filter.rename(request, ticket, info).await);
        }));

        unsafe { ret.assume_init() }
    }

    fn renamed(&self, request: Request, info: info::Renamed) {
        tracing::trace!(target: "cfapi::filter::async_filter", "Renamed enter");
        (self.block_on)(Box::pin(self.filter.renamed(request, info)))
    }

    fn state_changed(&self, changes: Vec<PathBuf>) {
        tracing::trace!(target: "cfapi::filter::async_filter", "StateChanged enter");
        (self.block_on)(Box::pin(self.filter.state_changed(changes)))
    }
}

impl<F, B> Deref for AsyncBridge<F, B>
where
    F: Filter,
    B: Fn(LocalBoxFuture<'_, ()>) + Send + Sync,
{
    type Target = F;

    fn deref(&self) -> &Self::Target {
        &self.filter
    }
}
