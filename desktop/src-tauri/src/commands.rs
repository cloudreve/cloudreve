use crate::AppStateHandle;
use base64::{engine::general_purpose::STANDARD as BASE64, Engine as _};
use chrono::{Duration, Utc};
use cloudreve_sync::drive::commands::ConflictAction;
use cloudreve_sync::{
    config::LogLevel, ConfigManager, Credentials, DriveConfig, DriveInfo, StatusSummary,
};
#[cfg(windows)]
use tauri::utils::{config::WindowEffectsConfig, WindowEffect};
#[cfg(target_os = "macos")]
use tauri::TitleBarStyle;
use tauri::{
    tray::TrayIcon,
    utils::config::Color,
    webview::{WebviewWindow, WebviewWindowBuilder},
    AppHandle, Manager, State, WebviewUrl,
};
#[cfg(windows)]
use tauri_plugin_frame::WebviewWindowExt;
use tauri_plugin_positioner::{Position, WindowExt};
use uuid::Uuid;
#[cfg(windows)]
use windows::ApplicationModel::{StartupTask, StartupTaskState};

/// Result type for Tauri commands
type CommandResult<T> = Result<T, String>;

/// Check if a path is a root drive (e.g., "C:\", "D:\", "E:\")
fn is_root_drive(path: &str) -> bool {
    let path = path.trim();
    let chars: Vec<char> = path.chars().collect();

    // Must start with a single ASCII letter followed by ':'
    if chars.len() < 2 || !chars[0].is_ascii_alphabetic() || chars[1] != ':' {
        return false;
    }

    // "C:" or "C:\" or "C:/" (with optional trailing slashes)
    let rest: String = chars[2..].iter().collect();
    let rest = rest.trim_end_matches(|c| c == '\\' || c == '/');
    rest.is_empty()
}

/// Get the URL with language query parameter appended
fn get_url_with_lang(base_path: &str) -> String {
    let locale = crate::get_effective_locale();
    if base_path.contains('?') {
        format!("{}&lng={}", base_path, locale)
    } else {
        format!("{}?lng={}", base_path, locale)
    }
}

/// Check if a directory is empty (has no files or subdirectories)
#[tauri::command]
pub async fn is_dir_empty(path: String) -> CommandResult<bool> {
    let p = std::path::Path::new(&path);
    if !p.exists() || !p.is_dir() {
        return Ok(true);
    }
    match std::fs::read_dir(p) {
        Ok(mut entries) => Ok(entries.next().is_none()),
        Err(_) => Ok(true),
    }
}

/// List all configured drives
#[tauri::command]
pub async fn list_drives(state: State<'_, AppStateHandle>) -> CommandResult<Vec<DriveConfig>> {
    let app_state = state
        .get()
        .ok_or_else(|| "App not yet initialized".to_string())?;
    Ok(app_state.drive_manager.list_drives().await)
}

#[derive(serde::Deserialize)]
pub struct AddDriveArgs {
    pub site_url: String,
    pub access_token: String,
    pub refresh_token: String,
    pub access_token_expires: u64,
    pub refresh_token_expires: u64,
    pub drive_name: String,
    pub remote_path: String,
    pub local_path: String,
    pub user_id: String,
    pub drive_id: Option<String>,
    /// "full" (default) mirrors everything; "ondemand" mounts a FUSE
    /// filesystem with hydrate-on-open. Only honored on Linux.
    pub sync_mode: Option<String>,
}

/// Add a new drive configuration
#[tauri::command]
pub async fn add_drive(
    state: State<'_, AppStateHandle>,
    config: AddDriveArgs,
) -> CommandResult<String> {
    let app_state = state
        .get()
        .ok_or_else(|| "App not yet initialized".to_string())?;

    // Validate local_path for new drives (not for reauthorization)
    if config.drive_id.is_none() && is_root_drive(&config.local_path) {
        return Err(t!("localPathCannotBeRootDrive").to_string());
    }

    // Normalize the site URL once so every consumer (API client, reauthorize
    // window, view-online links) sees the same canonical form.
    let site_url = cloudreve_sync::normalize_site_url(&config.site_url);

    // Convert relative expiry times (seconds) to absolute RFC3339 timestamps
    let now = Utc::now();
    let access_expires = (now + Duration::seconds(config.access_token_expires as i64)).to_rfc3339();
    let refresh_expires =
        (now + Duration::seconds(config.refresh_token_expires as i64)).to_rfc3339();

    let credentials = Credentials {
        access_token: Some(config.access_token),
        refresh_token: config.refresh_token,
        access_expires: Some(access_expires),
        refresh_expires,
    };

    // If drive_id is provided, update existing drive instead of creating a new one
    if let Some(drive_id) = config.drive_id {
        app_state
            .drive_manager
            .update_drive_credentials(
                &drive_id,
                config.drive_name,
                site_url.clone(),
                credentials,
                &config.user_id,
            )
            .await
            .map_err(|e| e.to_string())?;

        // Persist drive configurations
        app_state
            .drive_manager
            .persist()
            .await
            .map_err(|e| e.to_string())?;

        return Ok(drive_id);
    }

    // Generate a new UUID for a new drive
    let drive_id = Uuid::new_v4().to_string();

    let sync_mode = match config.sync_mode.as_deref() {
        Some("ondemand") => cloudreve_sync::drive::mounts::DriveSyncMode::OnDemand,
        _ => cloudreve_sync::drive::mounts::DriveSyncMode::Full,
    };

    let drive_config = DriveConfig {
        id: drive_id,
        name: config.drive_name,
        instance_url: site_url,
        remote_path: config.remote_path,
        credentials,
        sync_path: config.local_path.into(),
        icon_path: None,
        raw_icon_path: None,
        enabled: true,
        user_id: config.user_id,
        sync_mode,
        sync_root_id: None,
        ignore_patterns: Vec::new(),
        extra: Default::default(),
    };

    // Add drive to manager
    let id = app_state
        .drive_manager
        .add_drive(drive_config)
        .await
        .map_err(|e| e.to_string())?;

    // Persist drive configurations
    app_state
        .drive_manager
        .persist()
        .await
        .map_err(|e| e.to_string())?;

    Ok(id)
}

