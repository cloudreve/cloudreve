use cloudreve_sync::events::Event;
use tauri::{AppHandle, Emitter};

use crate::commands::{show_add_drive_window_impl, show_main_window_center, show_settings_window_impl};

/// Handle incoming events from the event broadcaster.
/// Returns true if the event was handled, false otherwise.
pub fn handle_event(app_handle: &AppHandle, event: &Event) {
    match event {
        Event::NoDrive { .. } => handle_no_drive(app_handle),
        Event::ConnectionStatusChanged { .. } => {
            // Currently just forwarded to frontend via emit
        }
        Event::OpenSyncStatusWindow => handle_open_sync_status_window(app_handle),
        Event::OpenSettingsWindow => handle_open_settings_window(app_handle),
    }
}

fn handle_no_drive(app_handle: &AppHandle) {
    show_add_drive_window_impl(app_handle);
}

fn handle_open_sync_status_window(app_handle: &AppHandle) {
    // Open the main popup window which shows sync status
    show_main_window_center(app_handle);
}

fn handle_open_settings_window(app_handle: &AppHandle) {
    show_settings_window_impl(app_handle);
}

/// Emit an event to the frontend
pub fn emit_event(app_handle: &AppHandle, event: &Event) {
    if let Err(e) = app_handle.emit(event.name(), event) {
        tracing::error!(target: "events", error = %e, "Failed to emit event to frontend");
    } else {
        tracing::trace!(target: "events", event = ?event, "Event emitted to frontend");
    }
}
