#![allow(clippy::missing_safety_doc)]

use std::sync::{Arc, Weak};

use windows::Win32::Storage::CloudFilters::{
    self, CF_CALLBACK_INFO, CF_CALLBACK_PARAMETERS, CF_CALLBACK_REGISTRATION,
};

use crate::cfapi::{
    command::{self, Fallible},
    filter::{Request, SyncFilter, info, ticket},
};

pub type Callbacks = [CF_CALLBACK_REGISTRATION; 14];

macro_rules! callbacks {
    ($([$type:path, $name:ident]),*) => {
        [
            $(
                CF_CALLBACK_REGISTRATION {
                    Type: $type,
                    Callback: Some($name::<T>),
                },
            )*
            CF_CALLBACK_REGISTRATION {
                Type: CloudFilters::CF_CALLBACK_TYPE_NONE,
                Callback: None,
            },
        ]
    };
}

// TODO: const this
pub fn callbacks<T: SyncFilter + 'static>() -> Callbacks {
    callbacks!(
        [CloudFilters::CF_CALLBACK_TYPE_FETCH_DATA, fetch_data],
        [CloudFilters::CF_CALLBACK_TYPE_VALIDATE_DATA, validate_data],
        [
            CloudFilters::CF_CALLBACK_TYPE_CANCEL_FETCH_DATA,
            cancel_fetch_data
        ],
        [
            CloudFilters::CF_CALLBACK_TYPE_FETCH_PLACEHOLDERS,
            fetch_placeholders
        ],
        [
            CloudFilters::CF_CALLBACK_TYPE_CANCEL_FETCH_PLACEHOLDERS,
            cancel_fetch_placeholders
        ],
        [
            CloudFilters::CF_CALLBACK_TYPE_NOTIFY_FILE_OPEN_COMPLETION,
            notify_file_open_completion
        ],
        [
            CloudFilters::CF_CALLBACK_TYPE_NOTIFY_FILE_CLOSE_COMPLETION,
            notify_file_close_completion
        ],
        [
            CloudFilters::CF_CALLBACK_TYPE_NOTIFY_DEHYDRATE,
            notify_dehydrate
        ],
        [
            CloudFilters::CF_CALLBACK_TYPE_NOTIFY_DEHYDRATE_COMPLETION,
            notify_dehydrate_completion
        ],
        [CloudFilters::CF_CALLBACK_TYPE_NOTIFY_DELETE, notify_delete],
        [
            CloudFilters::CF_CALLBACK_TYPE_NOTIFY_DELETE_COMPLETION,
            notify_delete_completion
        ],
        [CloudFilters::CF_CALLBACK_TYPE_NOTIFY_RENAME, notify_rename],
        [
            CloudFilters::CF_CALLBACK_TYPE_NOTIFY_RENAME_COMPLETION,
            notify_rename_completion
        ]
    )
}

pub unsafe extern "system" fn fetch_data<T: SyncFilter + 'static>(
    info: *const CF_CALLBACK_INFO,
    params: *const CF_CALLBACK_PARAMETERS,
) { unsafe {
    if let Some(filter) = filter_from_info::<T>(info) {
        let request = Request::new(*info);
        let connection_key = request.connection_key();
        let transfer_key = request.transfer_key();
        let ticket = ticket::FetchData::new(connection_key, transfer_key);

        let Err(e) = filter.fetch_data(
            request,
            ticket,
            info::FetchData((*params).Anonymous.FetchData),
        ) else {
            return;
        };

        if let Err(e) = command::Write::fail(connection_key, transfer_key, e) {
            tracing::error!(target: "cfapi::filter::proxy", "Failed to write data: {:?}", e);
        }
    }
}}