/// Remove a drive by ID
#[tauri::command]
pub async fn remove_drive(
    state: State<'_, AppStateHandle>,
    drive_id: String,
) -> CommandResult<Option<DriveConfig>> {
    let app_state = state
        .get()
        .ok_or_else(|| "App not yet initialized".to_string())?;

    let result = app_state
        .drive_manager
        .remove_drive(&drive_id)
        .await
        .map_err(|e| e.to_string())?;

    // Persist drive configurations after removal
    app_state
        .drive_manager
        .persist()
        .await
        .map_err(|e| e.to_string())?;

    Ok(result)
}

/// Manually reconnect a drive: restart its remote event listener with a
/// fresh retry backoff and trigger a full sync. Recovers drives stuck in
/// the long-retry wait after repeated connection failures.
#[tauri::command]
pub async fn reconnect_drive(
    state: State<'_, AppStateHandle>,
    drive_id: String,
) -> CommandResult<()> {
    let app_state = state
        .get()
        .ok_or_else(|| "App not yet initialized".to_string())?;
    let mount = app_state
        .drive_manager
        .get_drive(&drive_id)
        .await
        .ok_or_else(|| "Drive not found".to_string())?;
    mount
        .command_tx
        .send(cloudreve_sync::drive::commands::MountCommand::Reconnect)
        .map_err(|e| e.to_string())
}

/// Get ignore patterns for a drive
#[tauri::command]
pub async fn get_ignore_patterns(
    state: State<'_, AppStateHandle>,
    drive_id: String,
) -> CommandResult<Vec<String>> {
    let app_state = state
        .get()
        .ok_or_else(|| "App not yet initialized".to_string())?;
    app_state
        .drive_manager
        .get_ignore_patterns(&drive_id)
        .await
        .map_err(|e| e.to_string())
}

/// Set ignore patterns for a drive
#[tauri::command]
pub async fn set_ignore_patterns(
    state: State<'_, AppStateHandle>,
    drive_id: String,
    patterns: Vec<String>,
) -> CommandResult<()> {
    let app_state = state
        .get()
        .ok_or_else(|| "App not yet initialized".to_string())?;

    app_state
        .drive_manager
        .update_ignore_patterns(&drive_id, patterns)
        .await
        .map_err(|e| e.to_string())?;

    app_state
        .drive_manager
        .persist()
        .await
        .map_err(|e| e.to_string())?;

    Ok(())
}

/// Get sync status for a drive
#[tauri::command]
pub async fn get_sync_status(
    state: State<'_, AppStateHandle>,
    drive_id: String,
) -> CommandResult<serde_json::Value> {
    let app_state = state
        .get()
        .ok_or_else(|| "App not yet initialized".to_string())?;
    app_state
        .drive_manager
        .get_sync_status(&drive_id)
        .await
        .map_err(|e| e.to_string())
}

/// Get status summary including all drives and recent tasks
#[tauri::command]
pub async fn get_status_summary(
    state: State<'_, AppStateHandle>,
    drive_id: Option<String>,
) -> CommandResult<StatusSummary> {
    let app_state = state
        .get()
        .ok_or_else(|| "App not yet initialized".to_string())?;
    app_state
        .drive_manager
        .get_status_summary(drive_id.as_deref())
        .await
        .map_err(|e| e.to_string())
}

/// Resolve a pending local-vs-remote conflict.
#[tauri::command]
pub async fn resolve_conflict(
    state: State<'_, AppStateHandle>,
    drive_id: String,
    file_id: i64,
    path: String,
    action: String,
) -> CommandResult<()> {
    let app_state = state
        .get()
        .ok_or_else(|| "App not yet initialized".to_string())?;
    // Keep the frontend contract string-based so TS does not need to mirror the
    // Rust enum layout. The accepted values are the same action IDs used by the
    // Windows shell/toast resolver.
    let action = ConflictAction::from_str(&action)
        .ok_or_else(|| format!("Invalid conflict action: {action}"))?;
    let drive = app_state
        .drive_manager
        .get_drive(&drive_id)
        .await
        .ok_or_else(|| format!("Drive not found: {drive_id}"))?;

    drive
        .resolve_conflict(action, file_id, path)
        .await
        .map_err(|e| e.to_string())
}

/// Resolve all pending conflicts for a drive (or all drives if drive_id is None).
#[tauri::command]
pub async fn resolve_all_conflicts(
    state: State<'_, AppStateHandle>,
    drive_id: Option<String>,
    action: String,
) -> CommandResult<(usize, usize)> {
    let app_state = state
        .get()
        .ok_or_else(|| "App not yet initialized".to_string())?;
    let action = ConflictAction::from_str(&action)
        .ok_or_else(|| format!("Invalid conflict action: {action}"))?;

    let mut total_success = 0usize;
    let mut total_failed = 0usize;

    if let Some(id) = drive_id {
        let drive = app_state
            .drive_manager
            .get_drive(&id)
            .await
            .ok_or_else(|| format!("Drive not found: {id}"))?;
        let (s, f) = drive
            .resolve_all_conflicts(action)
            .await
            .map_err(|e| e.to_string())?;
        total_success += s;
        total_failed += f;
    } else {
        let drives = app_state.drive_manager.list_drives().await;
        for config in drives {
            if let Some(drive) = app_state.drive_manager.get_drive(&config.id).await {
                match drive.resolve_all_conflicts(action).await {
                    Ok((s, f)) => {
                        total_success += s;
                        total_failed += f;
                    }
                    Err(e) => {
                        tracing::error!(
                            target: "commands",
                            drive_id = %config.id,
                            error = %e,
                            "Failed to resolve all conflicts for drive"
                        );
                        // We intentionally continue with other drives rather
                        // than failing the whole batch.
                    }
                }
            }
        }
    }

    Ok((total_success, total_failed))
}

/// Get all drives with their status information for the settings UI
#[tauri::command]
pub async fn get_drives_info(state: State<'_, AppStateHandle>) -> CommandResult<Vec<DriveInfo>> {
    let app_state = state
        .get()
        .ok_or_else(|| "App not yet initialized".to_string())?;
    app_state
        .drive_manager
        .get_drives_info()
        .await
        .map_err(|e| e.to_string())
}

/// File icon response containing base64 encoded RGBA pixel data
#[derive(serde::Serialize)]
pub struct FileIconResponse {
    /// Base64 encoded RGBA pixel data
    pub data: String,
    /// Icon width in pixels
    pub width: u32,
    /// Icon height in pixels
    pub height: u32,
}

fn file_icon_to_response(icon: file_icon_provider::Icon) -> FileIconResponse {
    FileIconResponse {
        data: BASE64.encode(&icon.pixels),
        width: icon.width,
        height: icon.height,
    }
}

