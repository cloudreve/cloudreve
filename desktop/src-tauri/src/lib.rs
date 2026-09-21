use anyhow::Context;
use cloudreve_sync::{
    shellext::shell_service::ServiceHandle, ConfigManager, DriveManager, EventBroadcaster,
    LogConfig, LogGuard,
};
use std::sync::{Arc, Mutex};
use tauri::{
    async_runtime::spawn,
    menu::{Menu, MenuItem},
    tray::{MouseButton, MouseButtonState, TrayIcon, TrayIconBuilder, TrayIconEvent},
    AppHandle, Emitter, Listener, Manager, RunEvent,
};
#[cfg(not(target_os = "macos"))]
use tauri_plugin_deep_link::DeepLinkExt;
use tokio::sync::OnceCell;

use crate::commands::{show_add_drive_window_impl, show_main_window, show_settings_window_impl};
mod commands;
mod event_handler;

#[macro_use]
extern crate rust_i18n;

i18n!("../locales", fallback = "en-US");

/// Normalize a raw locale string to one supported by the app.
///
/// macOS may report locales like "zh-Hans-CN" which don't directly match our
/// translation files ("zh-CN"). This strips script tags and falls back to the
/// language code before giving up and returning "en-US".
fn normalize_locale(locale: &str) -> String {
    let locale = locale.trim();
    if locale.is_empty() {
        return "en-US".to_string();
    }

    let available = available_locales!();
    let lower = locale.to_lowercase();

    // Exact match (case-insensitive)
    for l in &available {
        if l.to_lowercase() == lower {
            return l.to_string();
        }
    }

    // Try to strip script subtag: zh-Hans-CN -> zh-CN
    let parts: Vec<&str> = locale.split('-').collect();
    if parts.len() >= 3 {
        let without_script = format!("{}-{}", parts[0], parts.last().unwrap());
        let lower_without = without_script.to_lowercase();
        for l in &available {
            if l.to_lowercase() == lower_without {
                return l.to_string();
            }
        }
    }

    // Try language code only: en-US -> en
    if !parts.is_empty() {
        let lang = parts[0].to_lowercase();
        for l in &available {
            if l.to_lowercase() == lang {
                return l.to_string();
            }
        }
    }

    "en-US".to_string()
}

/// Initialize i18n based on config setting or system locale
fn init_i18n() {
    use rust_i18n::set_locale;
    use sys_locale::get_locale;

    // Try to get language from config, fallback to system locale
    let locale = ConfigManager::try_get()
        .and_then(|cm| cm.language())
        .unwrap_or_else(|| get_locale().unwrap_or_else(|| String::from("en-US")));
    set_locale(normalize_locale(&locale).as_str());
}

/// Get the current effective locale (from config or system)
pub fn get_effective_locale() -> String {
    use sys_locale::get_locale;

    let locale = ConfigManager::try_get()
        .and_then(|cm| cm.language())
        .unwrap_or_else(|| get_locale().unwrap_or_else(|| String::from("en-US")));
    normalize_locale(&locale)
}

/// Application state containing the drive manager and event broadcaster
pub struct AppState {
    pub drive_manager: Arc<DriveManager>,
    pub event_broadcaster: Arc<EventBroadcaster>,
    // Keep the log guard alive for the entire application lifetime
    #[allow(dead_code)]
    log_guard: LogGuard,
    // Keep the shell service handle alive for the entire application lifetime
    #[allow(dead_code)]
    shell_service: Mutex<ServiceHandle>,
}

/// Global cell to store the app state once initialization is complete
static APP_STATE: OnceCell<AppState> = OnceCell::const_new();