pub unsafe extern "system" fn validate_data<T: SyncFilter + 'static>(
    info: *const CF_CALLBACK_INFO,
    params: *const CF_CALLBACK_PARAMETERS,
) { unsafe {
    if let Some(filter) = filter_from_info::<T>(info) {
        let request = Request::new(*info);
        let connection_key = request.connection_key();
        let transfer_key = request.transfer_key();
        let ticket = ticket::ValidateData::new(connection_key, transfer_key);

        let Err(e) = filter.validate_data(
            request,
            ticket,
            info::ValidateData((*params).Anonymous.ValidateData),
        ) else {
            return;
        };

        if let Err(e) = command::Validate::fail(connection_key, transfer_key, e) {
            tracing::error!(target: "cfapi::filter::proxy", "Failed to validate data: {:?}", e);
        }
    }
}}

pub unsafe extern "system" fn cancel_fetch_data<T: SyncFilter + 'static>(
    info: *const CF_CALLBACK_INFO,
    params: *const CF_CALLBACK_PARAMETERS,
) { unsafe {
    if let Some(filter) = filter_from_info::<T>(info) {
        filter.cancel_fetch_data(
            Request::new(*info),
            info::CancelFetchData((*params).Anonymous.Cancel),
        );
    }
}}

pub unsafe extern "system" fn fetch_placeholders<T: SyncFilter + 'static>(
    info: *const CF_CALLBACK_INFO,
    params: *const CF_CALLBACK_PARAMETERS,
) { unsafe {
    if let Some(filter) = filter_from_info::<T>(info) {
        let request = Request::new(*info);
        let connection_key = request.connection_key();
        let transfer_key = request.transfer_key();
        let ticket = ticket::FetchPlaceholders::new(connection_key, transfer_key);

        let Err(e) = filter.fetch_placeholders(
            request,
            ticket,
            info::FetchPlaceholders((*params).Anonymous.FetchPlaceholders),
        ) else {
            return;
        };

        if let Err(e) = command::CreatePlaceholders::fail(connection_key, transfer_key, e) {
            tracing::error!(target: "cfapi::filter::proxy", "Failed to create placeholders: {:?}", e);
        }
    }
}}

pub unsafe extern "system" fn cancel_fetch_placeholders<T: SyncFilter + 'static>(
    info: *const CF_CALLBACK_INFO,
    params: *const CF_CALLBACK_PARAMETERS,
) { unsafe {
    if let Some(filter) = filter_from_info::<T>(info) {
        filter.cancel_fetch_placeholders(
            Request::new(*info),
            info::CancelFetchPlaceholders((*params).Anonymous.Cancel),
        );
    }
}}

pub unsafe extern "system" fn notify_file_open_completion<T: SyncFilter + 'static>(
    info: *const CF_CALLBACK_INFO,
    params: *const CF_CALLBACK_PARAMETERS,
) { unsafe {
    if let Some(filter) = filter_from_info::<T>(info) {
        filter.opened(
            Request::new(*info),
            info::Opened((*params).Anonymous.OpenCompletion),
        );
    }
}}

pub unsafe extern "system" fn notify_file_close_completion<T: SyncFilter + 'static>(
    info: *const CF_CALLBACK_INFO,
    params: *const CF_CALLBACK_PARAMETERS,
) { unsafe {
    if let Some(filter) = filter_from_info::<T>(info) {
        filter.closed(
            Request::new(*info),
            info::Closed((*params).Anonymous.CloseCompletion),
        );
    }
}}

pub unsafe extern "system" fn notify_dehydrate<T: SyncFilter + 'static>(
    info: *const CF_CALLBACK_INFO,
    params: *const CF_CALLBACK_PARAMETERS,
) { unsafe {
    if let Some(filter) = filter_from_info::<T>(info) {
        let request = Request::new(*info);
        let connection_key = request.connection_key();
        let transfer_key = request.transfer_key();
        let ticket = ticket::Dehydrate::new(connection_key, transfer_key);

        let Err(e) = filter.dehydrate(
            request,
            ticket,
            info::Dehydrate((*params).Anonymous.Dehydrate),
        ) else {
            return;
        };

        command::Dehydrate::fail(connection_key, transfer_key, e).unwrap();
    }
}}