/// Get file icon for a given path
/// Returns base64 encoded RGBA pixel data with dimensions
#[tauri::command]
pub async fn get_file_icon(
    #[allow(unused_variables)] app: AppHandle,
    path: String,
    size: Option<u16>,
) -> CommandResult<FileIconResponse> {
    let icon_size = size.unwrap_or(32);

    #[cfg(target_os = "linux")]
    {
        // file_icon_provider uses GTK on Linux. GTK APIs must run on the main
        // thread, so do not use `spawn_blocking` here even though icon lookup can
        // be slow; doing so causes gtk::IconTheme to panic on worker threads.
        let (tx, rx) = tokio::sync::oneshot::channel();
        app.run_on_main_thread(move || {
            let result = file_icon_provider::get_file_icon(&path, icon_size)
                .map(file_icon_to_response)
                .map_err(|e| format!("Failed to get file icon: {:?}", e));
            let _ = tx.send(result);
        })
        .map_err(|e| format!("Failed to schedule file icon lookup: {}", e))?;

        return rx
            .await
            .map_err(|e| format!("File icon lookup was cancelled: {}", e))?;
    }

    #[cfg(not(target_os = "linux"))]
    {
        // Run the blocking icon retrieval in a separate thread
        let result = tokio::task::spawn_blocking(move || {
            file_icon_provider::get_file_icon(&path, icon_size)
        })
        .await
        .map_err(|e| format!("Task join error: {}", e))?
        .map_err(|e| format!("Failed to get file icon: {:?}", e))?;

        Ok(file_icon_to_response(result))
    }
}

/// Show or create the main window (positioned at tray center)
pub fn show_main_window(app: &AppHandle) {
    show_main_window_at_position(app, Position::TrayCenter);
}

/// Show or create the main window (positioned at bottom right)
pub fn show_main_window_center(app: &AppHandle) {
    show_main_window_at_position(app, Position::Center);
}

fn move_window_safely(window: &WebviewWindow, position: Position, label: &str) {
    // tauri-plugin-positioner assumes tray/monitor geometry is available and
    // can panic or error on some Linux desktop sessions when the tray position
    // is unknown. Probe the monitor first and fall back to centering so popup
    // display remains usable instead of crashing the runtime worker.
    match position {
        Position::Center => {
            if let Err(err) = window.center() {
                tracing::warn!(
                    target: "main",
                    window = label,
                    error = %err,
                    "Failed to center window"
                );
            }
        }
        position => match window.current_monitor() {
            Ok(Some(_)) => {
                if let Err(err) = window.move_window(position) {
                    tracing::warn!(
                        target: "main",
                        window = label,
                        error = %err,
                        "Failed to move window with positioner; falling back to center"
                    );
                    let _ = window.center();
                }
            }
            Ok(None) => {
                tracing::warn!(
                    target: "main",
                    window = label,
                    "Window has no current monitor; falling back to center"
                );
                let _ = window.center();
            }
            Err(err) => {
                tracing::warn!(
                    target: "main",
                    window = label,
                    error = %err,
                    "Failed to get current monitor; falling back to center"
                );
                let _ = window.center();
            }
        },
    }
}

fn apply_default_window_icon<'a>(
    builder: WebviewWindowBuilder<'a, tauri::Wry, AppHandle>,
    app: &'a AppHandle,
    label: &str,
) -> Option<WebviewWindowBuilder<'a, tauri::Wry, AppHandle>> {
    // Non-Windows desktops otherwise tend to show the Wayland/X11 default icon
    // for custom windows. Reusing Tauri's default icon keeps taskbar entries
    // consistent without hard-coding a platform-specific icon path here.
    let Some(icon) = app.default_window_icon() else {
        tracing::warn!(
            target: "main",
            window = label,
            "No default window icon is configured"
        );
        return Some(builder);
    };

    match builder.icon(icon.clone()) {
        Ok(builder) => Some(builder),
        Err(err) => {
            tracing::warn!(
                target: "main",
                window = label,
                error = %err,
                "Failed to set window icon"
            );
            None
        }
    }
}

/// Attach a handler that updates macOS Dock visibility when the window is
/// closed, destroyed, or loses focus. This is a per-window safeguard in
/// addition to the global `RunEvent::WindowEvent` handler because some window
/// state changes (e.g. clicking outside the add-drive/settings popup) are not
/// reliably reflected by `webview_windows()`/`is_visible()` immediately. The
/// check is retried several times to give AppKit time to catch up.
#[cfg(target_os = "macos")]
fn update_dock_on_window_close(window: &WebviewWindow) {
    let window_clone = window.clone();
    window.on_window_event(move |event| {
        if matches!(
            event,
            tauri::WindowEvent::CloseRequested { .. }
                | tauri::WindowEvent::Destroyed
                | tauri::WindowEvent::Focused(false)
        ) {
            crate::schedule_update_dock_visibility(&window_clone.app_handle().clone());
        }
    });
}

