use crate::AppStateHandle;
use base64::{engine::general_purpose::STANDARD as BASE64, Engine as _};
use chrono::{Duration, Utc};
use cloudreve_sync::{
    config::LogLevel, ConfigManager, Credentials, DriveConfig, DriveInfo, StatusSummary,
};
#[cfg(target_os = "macos")]
use tauri::TitleBarStyle;
use tauri::{
    utils::{config::WindowEffectsConfig, WindowEffect},
    webview::WebviewWindowBuilder,
    AppHandle, Manager, State, WebviewUrl,
};
use tauri_plugin_frame::WebviewWindowExt;
use tauri_plugin_positioner::{Position, WindowExt};
use uuid::Uuid;
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
                config.site_url,
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

    let drive_config = DriveConfig {
        id: drive_id,
        name: config.drive_name,
        instance_url: config.site_url,
        remote_path: config.remote_path,
        credentials,
        sync_path: config.local_path.into(),
        icon_path: None,
        raw_icon_path: None,
        enabled: true,
        user_id: config.user_id,
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

/// Get file icon for a given path
/// Returns base64 encoded RGBA pixel data with dimensions
#[tauri::command]
pub async fn get_file_icon(path: String, size: Option<u16>) -> CommandResult<FileIconResponse> {
    let icon_size = size.unwrap_or(32);

    // Run the blocking icon retrieval in a separate thread
    let result =
        tokio::task::spawn_blocking(move || file_icon_provider::get_file_icon(&path, icon_size))
            .await
            .map_err(|e| format!("Task join error: {}", e))?
            .map_err(|e| format!("Failed to get file icon: {:?}", e))?;

    Ok(FileIconResponse {
        data: BASE64.encode(&result.pixels),
        width: result.width,
        height: result.height,
    })
}

/// Show or create the main window (positioned at tray center)
pub fn show_main_window(app: &AppHandle) {
    show_main_window_at_position(app, Position::TrayCenter);
}

/// Show or create the main window (positioned at bottom right)
pub fn show_main_window_center(app: &AppHandle) {
    show_main_window_at_position(app, Position::Center);
}

/// Internal function to show or create the main window at a specific position
fn show_main_window_at_position(app: &AppHandle, position: Position) {
    // Check if window already exists
    if let Some(window) = app.get_webview_window("main_popup") {
        let _ = window.move_window(position);
        let _ = window.show();
        let _ = window.unminimize();
        let _ = window.set_focus();
        return;
    }

    // Create new main window
    match WebviewWindowBuilder::new(
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
    .minimizable(false)
    .build()
    {
        Ok(window) => {
            // Set up close request handler for fast popup launch
            let window_clone = window.clone();
            window.on_window_event(move |event| {
                if let tauri::WindowEvent::CloseRequested { api, .. } = event {
                    // Check if fast popup launch is enabled
                    if ConfigManager::get().fast_popup_launch() {
                        // Prevent default close behavior and hide window instead
                        api.prevent_close();
                        let _ = window_clone.hide();
                    }
                }
            });

            let _ = window.move_window(position);
            let _ = window.show();
            let _ = window.set_focus();
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
    show_drive_window_internal(app, "Add Drive", &get_url_with_lang("index.html/#/add-drive"));
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
        return;
    }

    // Create new window with mica effect
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
        .transparent(true)
        .effects(effects)
        .decorations(false)
        .minimizable(false);

    // Platform-specific: title_bar_style and hidden_title are macOS-only
    #[cfg(target_os = "macos")]
    let builder = builder
        .title_bar_style(TitleBarStyle::Overlay)
        .hidden_title(true);

    match builder.build() {
        Ok(window) => {
            let _ = window.move_window(Position::Center);
            let _ = window.create_overlay_titlebar();
            let _ = window.show();
            let _ = window.set_focus();
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
        return;
    }

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

    // Platform-specific: title_bar_style and hidden_title are macOS-only
    #[cfg(target_os = "macos")]
    let builder = builder
        .title_bar_style(TitleBarStyle::Overlay)
        .hidden_title(true);

    match builder.build() {
        Ok(window) => {
            let _ = window.move_window(Position::Center);
            let _ = window.create_overlay_titlebar();
            let _ = window.show();
            let _ = window.set_focus();
        }
        Err(e) => {
            tracing::error!(target: "main", error = %e, "Failed to create settings window");
        }
    }
}

/// The TaskId defined in AppxManifest.xml for the startup task
const STARTUP_TASK_ID: &str = "cloudreve";

/// Get whether auto-start is enabled using Windows StartupTask API
#[tauri::command]
pub async fn get_auto_start_enabled() -> CommandResult<bool> {
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

/// Set auto-start configuration using Windows StartupTask API
#[tauri::command]
pub async fn set_auto_start(enabled: bool) -> CommandResult<bool> {
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

/// Set language setting and update rust_i18n locale
#[tauri::command]
pub async fn set_language(app: AppHandle, language: Option<String>) -> CommandResult<()> {
    // Update the config
    ConfigManager::get()
        .set_language(language.clone())
        .map_err(|e| e.to_string())?;

    // Update rust_i18n locale
    let locale = language.unwrap_or_else(|| {
        sys_locale::get_locale().unwrap_or_else(|| String::from("en-US"))
    });
    rust_i18n::set_locale(&locale);

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
