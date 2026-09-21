#[cfg(windows)]
pub mod cfapi;
#[cfg(not(windows))]
#[path = "cfapi/non_windows.rs"]
pub mod cfapi;
pub mod config;
pub mod drive;
pub mod events;
pub mod inventory;
pub mod logging;
#[cfg(windows)]
pub mod shellext;
#[cfg(not(windows))]
#[path = "shellext/non_windows.rs"]
pub mod shellext;
pub mod tasks;
pub mod uploader;
pub mod utils;

// Re-export commonly used types
pub use cloudreve_api::models::share::ShareCreateService;
pub use cloudreve_api::normalize_site_url;
pub use config::{AppConfig, ConfigManager};
pub use drive::manager::{
    DriveInfo, DriveInfoStatus, DriveManager, StatusSummary, TaskWithProgress,
};
pub use drive::mounts::{Credentials, DriveConfig};
pub use events::{Event, EventBroadcaster};
pub use logging::{LogConfig, LogGuard};

/// User agent string for HTTP requests
pub const USER_AGENT: &str = concat!("cloudreve-desktop/", env!("CARGO_PKG_VERSION"));

#[macro_use]
extern crate rust_i18n;

i18n!("../../locales", fallback = "en-US");

/// Initialize the application root path (Windows Package detection)
pub fn init_app_root() {
    utils::app::init_app_root();
}
