import {
  Box,
  LinearProgress,
  Link,
  ListItem,
  ListItemIcon,
  ListItemText,
  Typography,
} from "@mui/material";
import {
  CheckCircle as CheckCircleIcon,
  Error as ErrorIcon,
  CloudUpload as UploadIcon,
  CloudDownload as DownloadIcon,
} from "@mui/icons-material";
import { invoke } from "@tauri-apps/api/core";
import TimeAgo from "react-timeago";
import { useTranslation } from "react-i18next";
import type { TaskWithProgress, TaskRecord } from "./types";
import { formatBytes, getFileName, getParentFolderName } from "./utils";
import FileIcon from "./FileIcon";

interface TaskItemProps {
  task: TaskWithProgress | TaskRecord;
  isActive?: boolean;
}

export default function TaskItem({ task, isActive = false }: TaskItemProps) {
  const { t } = useTranslation();
  const activeTask = task as TaskWithProgress;
  const liveProgress = activeTask.live_progress;
  const progress = liveProgress?.progress ?? task.progress;
  const isUpload = task.task_type === "upload";
  const fileName = getFileName(task.local_path);
  const parentFolderName = getParentFolderName(task.local_path);
  const isFailed = task.status === "Failed";

  const timeAgoFormatter = (
    value: number,
    unit: string,
    suffix: string
  ): string => {
    if (unit === "second") {
      return t("timeAgo.justNow", "Just now");
    }
    const unitKey = value === 1 ? unit : `${unit}s`;
    return t(`timeAgo.${unitKey}${suffix === "ago" ? "Ago" : "FromNow"}`, `${value} ${unitKey} ${suffix}`, { value });
  };

  const handleShowInExplorer = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    invoke("show_file_in_explorer", { path: task.local_path });
  };

  const getStatusBadge = () => {
    if (isActive) {
      return isUpload ? (
        <UploadIcon sx={{ fontSize: 14 }} color="primary" />
      ) : (
        <DownloadIcon sx={{ fontSize: 14 }} color="primary" />
      );
    }
    switch (task.status) {
      case "Completed":
        return <CheckCircleIcon sx={{ fontSize: 14 }} color="success" />;
      case "Failed":
      case "Cancelled":
        return <ErrorIcon sx={{ fontSize: 14 }} color="error" />;
      default:
        return null;
    }
  };

  const getSecondaryText = () => {
    if (isActive && liveProgress) {
      const processed = formatBytes(liveProgress.processed_bytes ?? 0);
      const total = formatBytes(liveProgress.total_bytes ?? 0);
      const speed = formatBytes(liveProgress.speed_bytes_per_sec);
      return `${processed} / ${total} - ${speed}/s`;
    }
    if (isActive) {
      return task.status === "Pending"
        ? t("popup.waiting", "Waiting...")
        : t("popup.processing", "Processing...");
    }
    return null;
  };

  const statusBadge = getStatusBadge();
  const secondaryText = getSecondaryText();

  return (
    <ListItem
      sx={{
        px: 2,
        py: 1,
        "&:hover": {
          bgcolor: "action.hover",
        },
      }}
    >
      <ListItemIcon sx={{ minWidth: 40 }}>
        <Box sx={{ position: "relative", width: 28, height: 28 }}>
          <FileIcon path={task.local_path} size={28} />
          {statusBadge && (
            <Box
              sx={{
                position: "absolute",
                bottom: -4,
                right: -4,
                bgcolor: "background.paper",
                borderRadius: "50%",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                width: 18,
                height: 18,
              }}
            >
              {statusBadge}
            </Box>
          )}
        </Box>
      </ListItemIcon>
      <ListItemText
        primary={
          <Typography variant="body2" noWrap sx={{ fontWeight: 500 }}>
            {fileName}
          </Typography>
        }
        secondary={
          <Box>
            {isFailed && task.error ? (
              <Typography variant="caption" color="error" component="span">
                {task.error}
              </Typography>
            ) : (
              <Typography variant="caption" color="text.secondary" component="span">
                {secondaryText ?? (
                  <TimeAgo
                    date={task.updated_at * 1000}
                    formatter={timeAgoFormatter}
                  />
                )}
              </Typography>
            )}
            {!isActive && (
              <>
                <Typography variant="caption" color="text.secondary" component="span">
                  {" · "}
                </Typography>
                <Link
                  component="button"
                  variant="caption"
                  color="text.secondary"
                  onClick={handleShowInExplorer}
                  underline="always"
                  sx={{
                  }}
                >
                  {parentFolderName}
                </Link>
              </>
            )}
            {isActive && (
              <LinearProgress
                variant="determinate"
                value={progress * 100}
                sx={{ mt: 0.5, height: 4, borderRadius: 2 }}
              />
            )}
          </Box>
        }
      />
    </ListItem>
  );
}