/// Internal function to show or create the main window at a specific position
fn show_main_window_at_position(app: &AppHandle, position: Position) {
    // Check if window already exists
    if let Some(window) = app.get_webview_window("main_popup") {
        move_window_safely(&window, position, "main_popup");
        let _ = window.show();
        let _ = window.unminimize();
        let _ = window.set_focus();
        #[cfg(target_os = "macos")]
        crate::update_dock_visibility(app);
        return;
    }

    // Create new main window
    let builder = WebviewWindowBuilder::new(
        app,
        "main_popup",
        WebviewUrl::App(get_url_with_lang("index.html/#/popup").into()),
    )
    .title("Cloudreve")
    .inner_size(370.0, 530.0)
    .resizable(false)
    .visible(false)
    .decorations(false)
    .skip_taskbar(true)
    .minimizable(false);

    #[cfg(not(windows))]
    // Transparent webviews render differently across GTK/WebKit and AppKit; on
    // non-Windows this produced unreadable shadows/ghosting. Use an opaque
    // white background there and keep Windows transparency for Mica/Acrylic.
    let builder = builder.background_color(Color(255, 255, 255, 255));

    let Some(builder) = apply_default_window_icon(builder, app, "main_popup") else {
        return;
    };

    match builder.build() {
        Ok(window) => {
            #[cfg(target_os = "macos")]
            update_dock_on_window_close(&window);

            // Set up window event handlers for macOS:
            // - CloseRequested: when fast popup launch is enabled, hide instead of
            //   destroying so the popup can be reshown quickly.
            // - Focused(false): dismiss the popup when clicking outside and update
            //   Dock visibility immediately.
            let window_for_events = window.clone();
            window.on_window_event(move |event| {
                match event {
                    tauri::WindowEvent::CloseRequested { api, .. } => {
                        if ConfigManager::get().fast_popup_launch() {
                            api.prevent_close();
                            let _ = window_for_events.hide();
                            // The window is hidden rather than destroyed; make
                            // sure the Dock icon is re-evaluated repeatedly.
                            #[cfg(target_os = "macos")]
                            crate::schedule_update_dock_visibility(
                                &window_for_events.app_handle().clone(),
                            );
                        }
                    }
                    #[cfg(target_os = "macos")]
                    tauri::WindowEvent::Focused(false) => {
                        let _ = window_for_events.hide();
                        // Schedule multiple Dock visibility checks because the
                        // window state reported by Tauri can lag behind AppKit.
                        crate::schedule_update_dock_visibility(
                            &window_for_events.app_handle().clone(),
                        );
                    }
                    _ => {}
                }
            });

            move_window_safely(&window, position, "main_popup");
            let _ = window.show();
            let _ = window.set_focus();
            #[cfg(target_os = "macos")]
            crate::update_dock_visibility(app);
        }
        Err(e) => {
            tracing::error!(target: "main_popup", error = %e, "Failed to create main window");
        }
    }
}

/// Show a file in the system file explorer (Windows Explorer, Finder, etc.)
/// This will open the parent folder and select/highlight the file.
#[tauri::command]
pub async fn show_file_in_explorer(path: String) -> CommandResult<()> {
    showfile::show_path_in_file_manager(&path);
    Ok(())
}

/// Command to show the add-drive window
#[tauri::command]
pub async fn show_add_drive_window(app: AppHandle) -> CommandResult<()> {
    show_add_drive_window_impl(&app);
    Ok(())
}

/// Command to show the reauthorize window for a specific drive
#[tauri::command]
pub async fn show_reauthorize_window(
    app: AppHandle,
    drive_id: String,
    site_url: String,
    drive_name: String,
) -> CommandResult<()> {
    show_reauthorize_window_impl(&app, &drive_id, &site_url, &drive_name);
    Ok(())
}

/// Show or create the add-drive window
pub fn show_add_drive_window_impl(app: &AppHandle) {
    show_drive_window_internal(
        app,
        "Add Drive",
        &get_url_with_lang("index.html/#/add-drive"),
    );
}

/// Show or create the reauthorize window for a specific drive
pub fn show_reauthorize_window_impl(
    app: &AppHandle,
    drive_id: &str,
    site_url: &str,
    drive_name: &str,
) {
    // URL encode the site_url to safely pass it in the route
    let encoded_site_url = urlencoding::encode(site_url);
    let encoded_drive_name = urlencoding::encode(drive_name);
    let url_path = format!(
        "index.html/#/reauthorize/{}/{}/{}",
        drive_id, encoded_site_url, encoded_drive_name
    );
    show_drive_window_internal(app, "Reauthorize Drive", &get_url_with_lang(&url_path));
}

/// Internal function to show or create the add-drive/reauthorize window
fn show_drive_window_internal(app: &AppHandle, title: &str, url_path: &str) {
    // Check if window already exists
    if let Some(window) = app.get_webview_window("add-drive") {
        let _ = window.show();
        let _ = window.unminimize();
        let _ = window.set_focus();
        #[cfg(target_os = "macos")]
        crate::update_dock_visibility(app);
        return;
    }

    // Create new window with mica effect on Windows only.
    #[cfg(windows)]
    let effects = WindowEffectsConfig {
        effects: vec![WindowEffect::Mica, WindowEffect::Acrylic],
        state: None,
        radius: None,
        color: None,
    };

    let builder = WebviewWindowBuilder::new(app, "add-drive", WebviewUrl::App(url_path.into()))
        .title(title)
        .inner_size(470.0, 630.0)
        .resizable(false)
        .visible(false)
        .decorations(false)
        .minimizable(false);

    #[cfg(not(windows))]
    // See `show_main_window_at_position`: keep non-Windows webviews opaque to
    // avoid platform compositor artifacts while preserving Windows effects.
    let builder = builder.background_color(Color(255, 255, 255, 255));

    // Windows Mica/Acrylic effects require the window to be created transparent
    // (WebviewWindow has no post-build transparency switch in Tauri 2).
    #[cfg(windows)]
    let builder = builder.transparent(true);

    // Platform-specific: title_bar_style and hidden_title are macOS-only
    #[cfg(target_os = "macos")]
    let builder = builder
        .title_bar_style(TitleBarStyle::Overlay)
        .hidden_title(true);

    let Some(builder) = apply_default_window_icon(builder, app, "add-drive") else {
        return;
    };

    match builder.build() {
        Ok(window) => {
            #[cfg(target_os = "macos")]
            update_dock_on_window_close(&window);

            #[cfg(windows)]
            {
                let _ = window.set_effects(effects);
            }

            move_window_safely(&window, Position::Center, "add-drive");
            #[cfg(windows)]
            let _ = window.create_overlay_titlebar();
            let _ = window.show();
            let _ = window.set_focus();
            #[cfg(target_os = "macos")]
            crate::update_dock_visibility(app);
        }
        Err(e) => {
            tracing::error!(target: "main", error = %e, "Failed to create window: {}", title);
        }
    }
}

/// Command to show the settings window
#[tauri::command]
pub async fn show_settings_window(app: AppHandle) -> CommandResult<()> {
    show_settings_window_impl(&app);
    Ok(())
}

