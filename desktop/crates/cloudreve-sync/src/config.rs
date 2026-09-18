use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};
use std::fs;
use std::path::PathBuf;
use std::sync::{OnceLock, RwLock};

/// Global config manager instance
static CONFIG_MANAGER: OnceLock<ConfigManager> = OnceLock::new();

/// Log level configuration
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, Default)]
#[serde(rename_all = "lowercase")]
pub enum LogLevel {
    Trace,
    Debug,
    #[default]
    Info,
    Warn,
    Error,
}

impl LogLevel {
    pub fn as_str(&self) -> &'static str {
        match self {
            LogLevel::Trace => "trace",
            LogLevel::Debug => "debug",
            LogLevel::Info => "info",
            LogLevel::Warn => "warn",
            LogLevel::Error => "error",
        }
    }

    pub fn from_str(s: &str) -> Self {
        match s.to_lowercase().as_str() {
            "trace" => LogLevel::Trace,
            "debug" => LogLevel::Debug,
            "info" => LogLevel::Info,
            "warn" => LogLevel::Warn,
            "error" => LogLevel::Error,
            _ => LogLevel::Info,
        }
    }
}

/// Application configuration stored as JSON
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(default)]
pub struct AppConfig {
    /// Whether to automatically start the application when the system boots
    pub auto_start: bool,
    /// Whether to show notifications when credentials expire
    pub notify_credential_expired: bool,
    /// Whether to show notifications when file conflicts occur
    pub notify_file_conflict: bool,
    /// Whether to keep the popup window alive (hide instead of close) for faster launch
    pub fast_popup_launch: bool,
    /// Whether to write logs to file
    pub log_to_file: bool,
    /// Log level (trace, debug, info, warn, error)
    pub log_level: LogLevel,
    /// Maximum number of log files to keep
    pub log_max_files: usize,
    /// Language/locale setting (e.g., "en-US", "zh-CN"). None means use system default.
    pub language: Option<String>,
}

impl Default for AppConfig {
    fn default() -> Self {
        Self {
            auto_start: true,
            notify_credential_expired: true,
            notify_file_conflict: true,
            fast_popup_launch: true,
            log_to_file: true,
            log_level: LogLevel::Debug,
            log_max_files: 5,
            language: None,
        }
    }
}

/// Thread-safe configuration manager that persists settings to JSON
pub struct ConfigManager {
    config: RwLock<AppConfig>,
    config_path: PathBuf,
}

impl ConfigManager {
    /// Initialize the global config manager.
    /// This should be called once at application startup.
    pub fn init() -> Result<&'static ConfigManager> {
        let config_path = Self::get_config_path()?;
        let config = Self::load_from_path(&config_path)?;

        let manager = ConfigManager {
            config: RwLock::new(config),
            config_path,
        };

