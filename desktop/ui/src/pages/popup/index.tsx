import {
  Box,
  IconButton,
  List,
  Typography,
  Divider,
} from "@mui/material";
import {
  Folder as FolderIcon,
  CheckCircle as CheckCircleIcon,
  Refresh as RefreshIcon,
} from "@mui/icons-material";
import { useCallback, useEffect, useRef, useState } from "react";
import { invoke } from "@tauri-apps/api/core";
import { getCurrentWindow } from "@tauri-apps/api/window";
import { useTranslation } from "react-i18next";
import Settings from "../../common/icons/Settings";
import CloudreveLogo from "../../common/CloudreveLogo";
import type { StatusSummary } from "./types";
import DriveChips from "./DriveChips";
import TaskItem from "./TaskItem";

export default function Popup() {
  const { t } = useTranslation();
  const [summary, setSummary] = useState<StatusSummary | null>(null);
  const [selectedDrive, setSelectedDrive] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const isFetchingRef = useRef(false);

  // Close window on blur (when it loses focus)
  useEffect(() => {
    let unlisten: () => void;
    const currentWindow = getCurrentWindow();

    currentWindow
      .onFocusChanged(({ payload: focused }) => {
        if (!focused) {
          currentWindow.close();
        }
      })
      .then((fn) => {
        unlisten = fn;
      });

    return () => {
      if (unlisten) {
        unlisten();
      }
    };
  }, []);

  // Fetch status summary
  const fetchSummary = useCallback(async () => {
    if (isFetchingRef.current) return;

    isFetchingRef.current = true;
    try {
      const result = await invoke<StatusSummary>("get_status_summary", {
        driveId: selectedDrive,
      });
      setSummary(result);
    } catch (error) {
      console.error("Failed to fetch status summary:", error);
    } finally {
      isFetchingRef.current = false;
      setLoading(false);
    }
  }, [selectedDrive]);

  // Initial fetch and polling
  useEffect(() => {
    fetchSummary();

    const intervalId = setInterval(() => {
      fetchSummary();
    }, 1000);

    return () => {
      clearInterval(intervalId);
    };
  }, [fetchSummary]);

  const handleDriveSelect = (driveId: string | null) => {
    setSelectedDrive(driveId);
  };

  const handleAddDrive = async () => {
    try {
      await invoke("show_add_drive_window");
    } catch (error) {
      console.error("Failed to open add drive window:", error);
    }
  };

  const handleSettings = async () => {
    try {
      await invoke("show_settings_window");
    } catch (error) {
      console.error("Failed to open settings window:", error);
    }
  };

  const hasActiveTasks =
    summary?.active_tasks && summary.active_tasks.length > 0;
  const hasFinishedTasks =
    summary?.finished_tasks && summary.finished_tasks.length > 0;

  return (
    <Box
      sx={{
        height: "100vh",
        display: "flex",
        flexDirection: "column",
        bgcolor: "background.paper",
        overflow: "hidden",
      }}
    >
      {/* Header */}
      <Box
        sx={{
          px: 2,
          pt: 1.5,
          pb: 1,
          borderBottom: 1,
          borderColor: "divider",
          backgroundColor: (theme) =>
            theme.palette.mode === "light" ? theme.palette.grey[100] : theme.palette.grey[900],
        }}
      >
        {/* Top row: Logo and Settings */}
        <Box
          sx={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            mb: 1.5,
          }}
        >
          <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
            <CloudreveLogo height={28} />
          </Box>
          <IconButton size="small" onClick={handleSettings}>
            <Settings fontSize="small" />
          </IconButton>
        </Box>

        {/* Drive filter chips */}
        <DriveChips
          drives={summary?.drives ?? []}
          selectedDrive={selectedDrive}
          onDriveSelect={handleDriveSelect}
          onAddDrive={handleAddDrive}
        />
      </Box>

      {/* Task List */}
      <Box sx={{ flex: 1, overflow: "auto" }}>
        {loading ? (
          <Box
            sx={{
              display: "flex",
              justifyContent: "center",
              alignItems: "center",
              height: "100%",
            }}
          >
            <Typography variant="body2" color="text.secondary">
              {t("popup.loading", "Loading...")}
            </Typography>
          </Box>
        ) : !hasActiveTasks && !hasFinishedTasks ? (
          <Box
            sx={{
              display: "flex",
              flexDirection: "column",
              justifyContent: "center",
              alignItems: "center",
              height: "100%",
              gap: 1,
            }}
          >
            <FolderIcon sx={{ fontSize: 48, color: "text.disabled" }} />
            <Typography variant="body2" color="text.secondary">
              {t("popup.noActivity", "No recent activity")}
            </Typography>
          </Box>
        ) : (
          <List disablePadding>
            {/* Active Tasks */}
            {hasActiveTasks && (
              <>
                <Typography
                  variant="caption"
                  color="text.secondary"
                  sx={{
                    px: 2,
                    py: 1,
                    pb:0,
                    display: "block",
                    fontWeight: 600,
                    textTransform: "uppercase",
                  }}
                >
                  {t("popup.syncing", "Syncing")}
                </Typography>
                {summary?.active_tasks.map((task) => (
                  <TaskItem key={task.id} task={task} isActive />
                ))}
              </>
            )}

            {/* Divider between active and finished */}
            {hasActiveTasks && hasFinishedTasks && (
              <Divider sx={{ my: 1 }} />
            )}

            {/* Finished Tasks */}
            {hasFinishedTasks && (
              <>
                <Typography
                  variant="caption"
                  color="text.secondary"
                  sx={{
                    px: 2,
                    py: 1,
                    pb:0,
                    display: "block",
                    fontWeight: 600,
                    textTransform: "uppercase",
                  }}
                >
                  {t("popup.recent", "Recent")}
                </Typography>
                {summary?.finished_tasks.map((task) => (
                  <TaskItem key={task.id} task={task} />
                ))}
              </>
            )}
          </List>
        )}
      </Box>

      {/* Footer Status */}
      <Box
        sx={{
          px: 2,
          py: 1,
          borderTop: 1,
          borderColor: "divider",
          display: "flex",
          alignItems: "center",
          gap: 1,
        }}
      >
        {hasActiveTasks ? (
          <RefreshIcon
            sx={{
              fontSize: 18,
              color: "primary.main",
              animation: "spin 1s linear infinite",
              "@keyframes spin": {
                "0%": { transform: "rotate(0deg)" },
                "100%": { transform: "rotate(360deg)" },
              },
            }}
          />
        ) : (
          <CheckCircleIcon
            sx={{ fontSize: 18, color: "success.main" }}
          />
        )}
        <Typography variant="caption" color="text.secondary">
          {hasActiveTasks
            ? t("popup.syncingStatus", "Syncing {{count}} file(s)...", {
                count: summary?.active_tasks.length ?? 0,
              })
            : t("popup.upToDate", "Your files are up to date")}
        </Typography>
      </Box>
    </Box>
  );
}