/// Show or create the settings window
pub fn show_settings_window_impl(app: &AppHandle) {
    // Check if window already exists
    if let Some(window) = app.get_webview_window("settings") {
        let _ = window.show();
        let _ = window.unminimize();
        let _ = window.set_focus();
        #[cfg(target_os = "macos")]
        crate::update_dock_visibility(app);
        return;
    }

    // Create new window with mica effect on Windows only.
    #[cfg(windows)]
    let effects = WindowEffectsConfig {
        effects: vec![WindowEffect::Mica, WindowEffect::Acrylic],
        state: None,
        radius: None,
        color: None,
    };

    let builder = WebviewWindowBuilder::new(
        app,
        "settings",
        WebviewUrl::App(get_url_with_lang("index.html/#/settings").into()),
    )
    .title("Settings")
    .inner_size(700.0, 500.0)
    .min_inner_size(600.0, 400.0)
    .visible(false)
    .resizable(true)
    .decorations(false)
    .minimizable(true);

    #[cfg(not(windows))]
    // See `show_main_window_at_position`: keep non-Windows webviews opaque to
    // avoid platform compositor artifacts while preserving Windows effects.
    let builder = builder.background_color(Color(255, 255, 255, 255));

    // Windows Mica/Acrylic effects require the window to be created transparent
    // (WebviewWindow has no post-build transparency switch in Tauri 2).
    #[cfg(windows)]
    let builder = builder.transparent(true);

    // Platform-specific: title_bar_style and hidden_title are macOS-only
    #[cfg(target_os = "macos")]
    let builder = builder
        .title_bar_style(TitleBarStyle::Overlay)
        .hidden_title(true);

    let Some(builder) = apply_default_window_icon(builder, app, "settings") else {
        return;
    };

    match builder.build() {
        Ok(window) => {
            #[cfg(target_os = "macos")]
            update_dock_on_window_close(&window);

            #[cfg(windows)]
            {
                let _ = window.set_effects(effects);
            }

            move_window_safely(&window, Position::Center, "settings");
            #[cfg(windows)]
            let _ = window.create_overlay_titlebar();
            let _ = window.show();
            let _ = window.set_focus();
            #[cfg(target_os = "macos")]
            crate::update_dock_visibility(app);
        }
        Err(e) => {
            tracing::error!(target: "main", error = %e, "Failed to create settings window");
        }
    }
}

/// Show or create the share dialog window for a file or folder.
pub fn show_share_window_impl(
    app: &AppHandle,
    drive_id: &str,
    uri: &str,
    name: &str,
    is_dir: bool,
) {
    let url_path = format!(
        "index.html#/share?drive={}&uri={}&name={}&dir={}",
        urlencoding::encode(drive_id),
        urlencoding::encode(uri),
        urlencoding::encode(name),
        if is_dir { "1" } else { "0" },
    );
    let url_path = get_url_with_lang(&url_path);

    // A stale window could belong to a different item; rebuild it each time.
    if let Some(window) = app.get_webview_window("share") {
        let _ = window.destroy();
    }

    #[cfg(windows)]
    let effects = WindowEffectsConfig {
        effects: vec![WindowEffect::Mica, WindowEffect::Acrylic],
        state: None,
        radius: None,
        color: None,
    };

    let builder = WebviewWindowBuilder::new(app, "share", WebviewUrl::App(url_path.into()))
        .title("Share link")
        .inner_size(480.0, 640.0)
        .resizable(false)
        .visible(false)
        .decorations(false)
        .minimizable(false);

    #[cfg(not(windows))]
    // See `show_main_window_at_position`: keep non-Windows webviews opaque to
    // avoid platform compositor artifacts while preserving Windows effects.
    let builder = builder.background_color(Color(255, 255, 255, 255));

    #[cfg(windows)]
    let builder = builder.transparent(true);

    #[cfg(target_os = "macos")]
    let builder = builder
        .title_bar_style(TitleBarStyle::Overlay)
        .hidden_title(true);

    let Some(builder) = apply_default_window_icon(builder, app, "share") else {
        return;
    };

    match builder.build() {
        Ok(window) => {
            #[cfg(target_os = "macos")]
            update_dock_on_window_close(&window);

            #[cfg(windows)]
            {
                let _ = window.set_effects(effects);
            }

            move_window_safely(&window, Position::Center, "share");
            #[cfg(windows)]
            let _ = window.create_overlay_titlebar();
            let _ = window.show();
            let _ = window.set_focus();
            #[cfg(target_os = "macos")]
            crate::update_dock_visibility(app);
        }
        Err(e) => {
            tracing::error!(target: "main", error = %e, "Failed to create share window");
        }
    }
}

/// Options accepted by the share dialog, mapped onto the server's share
/// creation fields.
#[derive(Debug, serde::Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ShareOptions {
    pub password: Option<String>,
    pub downloads: Option<i32>,
    /// Seconds until expiry.
    pub expire: Option<i32>,
    pub preview_only: Option<bool>,
    pub allow_edit: Option<bool>,
    pub allow_upload: Option<bool>,
    pub upload_only: Option<bool>,
}

/// Create a share link from the share dialog, returning the share URL.
#[tauri::command]
pub async fn create_share(
    state: State<'_, AppStateHandle>,
    drive_id: String,
    uri: String,
    options: ShareOptions,
) -> CommandResult<String> {
    let app_state = state
        .get()
        .ok_or_else(|| "App not yet initialized".to_string())?;

    let mount = app_state
        .drive_manager
        .get_drive(&drive_id)
        .await
        .ok_or_else(|| "Drive not found".to_string())?;

    let password = options.password.filter(|p| !p.is_empty());
    let request = cloudreve_sync::ShareCreateService {
        uri,
        is_private: password.is_some(),
        password,
        downloads: options.downloads.filter(|d| *d > 0),
        expire: options.expire.filter(|e| *e > 0),
        preview_only: options.preview_only.unwrap_or(false),
        allow_edit: options.allow_edit.unwrap_or(false),
        allow_upload: options.allow_upload.unwrap_or(false),
        upload_only: options.upload_only.unwrap_or(false),
    };

    mount
        .create_share(request)
        .await
        .map_err(|e| e.to_string())
}

/// The TaskId defined in AppxManifest.xml for the startup task
#[cfg(windows)]
const STARTUP_TASK_ID: &str = "cloudreve";
#[cfg(target_os = "linux")]
const AUTOSTART_DESKTOP_FILE: &str = "cloudreve-desktop.desktop";
#[cfg(target_os = "macos")]
const MACOS_LAUNCH_AGENT_FILE: &str = "cloudreve.desktop.plist";
#[cfg(target_os = "macos")]
const MACOS_LAUNCH_AGENT_LABEL: &str = "cloudreve.desktop";

#[cfg(target_os = "linux")]
fn linux_autostart_path() -> CommandResult<std::path::PathBuf> {
    let config_home = std::env::var_os("XDG_CONFIG_HOME")
        .map(std::path::PathBuf::from)
        .or_else(|| {
            std::env::var_os("HOME")
                .map(std::path::PathBuf::from)
                .map(|home| home.join(".config"))
        })
        .ok_or_else(|| "Unable to determine XDG config directory".to_string())?;

    Ok(config_home.join("autostart").join(AUTOSTART_DESKTOP_FILE))
}