/// Initialize the sync service (DriveManager, shell services, etc.)
async fn init_sync_service(app: AppHandle) -> anyhow::Result<()> {
    // Initialize app root (Windows Package detection)
    cloudreve_sync::init_app_root();

    // Initialize logging system with config from ConfigManager
    let log_guard = cloudreve_sync::logging::init_logging(LogConfig::from_config_manager())
        .context("Failed to initialize logging system")?;

    tracing::info!(target: "main", "Starting Cloudreve Sync Service (Tauri)...");

    // Initialize EventBroadcaster
    let event_broadcaster = Arc::new(EventBroadcaster::new(100));
    tracing::info!(target: "main", "Event broadcasting system initialized");

    // Spawn event bridge to forward events to tarui
    spawn_event_bridge(app.clone(), &event_broadcaster);

    // Initialize DriveManager
    tracing::info!(target: "main", "Initializing DriveManager...");
    let drive_manager = Arc::new(
        DriveManager::new(event_broadcaster.clone()).context("Failed to create DriveManager")?,
    );

    // Spawn command processor for DriveManager
    drive_manager.spawn_command_processor().await;
    tracing::info!(target: "main", "DriveManager command processor started");

    // Load drive configurations from disk
    drive_manager
        .load()
        .await
        .context("Failed to load drive configurations")?;

    // Initialize and start the shell services (context menu handler) in a separate thread
    let mut shell_service =
        cloudreve_sync::shellext::shell_service::init_and_start_service_task(drive_manager.clone());

    // Wait for shell services to initialize
    if let Err(e) = shell_service.wait_for_init() {
        tracing::error!(target: "main", "Warning: Failed to initialize shell services: {:?}", e);
        tracing::info!(target: "main", "Continuing without context menu handler...");
    } else {
        tracing::info!(target: "main", "Shell services initialized successfully!");
    }

    // Broadcast initial connection status
    event_broadcaster.connection_status_changed(true);
    // Capture the no-drive state now, but do not emit it until APP_STATE is
    // installed below. The UI event handler opens the add-drive window and that
    // command reads AppStateHandle; emitting before `app.manage(AppStateHandle)`
    // can make the first startup report "no drive" even when later commands
    // cannot access the initialized manager.
    let has_no_drives = drive_manager.is_empty().await;

    // Store the state in the global cell
    let state = AppState {
        drive_manager,
        event_broadcaster: event_broadcaster.clone(),
        log_guard,
        shell_service: Mutex::new(shell_service),
    };

    APP_STATE
        .set(state)
        .map_err(|_| anyhow::anyhow!("App state already initialized"))?;

    // Store in Tauri's managed state as well for commands
    app.manage(AppStateHandle);

    if has_no_drives {
        event_broadcaster.no_drive();
    }

    tracing::info!(target: "main", "Tauri application setup complete");

    Ok(())
}

/// Marker struct for Tauri state that provides access to APP_STATE
pub struct AppStateHandle;

/// Update macOS Dock visibility based on whether any webview window is visible.
#[cfg(target_os = "macos")]
pub fn update_dock_visibility(app: &AppHandle) {
    let has_visible = app
        .webview_windows()
        .values()
        .any(|w| w.is_visible().unwrap_or(false));

    if let Err(err) = app.set_dock_visibility(has_visible) {
        tracing::warn!(
            target: "main",
            error = %err,
            visible = has_visible,
            "Failed to update Dock visibility"
        );
    }
}

/// Schedule multiple delayed checks of the Dock visibility.
///
/// `webview_windows()`/`is_visible()` can lag behind the actual window state on
/// macOS, especially after a window is hidden or closed by clicking outside the
/// frame. Calling `update_dock_visibility` immediately and then several more
/// times gives AppKit enough time to reflect the new visibility state so the
/// Dock icon reliably hides when no window is visible.
#[cfg(target_os = "macos")]
pub fn schedule_update_dock_visibility(app: &AppHandle) {
    let app = app.clone();
    tauri::async_runtime::spawn(async move {
        for delay_ms in [0, 50, 150, 350, 750] {
            if delay_ms > 0 {
                tokio::time::sleep(std::time::Duration::from_millis(delay_ms)).await;
            }
            update_dock_visibility(&app);
        }
    });
}

impl AppStateHandle {
    pub fn get(&self) -> Option<&'static AppState> {
        APP_STATE.get()
    }
}

