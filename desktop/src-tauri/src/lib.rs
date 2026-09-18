use anyhow::Context;
use cloudreve_sync::{ConfigManager, DriveManager, EventBroadcaster, LogConfig, LogGuard, shellext::shell_service::ServiceHandle};
use std::sync::{Arc, Mutex};
use tauri::{
    async_runtime::spawn,
    menu::{Menu, MenuItem},
    tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent},
    AppHandle, Emitter, Manager, RunEvent,
};
use tauri_plugin_deep_link::DeepLinkExt;
use tokio::sync::OnceCell;

use crate::commands::{show_add_drive_window_impl, show_main_window, show_settings_window_impl};
mod commands;
mod event_handler;

#[macro_use]
extern crate rust_i18n;

i18n!("../locales");

/// Initialize i18n based on config setting or system locale
fn init_i18n() {
    use rust_i18n::set_locale;
    use sys_locale::get_locale;

    // Try to get language from config, fallback to system locale
    let locale = ConfigManager::try_get()
        .and_then(|cm| cm.language())
        .unwrap_or_else(|| get_locale().unwrap_or_else(|| String::from("en-US")));
    set_locale(locale.as_str());
}

/// Get the current effective locale (from config or system)
pub fn get_effective_locale() -> String {
    use sys_locale::get_locale;

    ConfigManager::try_get()
        .and_then(|cm| cm.language())
        .unwrap_or_else(|| get_locale().unwrap_or_else(|| String::from("en-US")))
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

    tracing::info!(target: "main", "Tauri application setup complete");

    Ok(())
}

/// Marker struct for Tauri state that provides access to APP_STATE
pub struct AppStateHandle;

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

/// Setup the system tray icon
fn setup_tray(app: &tauri::App) -> anyhow::Result<()> {
    // Create menu items
    let show_i = MenuItem::with_id(app, "show", t!("show").as_ref(), true, None::<&str>)?;
    let add_drive_i = MenuItem::with_id(
        app,
        "add_drive",
        t!("addNewDrive").as_ref(),
        true,
        None::<&str>,
    )?;
    let settings_i = MenuItem::with_id(
        app,
        "settings",
        t!("settings").as_ref(),
        true,
        None::<&str>,
    )?;
    let quit_i = MenuItem::with_id(app, "quit", t!("quit").as_ref(), true, None::<&str>)?;
    let menu = Menu::with_items(app, &[&show_i, &add_drive_i, &settings_i, &quit_i])?;

    // Build tray icon
    TrayIconBuilder::new()
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

    tauri::Builder::default()
        .plugin(tauri_plugin_single_instance::init(|app, argv, _cwd| {
            tracing::info!("a new app instance was opened with {argv:?} and the deep link event was already triggered");
            if argv.len() > 1 {
                let _ = app.emit("deeplink", argv[1].clone());
                show_add_drive_window_impl(app);
            }
            // when defining deep link schemes at runtime, you must also check `argv` here
        }))
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_http::init())
        .plugin(tauri_plugin_frame::init())
        .plugin(tauri_plugin_deep_link::init())
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_os::init())
        .plugin(tauri_plugin_prevent_default::debug())
        .setup(|app| {
            #[cfg(desktop)]
            let _ = app.handle().plugin(tauri_plugin_positioner::init());

            // Setup system tray
            setup_tray(app)?;

            #[cfg(desktop)]
            app.deep_link().register("cloudreve")?;

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

            Ok(())
        })
        .invoke_handler(tauri::generate_handler![
            commands::is_dir_empty,
            commands::list_drives,
            commands::add_drive,
            commands::remove_drive,
            commands::get_ignore_patterns,
            commands::set_ignore_patterns,
            commands::get_sync_status,
            commands::get_status_summary,
            commands::get_drives_info,
            commands::get_file_icon,
            commands::show_file_in_explorer,
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
                _ => {}
            }
        });
}