#[cfg(target_os = "linux")]
fn linux_desktop_exec_quote(value: &str) -> String {
    let escaped = value
        .replace('\\', "\\\\")
        .replace('"', "\\\"")
        .replace('$', "\\$")
        .replace('`', "\\`")
        .replace('\n', "")
        .replace('\r', "");
    format!("\"{}\"", escaped)
}

#[cfg(target_os = "linux")]
fn linux_autostart_entry() -> CommandResult<String> {
    let exe = std::env::current_exe()
        .map_err(|e| format!("Failed to get current executable path: {}", e))?;
    let exec = linux_desktop_exec_quote(&exe.display().to_string());

    Ok(format!(
        "[Desktop Entry]\n\
         Type=Application\n\
         Version=1.0\n\
         Name=Cloudreve\n\
         Comment=Cloudreve Desktop Sync Client\n\
         Exec={}\n\
         Terminal=false\n\
         X-GNOME-Autostart-enabled=true\n\
         Hidden=false\n",
        exec
    ))
}

#[cfg(target_os = "linux")]
fn linux_get_auto_start_enabled() -> CommandResult<bool> {
    let path = linux_autostart_path()?;
    let content = match std::fs::read_to_string(path) {
        Ok(content) => content,
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => return Ok(false),
        Err(err) => return Err(format!("Failed to read autostart entry: {}", err)),
    };

    let disabled = content.lines().any(|line| {
        matches!(
            line.trim(),
            "Hidden=true" | "X-GNOME-Autostart-enabled=false"
        )
    });
    Ok(!disabled)
}

#[cfg(target_os = "linux")]
fn linux_set_auto_start(enabled: bool) -> CommandResult<bool> {
    let path = linux_autostart_path()?;

    if enabled {
        let parent = path
            .parent()
            .ok_or_else(|| "Invalid autostart entry path".to_string())?;
        std::fs::create_dir_all(parent)
            .map_err(|e| format!("Failed to create autostart directory: {}", e))?;
        std::fs::write(&path, linux_autostart_entry()?)
            .map_err(|e| format!("Failed to write autostart entry: {}", e))?;
        Ok(true)
    } else {
        match std::fs::remove_file(&path) {
            Ok(_) => Ok(false),
            Err(err) if err.kind() == std::io::ErrorKind::NotFound => Ok(false),
            Err(err) => Err(format!("Failed to remove autostart entry: {}", err)),
        }
    }
}

#[cfg(target_os = "macos")]
fn macos_launch_agent_path() -> CommandResult<std::path::PathBuf> {
    let home = std::env::var_os("HOME")
        .map(std::path::PathBuf::from)
        .ok_or_else(|| "Unable to determine home directory".to_string())?;
    Ok(home
        .join("Library")
        .join("LaunchAgents")
        .join(MACOS_LAUNCH_AGENT_FILE))
}

#[cfg(target_os = "macos")]
fn macos_plist_escape(value: &str) -> String {
    value
        .replace('&', "&amp;")
        .replace('<', "&lt;")
        .replace('>', "&gt;")
        .replace('"', "&quot;")
        .replace('\'', "&apos;")
        .replace('\n', "")
        .replace('\r', "")
}

#[cfg(target_os = "macos")]
fn macos_launch_agent_entry() -> CommandResult<String> {
    let exe = std::env::current_exe()
        .map_err(|e| format!("Failed to get current executable path: {}", e))?;
    let exe = macos_plist_escape(&exe.display().to_string());
    let label = macos_plist_escape(MACOS_LAUNCH_AGENT_LABEL);

    Ok(format!(
        r#"<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>{}</string>
  <key>ProgramArguments</key>
  <array>
    <string>{}</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>LimitLoadToSessionType</key>
  <string>Aqua</string>
</dict>
</plist>
"#,
        label, exe
    ))
}

#[cfg(target_os = "macos")]
fn macos_get_auto_start_enabled() -> CommandResult<bool> {
    let path = macos_launch_agent_path()?;
    let content = match std::fs::read_to_string(path) {
        Ok(content) => content,
        Err(err) if err.kind() == std::io::ErrorKind::NotFound => return Ok(false),
        Err(err) => return Err(format!("Failed to read LaunchAgent: {}", err)),
    };

    // Verify the plist is ours and configured to start at login.
    let has_label = content.contains(&format!("<string>{}</string>", MACOS_LAUNCH_AGENT_LABEL));
    let run_at_load = content.contains("<key>RunAtLoad</key>") && content.contains("<true/>");
    Ok(has_label && run_at_load)
}

#[cfg(target_os = "macos")]
fn macos_set_auto_start(enabled: bool) -> CommandResult<bool> {
    let path = macos_launch_agent_path()?;

    if enabled {
        let parent = path
            .parent()
            .ok_or_else(|| "Invalid LaunchAgent path".to_string())?;
        std::fs::create_dir_all(parent)
            .map_err(|e| format!("Failed to create LaunchAgents directory: {}", e))?;
        std::fs::write(&path, macos_launch_agent_entry()?)
            .map_err(|e| format!("Failed to write LaunchAgent: {}", e))?;

        // Load the agent so it applies to the current session and is enabled
        // for future logins. Ignore errors: launchctl may fail if the agent is
        // already loaded, which still leaves the plist in place for next boot.
        let path_str = path.display().to_string();
        let _ = std::process::Command::new("/bin/launchctl")
            .args(["load", "-w", &path_str])
            .output();

        Ok(true)
    } else {
        // Only remove the plist. We intentionally do not `launchctl unload`
        // because if the current process was launched by this agent, unload
        // would terminate the running app. The agent will not be started on
        // the next login since the plist is gone.
        match std::fs::remove_file(&path) {
            Ok(_) => Ok(false),
            Err(err) if err.kind() == std::io::ErrorKind::NotFound => Ok(false),
            Err(err) => Err(format!("Failed to remove LaunchAgent: {}", err)),
        }
    }
}