/// Spawn a task that bridges EventBroadcaster to Tauri events
fn spawn_event_bridge(app_handle: AppHandle, event_broadcaster: &EventBroadcaster) {
    let mut receiver = event_broadcaster.subscribe();

    spawn(async move {
        tracing::info!(target: "events", "Event bridge started");

        loop {
            match receiver.recv().await {
                Ok(event) => {
                    event_handler::handle_event(&app_handle, &event);
                    event_handler::emit_event(&app_handle, &event);
                }
                Err(tokio::sync::broadcast::error::RecvError::Lagged(n)) => {
                    tracing::warn!(target: "events", skipped = n, "Event receiver lagged, some events were skipped");
                }
                Err(tokio::sync::broadcast::error::RecvError::Closed) => {
                    tracing::info!(target: "events", "Event broadcaster closed, stopping bridge");
                    break;
                }
            }
        }
    });
}

/// Perform graceful shutdown
async fn shutdown() {
    tracing::info!(target: "main", "Initiating shutdown...");

    if let Some(state) = APP_STATE.get() {
        // Broadcast disconnection event
        state.event_broadcaster.connection_status_changed(false);

        // Shutdown drive manager
        tracing::info!(target: "main", "Shutting down drive manager...");
        state.drive_manager.shutdown().await;

        // Persist drive state
        tracing::info!(target: "main", "Persisting drive configurations...");
        if let Err(e) = state.drive_manager.persist().await {
            tracing::error!(target: "main", error = %e, "Failed to persist drive configurations");
        } else {
            tracing::info!(target: "main", "Drive configurations saved successfully");
        }
    }

    tracing::info!(target: "main", "Shutdown complete");
}

/// Resolve a tray menu label, falling back to the English text when rust_i18n
/// returns the raw key. This guards against missing translations or an
/// unloaded locale, which has been observed to show raw i18n keys on macOS.
macro_rules! tray_label {
    ($key:literal, $default:literal) => {{
        let translated = t!($key).to_string();
        if translated == $key || translated.is_empty() {
            $default.to_string()
        } else {
            translated
        }
    }};
}

