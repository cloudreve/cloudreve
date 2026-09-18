use anyhow::{Context, Result};
use std::path::PathBuf;
use std::sync::OnceLock;
use tracing_subscriber::{
    EnvFilter,
    fmt::{self, format::FmtSpan},
    layer::SubscriberExt,
    util::SubscriberInitExt,
};

use crate::config::{ConfigManager, LogLevel};

/// Configuration for the logging system
pub struct LogConfig {
    /// Directory where log files will be stored
    pub log_dir: PathBuf,
    /// Prefix for log file names
    pub file_prefix: String,
    /// Maximum number of log files to keep (rotation)
    pub max_files: usize,
    /// Whether to write logs to file
    pub log_to_file: bool,
    /// Log level filter string
    pub log_level: String,
}

impl Default for LogConfig {
    fn default() -> Self {
        let log_dir = dirs::home_dir()
            .unwrap_or_else(|| PathBuf::from("."))
            .join(".cloudreve")
            .join("logs");

        Self {
            log_dir,
            file_prefix: "cloudreve-sync".to_string(),
            max_files: 5,
            log_to_file: true,
            log_level: "info".to_string(),
        }
    }
}

impl LogConfig {
    /// Create LogConfig from ConfigManager settings
    pub fn from_config_manager() -> Self {
        if let Some(config_manager) = ConfigManager::try_get() {
            let config = config_manager.get_config();
            Self {
                log_dir: ConfigManager::get_log_dir(),
                file_prefix: "cloudreve-sync".to_string(),
                max_files: config.log_max_files,
                log_to_file: config.log_to_file,
                log_level: config.log_level.as_str().to_string(),
            }
        } else {
            Self::default()
        }
    }
}

/// Global flag for whether file logging is enabled
static FILE_LOGGING_ENABLED: OnceLock<std::sync::RwLock<bool>> = OnceLock::new();

/// Initialize the logging system with both file and stdout output
///
/// This sets up:
/// - File logging with rotation (max 5 files by default)
/// - Stdout logging with colors
/// - Component-specific log targets (api, drive, events, sync)
/// - Configurable log levels via RUST_LOG environment variable
///
/// # Log Targets
/// - `api` - API requests and responses
/// - `api::health` - Health check endpoints
/// - `api::drives` - Drive management operations
/// - `api::sync` - Sync operations
/// - `api::sse` - Server-Sent Events
/// - `api::error` - API error responses
/// - `drive` - DriveManager operations
/// - `events` - Event broadcasting
/// - `main` - Application lifecycle
///
/// # Example
/// ```bash
/// # Set log level for all components
/// RUST_LOG=debug cargo run
///
/// # Set different levels for different components
/// RUST_LOG=api=debug,drive=info,events=trace cargo run
///
/// # Show only specific component
/// RUST_LOG=api::drives=debug cargo run
/// ```
pub fn init_logging(config: LogConfig) -> Result<LogGuard> {
    // Ensure log directory exists
    std::fs::create_dir_all(&config.log_dir).context("Failed to create log directory")?;

    // Store file logging state
    FILE_LOGGING_ENABLED.get_or_init(|| std::sync::RwLock::new(config.log_to_file));

    // Configure environment filter with defaults
    let env_filter = EnvFilter::try_from_default_env()
        .unwrap_or_else(|_| EnvFilter::new(&config.log_level));

    // Initialize the subscriber based on whether file logging is enabled
    // We need separate branches due to tracing-subscriber's type system
    let worker_guard = if config.log_to_file {
        // Create file appender with rotation
        let file_appender = tracing_appender::rolling::RollingFileAppender::builder()
            .rotation(tracing_appender::rolling::Rotation::DAILY)
            .filename_prefix(&config.file_prefix)
            .filename_suffix("log")
            .max_log_files(config.max_files)
            .build(&config.log_dir)
            .context("Failed to create file appender")?;

        // Create non-blocking writer for file output
        let (non_blocking_file, worker_guard) = tracing_appender::non_blocking(file_appender);

        // Create file layer
        let file_layer = fmt::layer()
            .compact()
            .with_writer(non_blocking_file)
            .with_target(true)
            .with_thread_ids(true)
            .with_thread_names(true)
            .with_span_events(FmtSpan::NEW | FmtSpan::CLOSE);

        // Create stdout layer (human-readable with colors)
        let stdout_layer = fmt::layer()
            .compact()
            .with_target(true)
            .with_thread_ids(false)
            .with_line_number(true)
            .with_ansi(true);

        tracing_subscriber::registry()
            .with(env_filter)
            .with(file_layer)
            .with(stdout_layer)
            .init();

        worker_guard
    } else {
        // Create a dummy non-blocking writer that we won't use, just for the guard
        let (non_blocking_sink, worker_guard) = tracing_appender::non_blocking(std::io::sink());
        drop(non_blocking_sink);

        // Create stdout layer only (human-readable with colors)
        let stdout_layer = fmt::layer()
            .compact()
            .with_target(true)
            .with_thread_ids(false)
            .with_line_number(true)
            .with_ansi(true);

        tracing_subscriber::registry()
            .with(env_filter)
            .with(stdout_layer)
            .init();

        worker_guard
    };

    tracing::info!(
        target: "main",
        log_dir = %config.log_dir.display(),
        max_files = config.max_files,
        log_to_file = config.log_to_file,
        log_level = %config.log_level,
        "Logging system initialized"
    );

    Ok(LogGuard {
        _worker_guard: worker_guard,
    })
}

/// Update the log level setting (note: requires restart to take effect)
pub fn set_log_level(level: LogLevel) -> Result<()> {
    // The log level change is persisted to config but requires restart
    // because tracing-subscriber doesn't support runtime filter changes easily
    // when combined with multiple layers
    tracing::info!(target: "config", level = level.as_str(), "Log level setting updated (restart required)");
    Ok(())
}

/// Get the current file logging enabled state
pub fn is_file_logging_enabled() -> bool {
    FILE_LOGGING_ENABLED
        .get()
        .and_then(|lock| lock.read().ok())
        .map(|v| *v)
        .unwrap_or(true)
}

/// Guard that ensures logs are flushed before exit
/// This wraps the WorkerGuard from tracing_appender which MUST be kept alive
/// for the entire application lifetime to ensure file logging works properly
pub struct LogGuard {
    _worker_guard: tracing_appender::non_blocking::WorkerGuard,
}

impl Drop for LogGuard {
    fn drop(&mut self) {
        tracing::info!(target: "main", "Flushing logs before shutdown");
        // WorkerGuard will be dropped here, flushing remaining logs
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_default_log_config() {
        let config = LogConfig::default();
        assert_eq!(config.file_prefix, "cloudreve-sync");
        assert_eq!(config.max_files, 5);
        assert!(config.log_to_file);
        assert_eq!(config.log_level, "info");
    }
}