/// Get whether auto-start is enabled using Windows StartupTask API
#[tauri::command]
pub async fn get_auto_start_enabled() -> CommandResult<bool> {
    #[cfg(target_os = "macos")]
    {
        return tokio::task::spawn_blocking(macos_get_auto_start_enabled)
            .await
            .map_err(|e| format!("Task join error: {}", e))?;
    }

    #[cfg(target_os = "linux")]
    {
        return tokio::task::spawn_blocking(linux_get_auto_start_enabled)
            .await
            .map_err(|e| format!("Task join error: {}", e))?;
    }

    #[cfg(not(any(windows, target_os = "linux", target_os = "macos")))]
    {
        Ok(false)
    }

    #[cfg(windows)]
    {
        tokio::task::spawn_blocking(|| {
            let task_id: windows::core::HSTRING = STARTUP_TASK_ID.into();
            let task = StartupTask::GetAsync(&task_id)
                .map_err(|e| format!("Failed to get startup task: {}", e))?
                .get()
                .map_err(|e| format!("Failed to get startup task: {}", e))?;

            let state = task
                .State()
                .map_err(|e| format!("Failed to get task state: {}", e))?;

            Ok(matches!(
                state,
                StartupTaskState::Enabled | StartupTaskState::EnabledByPolicy
            ))
        })
        .await
        .map_err(|e| format!("Task join error: {}", e))?
    }
}

/// Set auto-start configuration using Windows StartupTask API
#[tauri::command]
pub async fn set_auto_start(enabled: bool) -> CommandResult<bool> {
    #[cfg(target_os = "macos")]
    {
        return tokio::task::spawn_blocking(move || macos_set_auto_start(enabled))
            .await
            .map_err(|e| format!("Task join error: {}", e))?;
    }

    #[cfg(target_os = "linux")]
    {
        return tokio::task::spawn_blocking(move || linux_set_auto_start(enabled))
            .await
            .map_err(|e| format!("Task join error: {}", e))?;
    }

    #[cfg(not(any(windows, target_os = "linux", target_os = "macos")))]
    {
        Err("Auto-start configuration is not supported on this platform yet".to_string())
    }

    #[cfg(windows)]
    {
        tokio::task::spawn_blocking(move || {
            let task_id: windows::core::HSTRING = STARTUP_TASK_ID.into();
            let task = StartupTask::GetAsync(&task_id)
                .map_err(|e| format!("Failed to get startup task: {}", e))?
                .get()
                .map_err(|e| format!("Failed to get startup task: {}", e))?;

            if enabled {
                // Request enable - may prompt user for consent
                let new_state = task
                    .RequestEnableAsync()
                    .map_err(|e| format!("Failed to request enable: {}", e))?
                    .get()
                    .map_err(|e| format!("Failed to enable startup task: {}", e))?;

                Ok(matches!(
                    new_state,
                    StartupTaskState::Enabled | StartupTaskState::EnabledByPolicy
                ))
            } else {
                task.Disable()
                    .map_err(|e| format!("Failed to disable startup task: {}", e))?;
                Ok(false)
            }
        })
        .await
        .map_err(|e| format!("Task join error: {}", e))?
    }
}

#[cfg(all(test, target_os = "linux"))]
mod linux_autostart_tests {
    use super::linux_desktop_exec_quote;

    #[test]
    fn quotes_exec_paths_for_desktop_entries() {
        assert_eq!(
            linux_desktop_exec_quote("/opt/Cloudreve Desktop/cloudreve"),
            "\"/opt/Cloudreve Desktop/cloudreve\""
        );
    }

    #[test]
    fn escapes_shell_sensitive_exec_characters() {
        assert_eq!(
            linux_desktop_exec_quote("/tmp/cloudreve\"$`\\bin"),
            "\"/tmp/cloudreve\\\"\\$\\`\\\\bin\""
        );
    }
}

/// Set notification settings for credential expiry
#[tauri::command]
pub async fn set_notify_credential_expired(enabled: bool) -> CommandResult<()> {
    ConfigManager::get()
        .set_notify_credential_expired(enabled)
        .map_err(|e| e.to_string())
}

/// Set notification settings for file conflicts
#[tauri::command]
pub async fn set_notify_file_conflict(enabled: bool) -> CommandResult<()> {
    ConfigManager::get()
        .set_notify_file_conflict(enabled)
        .map_err(|e| e.to_string())
}

/// Set fast popup launch setting
#[tauri::command]
pub async fn set_fast_popup_launch(enabled: bool) -> CommandResult<()> {
    ConfigManager::get()
        .set_fast_popup_launch(enabled)
        .map_err(|e| e.to_string())
}

/// Get all general settings
#[tauri::command]
pub async fn get_general_settings() -> CommandResult<GeneralSettings> {
    let config = ConfigManager::get().get_config();
    Ok(GeneralSettings {
        notify_credential_expired: config.notify_credential_expired,
        notify_file_conflict: config.notify_file_conflict,
        fast_popup_launch: config.fast_popup_launch,
        log_to_file: config.log_to_file,
        log_level: config.log_level.as_str().to_string(),
        log_max_files: config.log_max_files,
        sync_delay_seconds: config.sync_delay_seconds,
        hide_tray_icon: config.hide_tray_icon,
        log_dir: ConfigManager::get_log_dir().display().to_string(),
        language: config.language,
    })
}

#[derive(serde::Serialize)]
pub struct GeneralSettings {
    pub notify_credential_expired: bool,
    pub notify_file_conflict: bool,
    pub fast_popup_launch: bool,
    pub log_to_file: bool,
    pub log_level: String,
    pub log_max_files: usize,
    pub sync_delay_seconds: u64,
    pub hide_tray_icon: bool,
    pub log_dir: String,
    pub language: Option<String>,
}

/// Set log to file setting
#[tauri::command]
pub async fn set_log_to_file(enabled: bool) -> CommandResult<()> {
    ConfigManager::get()
        .set_log_to_file(enabled)
        .map_err(|e| e.to_string())
}

/// Set log level setting
#[tauri::command]
pub async fn set_log_level(level: String) -> CommandResult<()> {
    let log_level = LogLevel::from_str(&level);

    // Update config (requires restart to take effect)
    ConfigManager::get()
        .set_log_level(log_level)
        .map_err(|e| e.to_string())
}

/// Set max log files setting
#[tauri::command]
pub async fn set_log_max_files(max_files: usize) -> CommandResult<()> {
    ConfigManager::get()
        .set_log_max_files(max_files)
        .map_err(|e| e.to_string())
}

/// Set sync delay in seconds
#[tauri::command]
pub async fn set_sync_delay_seconds(seconds: u64) -> CommandResult<()> {
    ConfigManager::get()
        .set_sync_delay_seconds(seconds)
        .map_err(|e| e.to_string())
}

