import {
  Box,
  Typography,
  Switch,
  Divider,
  Link,
  Select,
  MenuItem,
  FormControl,
} from "@mui/material";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { invoke } from "@tauri-apps/api/core";
import { languages } from "../../i18n";

interface SettingItemProps {
  title: string;
  description?: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
  disabled?: boolean;
  isLast?: boolean;
}

function SettingItem({
  title,
  description,
  checked,
  onChange,
  disabled,
  isLast,
}: SettingItemProps) {
  return (
    <>
      <Box
        sx={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          py: 1.5,
          px: 2,
        }}
      >
        <Box sx={{ flex: 1, minWidth: 0, mr: 2 }}>
          <Typography variant="body2">{title}</Typography>
          {description && (
            <Typography variant="caption" color="text.secondary">
              {description}
            </Typography>
          )}
        </Box>
        <Switch
          checked={checked}
          onChange={(e) => onChange(e.target.checked)}
          disabled={disabled}
        />
      </Box>
      {!isLast && <Divider />}
    </>
  );
}

interface SettingSelectItemProps {
  title: string;
  description?: string;
  value: string;
  options: { value: string; label: string }[];
  onChange: (value: string) => void;
  disabled?: boolean;
  isLast?: boolean;
}

function SettingSelectItem({
  title,
  description,
  value,
  options,
  onChange,
  disabled,
  isLast,
}: SettingSelectItemProps) {
  return (
    <>
      <Box
        sx={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          py: 1.5,
          px: 2,
        }}
      >
        <Box sx={{ flex: 1, minWidth: 0, mr: 2 }}>
          <Typography variant="body2">{title}</Typography>
          {description && (
            <Typography variant="caption" color="text.secondary">
              {description}
            </Typography>
          )}
        </Box>
        <FormControl size="small" sx={{ minWidth: 100 }}>
          <Select
            value={value}
            onChange={(e) => onChange(e.target.value)}
            disabled={disabled}
            size="small"
          >
            {options.map((option) => (
              <MenuItem key={option.value} value={option.value}>
                {option.label}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
      </Box>
      {!isLast && <Divider />}
    </>
  );
}

interface SettingActionItemProps {
  title: string;
  description?: string;
  actionLabel: string;
  onAction: () => void;
  disabled?: boolean;
  isLast?: boolean;
}

function SettingActionItem({
  title,
  description,
  actionLabel,
  onAction,
  disabled,
  isLast,
}: SettingActionItemProps) {
  return (
    <>
      <Box
        sx={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          py: 1.5,
          px: 2,
        }}
      >
        <Box sx={{ flex: 1, minWidth: 0, mr: 2 }}>
          <Typography variant="body2">{title}</Typography>
          {description && (
            <Typography variant="caption" color="text.secondary">
              {description}
            </Typography>
          )}
        </Box>
        <Link
          component="button"
          variant="body2"
          onClick={onAction}
          disabled={disabled}
          sx={{ whiteSpace: "nowrap" }}
        >
          {actionLabel}
        </Link>
      </Box>
      {!isLast && <Divider />}
    </>
  );
}

interface SettingsGroupProps {
  title: string;
  children: React.ReactNode;
}

function SettingsGroup({ title, children }: SettingsGroupProps) {
  return (
    <Box sx={{ mb: 3 }}>
      <Typography
        variant="caption"
        color="text.secondary"
        sx={{ mb: 1, display: "block", px: 0.5 }}
      >
        {title}
      </Typography>
      <Box
        sx={{
          borderRadius: 1,
          overflow: "hidden",
          border: 1,
          borderColor: "divider",
          bgcolor: (theme) =>
            theme.palette.mode === "light"
              ? theme.palette.grey[50]
              : theme.palette.grey[900],
        }}
      >
        {children}
      </Box>
    </Box>
  );
}

interface GeneralSettings {
  notify_credential_expired: boolean;
  notify_file_conflict: boolean;
  fast_popup_launch: boolean;
  log_to_file: boolean;
  log_level: string;
  log_max_files: number;
  log_dir: string;
  language: string | null;
}

const LOG_LEVELS = [
  { value: "trace", label: "Trace" },
  { value: "debug", label: "Debug" },
  { value: "info", label: "Info" },
  { value: "warn", label: "Warn" },
  { value: "error", label: "Error" },
];

const MAX_FILES_OPTIONS = [
  { value: "3", label: "3" },
  { value: "5", label: "5" },
  { value: "7", label: "7" },
  { value: "10", label: "10" },
];

export default function GeneralSection() {
  const { t, i18n } = useTranslation();
  const [autoStart, setAutoStart] = useState(true);
  const [notifyCredentialExpired, setNotifyCredentialExpired] = useState(true);
  const [notifyFileConflict, setNotifyFileConflict] = useState(true);
  const [fastPopupLaunch, setFastPopupLaunch] = useState(true);
  const [logToFile, setLogToFile] = useState(true);
  const [logLevel, setLogLevel] = useState("info");
  const [logMaxFiles, setLogMaxFiles] = useState(5);
  const [logDir, setLogDir] = useState("");
  const [language, setLanguage] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const loadSettings = async () => {
      try {
        const [enabled, settings] = await Promise.all([
          invoke<boolean>("get_auto_start_enabled"),
          invoke<GeneralSettings>("get_general_settings"),
        ]);
        setAutoStart(enabled);
        setNotifyCredentialExpired(settings.notify_credential_expired);
        setNotifyFileConflict(settings.notify_file_conflict);
        setFastPopupLaunch(settings.fast_popup_launch);
        setLogToFile(settings.log_to_file);
        setLogLevel(settings.log_level);
        setLogMaxFiles(settings.log_max_files);
        setLogDir(settings.log_dir);
        setLanguage(settings.language);
      } catch (error) {
        console.error("Failed to load settings:", error);
      } finally {
        setLoading(false);
      }
    };
    loadSettings();
  }, []);

  const handleAutoStartChange = async (checked: boolean) => {
    const previousValue = autoStart;
    setAutoStart(checked);
    try {
      // set_auto_start returns the actual state after the operation
      // (may differ from requested if user denies permission)
      const actualState = await invoke<boolean>("set_auto_start", { enabled: checked });
      setAutoStart(actualState);
    } catch (error) {
      console.error("Failed to change autostart setting:", error);
      setAutoStart(previousValue);
    }
  };

  const handleNotifyCredentialExpiredChange = async (checked: boolean) => {
    const previousValue = notifyCredentialExpired;
    setNotifyCredentialExpired(checked);
    try {
      await invoke("set_notify_credential_expired", { enabled: checked });
    } catch (error) {
      console.error("Failed to change notification setting:", error);
      setNotifyCredentialExpired(previousValue);
    }
  };

  const handleNotifyFileConflictChange = async (checked: boolean) => {
    const previousValue = notifyFileConflict;
    setNotifyFileConflict(checked);
    try {
      await invoke("set_notify_file_conflict", { enabled: checked });
    } catch (error) {
      console.error("Failed to change notification setting:", error);
      setNotifyFileConflict(previousValue);
    }
  };

  const handleFastPopupLaunchChange = async (checked: boolean) => {
    const previousValue = fastPopupLaunch;
    setFastPopupLaunch(checked);
    try {
      await invoke("set_fast_popup_launch", { enabled: checked });
    } catch (error) {
      console.error("Failed to change fast popup setting:", error);
      setFastPopupLaunch(previousValue);
    }
  };

  const handleLogToFileChange = async (checked: boolean) => {
    const previousValue = logToFile;
    setLogToFile(checked);
    try {
      await invoke("set_log_to_file", { enabled: checked });
    } catch (error) {
      console.error("Failed to change log to file setting:", error);
      setLogToFile(previousValue);
    }
  };

  const handleLogLevelChange = async (value: string) => {
    const previousValue = logLevel;
    setLogLevel(value);
    try {
      await invoke("set_log_level", { level: value });
    } catch (error) {
      console.error("Failed to change log level:", error);
      setLogLevel(previousValue);
    }
  };

  const handleLogMaxFilesChange = async (value: string) => {
    const numValue = parseInt(value, 10);
    const previousValue = logMaxFiles;
    setLogMaxFiles(numValue);
    try {
      await invoke("set_log_max_files", { maxFiles: numValue });
    } catch (error) {
      console.error("Failed to change max log files:", error);
      setLogMaxFiles(previousValue);
    }
  };

  const handleOpenLogFolder = async () => {
    try {
      await invoke("open_log_folder");
    } catch (error) {
      console.error("Failed to open log folder:", error);
    }
  };

  const handleLanguageChange = async (value: string) => {
    // "auto" means use system default (null in config)
    const newLanguage = value === "auto" ? null : value;
    const previousValue = language;
    setLanguage(newLanguage);
    try {
      // Update backend config and rust_i18n
      await invoke("set_language", { language: newLanguage });
      // Update frontend i18n without refresh
      const effectiveLanguage = newLanguage ?? navigator.language;
      await i18n.changeLanguage(effectiveLanguage);
    } catch (error) {
      console.error("Failed to change language:", error);
      setLanguage(previousValue);
    }
  };

  // Get the current language value for the select, "auto" if null
  const currentLanguageValue = language ?? "auto";

  return (
    <Box>
      <SettingsGroup title={t("settings.launchSettings")}>
        <SettingItem
          title={t("settings.autoStart")}
          description={t("settings.autoStartDescription")}
          checked={autoStart}
          onChange={handleAutoStartChange}
          disabled={loading}
          isLast={false}
        />
        <SettingItem
          title={t("settings.fastPopupLaunch")}
          description={t("settings.fastPopupLaunchDescription")}
          checked={fastPopupLaunch}
          onChange={handleFastPopupLaunchChange}
          disabled={loading}
          isLast={true}
        />
      </SettingsGroup>

      <SettingsGroup title={t("settings.languageSettings")}>
        <SettingSelectItem
          title={t("settings.language")}
          description={t("settings.languageDescription")}
          value={currentLanguageValue}
          options={[
            { value: "auto", label: t("settings.languageAuto") },
            ...languages.map((lang) => ({
              value: lang.code,
              label: lang.displayName,
            })),
          ]}
          onChange={handleLanguageChange}
          disabled={loading}
          isLast={true}
        />
      </SettingsGroup>

      <SettingsGroup title={t("settings.notificationSettings")}>
        <SettingItem
          title={t("settings.notifyCredentialExpired")}
          description={t("settings.notifyCredentialExpiredDescription")}
          checked={notifyCredentialExpired}
          onChange={handleNotifyCredentialExpiredChange}
          disabled={loading}
          isLast={false}
        />
        <SettingItem
          title={t("settings.notifyFileConflict")}
          description={t("settings.notifyFileConflictDescription")}
          checked={notifyFileConflict}
          onChange={handleNotifyFileConflictChange}
          disabled={loading}
          isLast={true}
        />
      </SettingsGroup>

      <SettingsGroup title={t("settings.logSettings")}>
        <SettingActionItem
          title={t("settings.logFolder")}
          description={logDir}
          actionLabel={t("settings.openFolder")}
          onAction={handleOpenLogFolder}
          disabled={loading}
          isLast={false}
        />
        <SettingItem
          title={t("settings.logToFile")}
          description={t("settings.logToFileDescription")}
          checked={logToFile}
          onChange={handleLogToFileChange}
          disabled={loading}
          isLast={false}
        />
        <SettingSelectItem
          title={t("settings.logLevel")}
          description={t("settings.logLevelDescription")}
          value={logLevel}
          options={LOG_LEVELS}
          onChange={handleLogLevelChange}
          disabled={loading}
          isLast={false}
        />
        <SettingSelectItem
          title={t("settings.logMaxFiles")}
          description={t("settings.logMaxFilesDescription")}
          value={logMaxFiles.toString()}
          options={MAX_FILES_OPTIONS}
          onChange={handleLogMaxFilesChange}
          disabled={loading}
          isLast={true}
        />
      </SettingsGroup>
    </Box>
  );
}
