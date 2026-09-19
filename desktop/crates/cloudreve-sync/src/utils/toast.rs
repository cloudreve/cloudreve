use std::path::PathBuf;

#[cfg(windows)]
use base64::{Engine as _, engine::general_purpose::URL_SAFE};
#[cfg(target_os = "macos")]
use mac_notification_sys::send_notification;
#[cfg(target_os = "linux")]
use notify_rust::{Notification, Timeout, Urgency};
#[cfg(windows)]
use win32_notif::{
    NotificationBuilder, ToastsNotifier,
    notification::{
        actions::{ActionButton, Input, input::Selection},
        visual::{Image, Placement, Text, text::HintStyle},
    },
};

use crate::config::ConfigManager;

#[cfg(windows)]
const APP_NAME: &str = "Cloudreve.Sync";
#[cfg(any(target_os = "linux", target_os = "macos"))]
const APP_NAME: &str = "Cloudreve";

#[cfg(target_os = "linux")]
fn send_linux_notification(title: &str, message: &str, urgency: Urgency) {
    match Notification::new()
        .appname(APP_NAME)
        .summary(title)
        .body(message)
        .icon("cloudreve")
        .urgency(urgency)
        .timeout(Timeout::Default)
        .show()
    {
        Ok(_) => tracing::debug!(target: "toast", title, message, "Linux notification sent"),
        Err(err) => tracing::warn!(
            target: "toast",
            title,
            message,
            error = %err,
            "Failed to send Linux desktop notification"
        ),
    }
}

#[cfg(target_os = "macos")]
fn send_macos_notification(title: &str, message: &str) {
    match send_notification(title, Some(APP_NAME), message, None) {
        Ok(_) => tracing::debug!(target: "toast", title, message, "macOS notification sent"),
        Err(err) => tracing::warn!(
            target: "toast",
            title,
            message,
            error = %err,
            "Failed to send macOS user notification"
        ),
    }
}

#[cfg(any(target_os = "linux", target_os = "macos"))]
fn conflict_notification_message(path: &PathBuf) -> String {
    let file_name = path
        .file_name()
        .unwrap_or_default()
        .to_string_lossy()
        .to_string();
    format!("{} - open Cloudreve to resolve the conflict.", file_name)
}

#[cfg(windows)]
pub fn send_general_text_toast(title: &str, message: &str) {
    let notifier = ToastsNotifier::new(APP_NAME).unwrap();

    let notif = NotificationBuilder::new()
        .visual(
            Text::create(1, title)
                .with_align_center(true)
                .with_wrap(true)
                .with_style(HintStyle::Title),
        )
        .visual(
            Text::create(2, message)
                .with_align_center(true)
                .with_wrap(true)
                .with_style(HintStyle::Body),
        )
        .build(0, &notifier, "01", "readme")
        .unwrap();

    notif.show().unwrap();
}

#[cfg(target_os = "linux")]
pub fn send_general_text_toast(title: &str, message: &str) {
    send_linux_notification(title, message, Urgency::Normal);
}

#[cfg(target_os = "macos")]
pub fn send_general_text_toast(title: &str, message: &str) {
    send_macos_notification(title, message);
}

#[cfg(not(any(windows, target_os = "linux", target_os = "macos")))]
pub fn send_general_text_toast(title: &str, message: &str) {
    tracing::info!(target: "toast", title, message, "Toast notification skipped on this platform");
}

/// Send a toast notification with a warning icon.
#[cfg(windows)]
pub fn send_warning_toast(title: &str, message: &str) {
    let notifier = ToastsNotifier::new(APP_NAME).unwrap();

    let notif = NotificationBuilder::new()
        .visual(
            Text::create(1, title)
                .with_align_center(true)
                .with_wrap(true)
                .with_style(HintStyle::Title),
        )
        .visual(
            Text::create(2, message)
                .with_align_center(true)
                .with_wrap(true)
                .with_style(HintStyle::Body),
        )
        .visual(
            Image::create(3, "ms-appx:///Images/warning.svg")
                .with_placement(Placement::AppLogoOverride),
        )
        .build(0, &notifier, "01", "warning")
        .unwrap();

    notif.show().unwrap();
}

#[cfg(target_os = "linux")]
pub fn send_warning_toast(title: &str, message: &str) {
    send_linux_notification(title, message, Urgency::Critical);
}

#[cfg(target_os = "macos")]
pub fn send_warning_toast(title: &str, message: &str) {
    send_macos_notification(title, message);
}

#[cfg(not(any(windows, target_os = "linux", target_os = "macos")))]
pub fn send_warning_toast(title: &str, message: &str) {
    tracing::warn!(target: "toast", title, message, "Warning toast skipped on this platform");
}

/// Send a toast notification for token expiry.
/// Uses drive_id as the tag to prevent duplicate notifications for the same drive.
/// Respects the notify_credential_expired config setting.
#[cfg(windows)]
pub fn send_token_expiry_toast(drive_id: &str, title: &str, message: &str) {
    // Check if credential expired notifications are enabled
    if let Some(config) = ConfigManager::try_get() {
        if !config.notify_credential_expired() {
            tracing::debug!(target: "toast", "Token expiry notification suppressed by config");
            return;
        }
    }

    let notifier = ToastsNotifier::new(APP_NAME).unwrap();

    let notif = NotificationBuilder::new()
        .visual(
            Text::create(1, title)
                .with_align_center(true)
                .with_wrap(true)
                .with_style(HintStyle::Title),
        )
        .visual(
            Text::create(2, message)
                .with_align_center(true)
                .with_wrap(true)
                .with_style(HintStyle::Body),
        )
        .visual(
            Image::create(3, "ms-appx:///Images/warning.svg")
                .with_placement(Placement::AppLogoOverride),
        )
        .with_launch("action=settings")
        .build(
            0,
            &notifier,
            &format!("token_expiry_{}", drive_id),
            "token_expiry",
        )
        .unwrap();

    notif.show().unwrap();
}