/// Return the tray menu item IDs and their localized titles.
fn tray_menu_entries() -> [(&'static str, String); 4] {
    [
        ("show", tray_label!("show", "Show")),
        ("add_drive", tray_label!("addNewDrive", "Add new drive")),
        ("settings", tray_label!("settings", "Settings")),
        ("quit", tray_label!("quit", "Quit")),
    ]
}

/// Build the localized tray context menu with the current locale.
fn build_tray_menu<R: tauri::Runtime>(manager: &impl Manager<R>) -> anyhow::Result<Menu<R>> {
    let entries = tray_menu_entries();
    let menu_items: Vec<MenuItem<R>> = entries
        .iter()
        .map(|(id, title)| MenuItem::with_id(manager, *id, title.clone(), true, None::<&str>))
        .collect::<Result<_, _>>()?;
    let item_refs: Vec<&dyn tauri::menu::IsMenuItem<R>> = menu_items
        .iter()
        .map(|item| item as &dyn tauri::menu::IsMenuItem<R>)
        .collect();
    let menu = Menu::with_items(manager, &item_refs)?;
    Ok(menu)
}

/// Rebuild the tray context menu using the current locale.
pub fn rebuild_tray_menu<R: tauri::Runtime>(app: &AppHandle<R>) -> anyhow::Result<()> {
    let menu = build_tray_menu(app)?;
    let tray = app.state::<TrayIcon<R>>();
    tray.set_menu(Some(menu))?;
    Ok(())
}

/// Setup the system tray icon
fn setup_tray(app: &tauri::App) -> anyhow::Result<()> {
    let menu = build_tray_menu(app)?;

    // Build tray icon
    let tray = TrayIconBuilder::new()
        .icon(app.default_window_icon().unwrap().clone())
        .menu(&menu)
        .show_menu_on_left_click(false)
        .on_menu_event(|app, event| match event.id.as_ref() {
            "show" => {
                show_main_window(app);
            }
            "add_drive" => {
                show_add_drive_window_impl(app);
            }
            "settings" => {
                show_settings_window_impl(app);
            }
            "quit" => {
                app.exit(0);
            }
            _ => {}
        })
        .on_tray_icon_event(|tray, event| {
            tauri_plugin_positioner::on_tray_event(tray.app_handle(), &event);
            if let TrayIconEvent::Click {
                button: MouseButton::Left,
                button_state: MouseButtonState::Up,
                ..
            } = event
            {
                let app = tray.app_handle();
                show_main_window(app);
            }
        })
        .build(app)?;

    // Honour the hide-tray-icon setting (TrayIconBuilder has no visibility
    // option, so it is applied right after construction).
    if ConfigManager::get().hide_tray_icon() {
        let _ = tray.set_visible(false);
    }

    // Keep the tray icon in Tauri's managed state so we can update its menu
    // when the language changes.
    app.manage(tray);

    Ok(())
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    // Initialize config manager first so i18n can read language setting
    if let Err(e) = ConfigManager::init() {
        eprintln!("Failed to initialize config manager: {}", e);
    }

    // Initialize i18n (uses config language setting or falls back to system locale)
    init_i18n();

    #[allow(unused_mut)]
    let mut builder = tauri::Builder::default()
        .plugin(tauri_plugin_single_instance::init(|app, argv, _cwd| {
            tracing::info!("a new app instance was opened with {argv:?} and the deep link event was already triggered");
            if argv.len() > 1 {
                let _ = app.emit("deeplink", argv[1].clone());
                show_add_drive_window_impl(app);
            }
            // when defining deep link schemes at runtime, you must also check `argv` here
        }))
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_http::init());

    #[cfg(windows)]
    {
        builder = builder.plugin(tauri_plugin_frame::init());
    }

    builder
        .plugin(tauri_plugin_deep_link::init())
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_os::init())
        .plugin(tauri_plugin_prevent_default::debug())
        .setup(|app| {
            #[cfg(desktop)]
            let _ = app.handle().plugin(tauri_plugin_positioner::init());

            // Setup system tray
            setup_tray(app)?;

            #[cfg(not(target_os = "macos"))]
            app.deep_link().register("cloudreve")?;

            // Listen for deep-link events (macOS and Linux)
            let app_handle = app.handle().clone();
            app.listen("deep-link://new-url", move |event: tauri::Event| {
                if let Ok(urls) = serde_json::from_str::<Vec<String>>(event.payload()) {
                    if let Some(url) = urls.first() {
                        tracing::info!(target: "main", "Received deep-link URL: {}", url);
                        let _ = app_handle.emit("deeplink", url.clone());
                        show_add_drive_window_impl(&app_handle);
                    }
                }
            });

            // Spawn async setup task - this runs in the background
            // while the app continues to start
            let app_handle = app.handle().clone();
            spawn(async move {
                if let Err(e) = init_sync_service(app_handle).await {
                    tracing::error!(target: "main", error = %e, "Failed to initialize sync service");
                }
            });

            // close default main window
            if let Some(window) = app.get_webview_window("main") {
                let _ = window.destroy();
            }

            // Keep the Dock icon hidden on macOS while no window is visible.
            #[cfg(target_os = "macos")]
            update_dock_visibility(app.handle());

            Ok(())
        })
        .invoke_handler(tauri::generate_handler![
            commands::is_dir_empty,
            commands::list_drives,
            commands::add_drive,
            commands::remove_drive,
            commands::reconnect_drive,
            commands::get_ignore_patterns,
            commands::set_ignore_patterns,
            commands::get_sync_status,
            commands::get_status_summary,
            commands::resolve_conflict,
            commands::resolve_all_conflicts,
            commands::get_drives_info,
            commands::get_file_icon,
            commands::show_file_in_explorer,
            commands::create_share,
            commands::show_add_drive_window,
            commands::show_reauthorize_window,
            commands::show_settings_window,
            commands::get_auto_start_enabled,
            commands::set_auto_start,
            commands::set_notify_credential_expired,
            commands::set_notify_file_conflict,
            commands::set_fast_popup_launch,
            commands::get_general_settings,
            commands::set_log_to_file,
            commands::set_log_level,
            commands::set_log_max_files,
            commands::set_sync_delay_seconds,
            commands::set_hide_tray_icon,
            commands::set_language,
            commands::open_log_folder,
        ])
        .build(tauri::generate_context!())
        .expect("error while building tauri application")
        .run(|_app_handle, event| {
            match event {
                RunEvent::ExitRequested { api,code,.. } => {
                     if code.is_none() {
                        api.prevent_exit();
                    } else {
                        tracing::info!("exit code: {:?}", code);
                    }
                    tracing::info!(target: "main", "Exit requested");
                }
                RunEvent::Exit => {
                    // Perform shutdown when the app is actually exiting
                    tauri::async_runtime::block_on(shutdown());
                }
                #[cfg(target_os = "macos")]
                RunEvent::WindowEvent {
                    event:
                        tauri::WindowEvent::CloseRequested { .. }
                        | tauri::WindowEvent::Destroyed
                        | tauri::WindowEvent::Focused(false),
                    ..
                } => {
                    // Re-evaluate Dock visibility several times after a window loses
                    // focus or is closed/destroyed. `webview_windows()`/`is_visible()`
                    // can lag, so retrying makes sure the Dock icon hides when no
                    // window remains.
                    crate::schedule_update_dock_visibility(_app_handle);
                }
                _ => {}
            }
        });
}