pub unsafe extern "system" fn notify_dehydrate_completion<T: SyncFilter + 'static>(
    info: *const CF_CALLBACK_INFO,
    params: *const CF_CALLBACK_PARAMETERS,
) { unsafe {
    if let Some(filter) = filter_from_info::<T>(info) {
        filter.dehydrated(
            Request::new(*info),
            info::Dehydrated((*params).Anonymous.DehydrateCompletion),
        );
    }
}}

pub unsafe extern "system" fn notify_delete<T: SyncFilter + 'static>(
    info: *const CF_CALLBACK_INFO,
    params: *const CF_CALLBACK_PARAMETERS,
) { unsafe {
    if let Some(filter) = filter_from_info::<T>(info) {
        let request = Request::new(*info);
        let connection_key = request.connection_key();
        let transfer_key = request.transfer_key();
        let ticket = ticket::Delete::new(connection_key, transfer_key);

        let Err(e) = filter.delete(request, ticket, info::Delete((*params).Anonymous.Delete))
        else {
            return;
        };

        command::Delete::fail(connection_key, transfer_key, e).unwrap();
    }
}}

pub unsafe extern "system" fn notify_delete_completion<T: SyncFilter + 'static>(
    info: *const CF_CALLBACK_INFO,
    params: *const CF_CALLBACK_PARAMETERS,
) { unsafe {
    if let Some(filter) = filter_from_info::<T>(info) {
        filter.deleted(
            Request::new(*info),
            info::Deleted((*params).Anonymous.DeleteCompletion),
        );
    }
}}

pub unsafe extern "system" fn notify_rename<T: SyncFilter + 'static>(
    info: *const CF_CALLBACK_INFO,
    params: *const CF_CALLBACK_PARAMETERS,
) { unsafe {
    if let Some(filter) = filter_from_info::<T>(info) {
        let request = Request::new(*info);
        let connection_key = request.connection_key();
        let transfer_key = request.transfer_key();
        let ticket = ticket::Rename::new(connection_key, transfer_key);
        let info = info::Rename((*params).Anonymous.Rename, request.volume_letter());

        let Err(e) = filter.rename(request, ticket, info) else {
            return;
        };

        command::Rename::fail(connection_key, transfer_key, e).unwrap();
    }
}}

pub unsafe extern "system" fn notify_rename_completion<T: SyncFilter + 'static>(
    info: *const CF_CALLBACK_INFO,
    params: *const CF_CALLBACK_PARAMETERS,
) { unsafe {
    if let Some(filter) = filter_from_info::<T>(info) {
        let request = Request::new(*info);
        let info = info::Renamed(
            (*params).Anonymous.RenameCompletion,
            request.volume_letter(),
        );
        filter.renamed(request, info);
    }
}}

unsafe fn filter_from_info<T: SyncFilter + 'static>(
    info: *const CF_CALLBACK_INFO,
) -> Option<Arc<T>> { unsafe {
    // get the original weak arc
    let weak = Weak::from_raw((*info).CallbackContext as *mut T);
    // attempt to upgrade it to a strong arc
    match weak.upgrade() {
        // if the memory exists then the filter hasn't been disconnected
        Some(strong) => {
            // forget the original weak arc for next use
            let _ = Weak::into_raw(weak);
            // return the strong arc
            Some(strong)
        }
        // if the memory is freed then the filter is disconnected
        None => {
            // TODO: the weak arc should be freed on disconnect making this scenario
            // relatively impossible. Although, if I free the weak arc and by some chance
            // a callback is called, then it could be reading memory from a dangling pointer.
            // I could either introduce some mega hack to free the memory after x seconds or
            // suffer from a few bytes of memory leak..
            None
        }
    }
}}