        Ok(CONFIG_MANAGER.get_or_init(|| manager))
    }

    /// Get the global config manager instance.
    /// Panics if `init()` has not been called.
    pub fn get() -> &'static ConfigManager {
        CONFIG_MANAGER
            .get()
            .expect("ConfigManager::init() must be called before ConfigManager::get()")
    }

    /// Try to get the global config manager instance.
    /// Returns None if `init()` has not been called.
    pub fn try_get() -> Option<&'static ConfigManager> {
        CONFIG_MANAGER.get()
    }

    /// Get the config file path (~/.cloudreve/config.json)
    fn get_config_path() -> Result<PathBuf> {
        let home_dir = dirs::home_dir().context("Failed to get user home directory")?;
        Ok(home_dir.join(".cloudreve").join("config.json"))
    }

    /// Load configuration from the specified path, using defaults for missing fields
    fn load_from_path(path: &PathBuf) -> Result<AppConfig> {
        if !path.exists() {
            tracing::info!(target: "config", path = %path.display(), "Config file not found, using defaults");
            return Ok(AppConfig::default());
        }

        let content = fs::read_to_string(path).context("Failed to read config file")?;

        // serde's #[serde(default)] handles missing fields automatically
        let config: AppConfig =
            serde_json::from_str(&content).context("Failed to parse config file")?;

        tracing::info!(target: "config", path = %path.display(), "Loaded configuration from file");

        Ok(config)
    }

    /// Save the current configuration to disk
    fn save(&self) -> Result<()> {
        // Ensure parent directory exists
        if let Some(parent) = self.config_path.parent() {
            if !parent.exists() {
                fs::create_dir_all(parent).context("Failed to create config directory")?;
            }
        }

        let config = self.config.read().map_err(|e| {
            anyhow::anyhow!("Failed to acquire read lock on config: {}", e)
        })?;

        let content =
            serde_json::to_string_pretty(&*config).context("Failed to serialize config")?;

        fs::write(&self.config_path, content).context("Failed to write config file")?;

        tracing::debug!(target: "config", path = %self.config_path.display(), "Configuration saved");

        Ok(())
    }

    /// Get the current configuration (cloned)
    pub fn get_config(&self) -> AppConfig {
        self.config
            .read()
            .map(|c| c.clone())
            .unwrap_or_else(|_| AppConfig::default())
    }

    /// Update the configuration with a closure and persist to disk
    pub fn update<F>(&self, f: F) -> Result<()>
    where
        F: FnOnce(&mut AppConfig),
    {
        {
            let mut config = self.config.write().map_err(|e| {
                anyhow::anyhow!("Failed to acquire write lock on config: {}", e)
            })?;
            f(&mut config);
        }
        self.save()
    }

    /// Get whether auto-start is enabled
    pub fn auto_start(&self) -> bool {
        self.config
            .read()
            .map(|c| c.auto_start)
            .unwrap_or(true)
    }

    /// Set whether auto-start is enabled
    pub fn set_auto_start(&self, enabled: bool) -> Result<()> {
        self.update(|config| {
            config.auto_start = enabled;
        })
    }

    /// Get whether credential expired notifications are enabled
    pub fn notify_credential_expired(&self) -> bool {
        self.config
            .read()
            .map(|c| c.notify_credential_expired)
            .unwrap_or(true)
    }

    /// Set whether credential expired notifications are enabled
    pub fn set_notify_credential_expired(&self, enabled: bool) -> Result<()> {
        self.update(|config| {
            config.notify_credential_expired = enabled;
        })
    }

    /// Get whether file conflict notifications are enabled
    pub fn notify_file_conflict(&self) -> bool {
        self.config
            .read()
            .map(|c| c.notify_file_conflict)
            .unwrap_or(true)
    }

    /// Set whether file conflict notifications are enabled
    pub fn set_notify_file_conflict(&self, enabled: bool) -> Result<()> {
        self.update(|config| {
            config.notify_file_conflict = enabled;
        })
    }

    /// Get whether fast popup launch is enabled
    pub fn fast_popup_launch(&self) -> bool {
        self.config
            .read()
            .map(|c| c.fast_popup_launch)
            .unwrap_or(true)
    }

    /// Set whether fast popup launch is enabled
    pub fn set_fast_popup_launch(&self, enabled: bool) -> Result<()> {
        self.update(|config| {
            config.fast_popup_launch = enabled;
        })
    }

    /// Get whether log to file is enabled
    pub fn log_to_file(&self) -> bool {
        self.config
            .read()
            .map(|c| c.log_to_file)
            .unwrap_or(true)
    }

    /// Set whether log to file is enabled
    pub fn set_log_to_file(&self, enabled: bool) -> Result<()> {
        self.update(|config| {
            config.log_to_file = enabled;
        })
    }

    /// Get the log level
    pub fn log_level(&self) -> LogLevel {
        self.config
            .read()
            .map(|c| c.log_level)
            .unwrap_or(LogLevel::Info)
    }

    /// Set the log level
    pub fn set_log_level(&self, level: LogLevel) -> Result<()> {
        self.update(|config| {
            config.log_level = level;
        })
    }

    /// Get the max log files
    pub fn log_max_files(&self) -> usize {
        self.config
            .read()
            .map(|c| c.log_max_files)
            .unwrap_or(5)
    }

    /// Set the max log files
    pub fn set_log_max_files(&self, max_files: usize) -> Result<()> {
        self.update(|config| {
            config.log_max_files = max_files;
        })
    }

    /// Get the language setting
    pub fn language(&self) -> Option<String> {
        self.config.read().ok().and_then(|c| c.language.clone())
    }

    /// Set the language setting
    pub fn set_language(&self, language: Option<String>) -> Result<()> {
        self.update(|config| {
            config.language = language;
        })
    }

    /// Get the log directory path
    pub fn get_log_dir() -> PathBuf {
        dirs::home_dir()
            .unwrap_or_else(|| PathBuf::from("."))
            .join(".cloudreve")
            .join("logs")
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Write;
    use tempfile::NamedTempFile;

    #[test]
    fn test_default_config() {
        let config = AppConfig::default();
        assert!(config.auto_start);
    }

    #[test]
    fn test_load_with_missing_fields() {
        // Create a temp file with partial config (missing auto_start)
        let mut temp_file = NamedTempFile::new().unwrap();
        writeln!(temp_file, "{{}}").unwrap();

        let config = ConfigManager::load_from_path(&temp_file.path().to_path_buf()).unwrap();
        assert!(config.auto_start); // Should use default (true)
    }

    #[test]
    fn test_load_with_all_fields() {
        let mut temp_file = NamedTempFile::new().unwrap();
        writeln!(temp_file, r#"{{"auto_start": false}}"#).unwrap();

        let config = ConfigManager::load_from_path(&temp_file.path().to_path_buf()).unwrap();
        assert!(!config.auto_start);
    }

    #[test]
    fn test_load_nonexistent_file() {
        let path = PathBuf::from("/nonexistent/path/config.json");
        let config = ConfigManager::load_from_path(&path).unwrap();
        assert!(config.auto_start); // Should use default (true)
    }
}