#[cfg(test)]
mod tests {
    use super::*;
    use serial_test::serial;

    #[test]
    #[serial(i18n_locale)]
    fn tray_menu_entries_are_localized() {
        rust_i18n::set_locale("en-US");
        let entries = tray_menu_entries();

        let expected = [
            ("show", "Show"),
            ("add_drive", "Add new drive"),
            ("settings", "Settings"),
            ("quit", "Quit"),
        ];

        assert_eq!(entries.len(), expected.len());
        for ((id, title), (expected_id, expected_title)) in entries.iter().zip(expected.iter()) {
            assert_eq!(*id, *expected_id);
            assert_eq!(title, *expected_title);
        }
    }

    #[test]
    fn normalize_locale_handles_script_subtag() {
        assert_eq!(normalize_locale("zh-Hans-CN"), "zh-CN");
        assert_eq!(normalize_locale("zh-Hant-TW"), "zh-TW");
    }

    #[test]
    fn normalize_locale_is_case_insensitive() {
        assert_eq!(normalize_locale("EN-us"), "en-US");
        assert_eq!(normalize_locale("ZH-cn"), "zh-CN");
    }

    #[test]
    fn normalize_locale_falls_back_to_default() {
        assert_eq!(normalize_locale("xx-YY"), "en-US");
        assert_eq!(normalize_locale(""), "en-US");
    }

    #[test]
    #[serial(i18n_locale)]
    fn tray_menu_entries_are_not_raw_i18n_keys() {
        rust_i18n::set_locale("en-US");
        let entries = tray_menu_entries();

        let raw_keys = ["show", "addNewDrive", "settings", "quit"];
        for ((id, title), raw_key) in entries.iter().zip(raw_keys.iter()) {
            assert_ne!(
                title, *raw_key,
                "menu title for {} should be localized, not raw key",
                id
            );
        }
    }

    #[test]
    #[serial(i18n_locale)]
    fn tray_menu_entries_update_with_locale() {
        rust_i18n::set_locale("zh-CN");
        let entries = tray_menu_entries();
        assert_eq!(entries[0].1, "显示");
        assert_eq!(entries[1].1, "添加新云盘");
        assert_eq!(entries[2].1, "设置");
        assert_eq!(entries[3].1, "退出");
    }

    #[test]
    #[serial(i18n_locale)]
    fn tray_menu_entries_fallback_to_default_locale() {
        // When the locale is unknown, rust_i18n should fall back to en-US
        // instead of returning raw i18n keys.
        rust_i18n::set_locale("xx-YY");
        let entries = tray_menu_entries();
        assert_eq!(entries[0].1, "Show");
        assert_eq!(entries[1].1, "Add new drive");
        assert_eq!(entries[2].1, "Settings");
        assert_eq!(entries[3].1, "Quit");
    }
}