/// Set whether the system tray icon is hidden and apply it live
#[tauri::command]
pub async fn set_hide_tray_icon(app: AppHandle, hide: bool) -> CommandResult<()> {
    ConfigManager::get()
        .set_hide_tray_icon(hide)
        .map_err(|e| e.to_string())?;
    if let Some(tray) = app.try_state::<TrayIcon>() {
        tray.set_visible(!hide).map_err(|e| e.to_string())?;
    }
    Ok(())
}

/// Set language setting and update rust_i18n locale
#[tauri::command]
pub async fn set_language(app: AppHandle, language: Option<String>) -> CommandResult<()> {
    // Update the config
    ConfigManager::get()
        .set_language(language.clone())
        .map_err(|e| e.to_string())?;

    // Update rust_i18n locale to the effective locale (config setting or
    // normalized system locale).
    rust_i18n::set_locale(&crate::get_effective_locale());

    // Rebuild the tray context menu with the new locale so it doesn't show
    // raw i18n keys.
    if let Err(e) = crate::rebuild_tray_menu(&app) {
        tracing::warn!(target: "main", error = %e, "Failed to rebuild tray menu after language change");
    }

    // Close main window to force reload with new language
    // Check if window already exists
    if let Some(window) = app.get_webview_window("main_popup") {
        let _ = window.close();
        let _ = window.destroy();
    }

    Ok(())
}

/// Open the log folder in file explorer
#[tauri::command]
pub async fn open_log_folder() -> CommandResult<()> {
    let log_dir = ConfigManager::get_log_dir();

    // Create the directory if it doesn't exist
    if !log_dir.exists() {
        std::fs::create_dir_all(&log_dir).map_err(|e| e.to_string())?;
    }

    showfile::show_path_in_file_manager(format!("{}\\", log_dir.display()));
    Ok(())
}

/// Metadata about an available app update, returned to the frontend.
#[derive(serde::Serialize)]
pub struct UpdateInfo {
    pub version: String,
    pub current_version: String,
    pub notes: Option<String>,
    pub date: Option<String>,
}

/// Progress payload emitted on `update-progress` while an update downloads.
#[derive(serde::Serialize, Clone)]
struct UpdateProgress {
    downloaded: u64,
    total: Option<u64>,
}

/// Check the configured update endpoint (GitHub releases `latest.json`) for a
/// newer desktop build. Returns `None` when already up to date or when no
/// signed updater artifact exists for this platform.
#[tauri::command]
pub async fn check_update(app: AppHandle) -> CommandResult<Option<UpdateInfo>> {
    use tauri_plugin_updater::UpdaterExt;

    let update = app
        .updater()
        .map_err(|e| e.to_string())?
        .check()
        .await
        .map_err(|e| e.to_string())?;

    Ok(update.map(|u| UpdateInfo {
        version: u.version.clone(),
        current_version: u.current_version.clone(),
        notes: u.body.clone(),
        date: u.date.map(|d| d.to_string()),
    }))
}

/// Download the pending update, emit `update-progress` events, and install it.
/// The frontend should call `restart_app` afterwards.
#[tauri::command]
pub async fn install_update(app: AppHandle) -> CommandResult<()> {
    use tauri::Emitter;
    use tauri_plugin_updater::UpdaterExt;

    let update = app
        .updater()
        .map_err(|e| e.to_string())?
        .check()
        .await
        .map_err(|e| e.to_string())?
        .ok_or_else(|| "No update available".to_string())?;

    let app2 = app.clone();
    let app3 = app.clone();
    update
        .download_and_install(
            move |chunk_len, content_len| {
                let _ = app2.emit(
                    "update-progress",
                    UpdateProgress {
                        downloaded: chunk_len as u64,
                        total: content_len,
                    },
                );
            },
            move || {
                let _ = app3.emit("update-finished", ());
            },
        )
        .await
        .map_err(|e| e.to_string())?;

    Ok(())
}

/// Restart the application (used after a successful update install).
#[tauri::command]
pub fn restart_app(app: AppHandle) {
    app.restart();
}

/// Show or create the in-app update window. The window itself runs the update
/// check so it works both for automatic prompts and manual "Check for
/// updates" from the About page.
pub fn show_update_window_impl(app: &AppHandle, auto_download: bool) {
    // One update window at a time — refocus instead of stacking dialogs.
    if let Some(window) = app.get_webview_window("update") {
        let _ = window.unminimize();
        let _ = window.show();
        let _ = window.set_focus();
        return;
    }

    let url_path = get_url_with_lang(if auto_download {
        "index.html#/update?auto=1"
    } else {
        "index.html#/update"
    });

    #[cfg(windows)]
    let effects = WindowEffectsConfig {
        effects: vec![WindowEffect::Mica, WindowEffect::Acrylic],
        state: None,
        radius: None,
        color: None,
    };

    let builder = WebviewWindowBuilder::new(app, "update", WebviewUrl::App(url_path.into()))
        .title("Software Update")
        .inner_size(460.0, 380.0)
        .resizable(false)
        .visible(false)
        .decorations(false)
        .minimizable(false);

    #[cfg(not(windows))]
    let builder = builder.background_color(Color(255, 255, 255, 255));

    #[cfg(windows)]
    let builder = builder.transparent(true);

    #[cfg(target_os = "macos")]
    let builder = builder
        .title_bar_style(TitleBarStyle::Overlay)
        .hidden_title(true);

    let Some(builder) = apply_default_window_icon(builder, app, "update") else {
        return;
    };

    match builder.build() {
        Ok(window) => {
            #[cfg(target_os = "macos")]
            update_dock_on_window_close(&window);

            #[cfg(windows)]
            {
                let _ = window.set_effects(effects);
            }

            move_window_safely(&window, Position::Center, "update");
            #[cfg(windows)]
            let _ = window.create_overlay_titlebar();
            let _ = window.show();
            let _ = window.set_focus();
            #[cfg(target_os = "macos")]
            crate::update_dock_visibility(app);
        }
        Err(e) => {
            tracing::error!(target: "main", error = %e, "Failed to create update window");
        }
    }
}

/// Open the in-app update window (Settings → About → "Check for updates").
#[tauri::command]
pub async fn show_update_window(app: AppHandle) -> CommandResult<()> {
    show_update_window_impl(&app, false);
    Ok(())
}
