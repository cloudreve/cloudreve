#[cfg(windows)]
pub mod callback;
pub mod commands;
pub mod event_blocker;
pub mod ignore;
pub mod manager;
pub mod mounts;
#[cfg(windows)]
pub mod placeholder;
#[cfg(not(windows))]
#[path = "placeholder_non_windows.rs"]
pub mod placeholder;
pub mod remote_events;
pub mod sync;
pub mod utils;