#[cfg(target_os = "linux")]
pub fn send_token_expiry_toast(_drive_id: &str, title: &str, message: &str) {
    if let Some(config) = ConfigManager::try_get() {
        if !config.notify_credential_expired() {
            tracing::debug!(target: "toast", "Token expiry notification suppressed by config");
            return;
        }
    }
    send_linux_notification(title, message, Urgency::Critical);
}

#[cfg(target_os = "macos")]
pub fn send_token_expiry_toast(_drive_id: &str, title: &str, message: &str) {
    if let Some(config) = ConfigManager::try_get() {
        if !config.notify_credential_expired() {
            tracing::debug!(target: "toast", "Token expiry notification suppressed by config");
            return;
        }
    }
    send_macos_notification(title, message);
}

#[cfg(not(any(windows, target_os = "linux", target_os = "macos")))]
pub fn send_token_expiry_toast(_drive_id: &str, title: &str, message: &str) {
    if let Some(config) = ConfigManager::try_get() {
        if !config.notify_credential_expired() {
            tracing::debug!(target: "toast", "Token expiry notification suppressed by config");
            return;
        }
    }
    tracing::warn!(target: "toast", title, message, "Token expiry toast skipped on this platform");
}

/// Send a toast notification for file conflicts.
/// Respects the notify_file_conflict config setting.
#[cfg(windows)]
pub fn send_conflict_toast(drive_id: &str, path: &PathBuf, inventory_id: i64) {
    // Check if file conflict notifications are enabled
    if let Some(config) = ConfigManager::try_get() {
        if !config.notify_file_conflict() {
            tracing::debug!(target: "toast", "Conflict notification suppressed by config");
            return;
        }
    }

    let notifier = ToastsNotifier::new(APP_NAME).unwrap();

    let notif = NotificationBuilder::new()
        .visual(
            Text::create(1, t!("conflictToastTitle").as_ref())
                .with_align_center(true)
                .with_wrap(true)
                .with_style(HintStyle::Title),
        )
        .visual(
            Text::create(
                2,
                path.file_name()
                    .unwrap_or_default()
                    .to_str()
                    .unwrap_or_default(),
            )
            .with_align_center(true)
            .with_wrap(true)
            .with_style(HintStyle::Body),
        )
        .actions(vec![
            Box::new(Input::create_selection_input(
                "selection",
                t!("selectAction").as_ref(),
                t!("selectAction").as_ref(),
                vec![
                    Selection::new("keep_remote", t!("acceptIncomming").as_ref()),
                    Selection::new("overwrite_remote", t!("overwriteRemote").as_ref()),
                    Selection::new("save_as_new", t!("saveAsNew").as_ref()),
                ],
                "keep_remote",
            )),
            Box::new(
                ActionButton::create(t!("resolveWithAction").as_ref())
                    .with_id(&format!(
                        "action=resolve&drive_id={}&file_id={}&path={}",
                        drive_id,
                        inventory_id,
                        URL_SAFE.encode(path.display().to_string())
                    ))
                    .with_tooltip(t!("resolveTooltip").as_ref()),
            ),
            Box::new(ActionButton::create(t!("dismiss").as_ref()).with_id("action=dismiss")),
        ])
        .build(
            0,
            &notifier,
            &format!("conflict_{}", inventory_id),
            "readme",
        )
        .unwrap();

    notif.show().unwrap();
}

#[cfg(target_os = "linux")]
pub fn send_conflict_toast(_drive_id: &str, path: &PathBuf, inventory_id: i64) {
    if let Some(config) = ConfigManager::try_get() {
        if !config.notify_file_conflict() {
            tracing::debug!(target: "toast", "Conflict notification suppressed by config");
            return;
        }
    }

    let message = conflict_notification_message(path);
    tracing::warn!(
        target: "toast",
        path = %path.display(),
        inventory_id,
        "Sending Linux conflict notification without Windows toast actions"
    );
    send_linux_notification(&t!("conflictToastTitle"), &message, Urgency::Critical);
}

#[cfg(target_os = "macos")]
pub fn send_conflict_toast(_drive_id: &str, path: &PathBuf, inventory_id: i64) {
    if let Some(config) = ConfigManager::try_get() {
        if !config.notify_file_conflict() {
            tracing::debug!(target: "toast", "Conflict notification suppressed by config");
            return;
        }
    }

    let message = conflict_notification_message(path);
    tracing::warn!(
        target: "toast",
        path = %path.display(),
        inventory_id,
        "Sending macOS conflict notification without Windows toast actions"
    );
    send_macos_notification(&t!("conflictToastTitle"), &message);
}

#[cfg(not(any(windows, target_os = "linux", target_os = "macos")))]
pub fn send_conflict_toast(_drive_id: &str, path: &PathBuf, inventory_id: i64) {
    if let Some(config) = ConfigManager::try_get() {
        if !config.notify_file_conflict() {
            tracing::debug!(target: "toast", "Conflict notification suppressed by config");
            return;
        }
    }
    tracing::warn!(
        target: "toast",
        path = %path.display(),
        inventory_id,
        "Conflict toast skipped on this platform"
    );
}
