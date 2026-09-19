import {
  Alert,
  Box,
  Button,
  Collapse,
  Link,
  ListItem,
  ListItemIcon,
  ListItemText,
  Stack,
  Typography,
} from "@mui/material";
import { WarningAmber as WarningIcon } from "@mui/icons-material";
import { invoke } from "@tauri-apps/api/core";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { PendingConflict } from "./types";
import { getFileName, getParentFolderName } from "./utils";
import FileIcon from "./FileIcon";

interface ConflictItemProps {
  conflict: PendingConflict;
  onResolved: () => void;
}

type ConflictAction = "keep_remote" | "overwrite_remote" | "save_as_new";

export default function ConflictItem({
  conflict,
  onResolved,
}: ConflictItemProps) {
  const { t } = useTranslation();
  const [resolvingAction, setResolvingAction] = useState<ConflictAction | null>(
    null
  );
  const [actionsOpen, setActionsOpen] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fileName = getFileName(conflict.local_path);
  const parentFolderName = getParentFolderName(conflict.local_path);

  const handleShowInExplorer = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    invoke("show_file_in_explorer", { path: conflict.local_path });
  };

  const handleResolve = async (action: ConflictAction) => {
    setError(null);
    setResolvingAction(action);
    try {
      // These action identifiers intentionally match the Rust ConflictAction
      // parser and the Windows shell/toast actions:
      // - keep_remote: discard the local conflicted copy and sync remote state
      // - overwrite_remote: force-upload the local file over the remote version
      // - save_as_new: keep both by renaming the local file before resyncing
      await invoke("resolve_conflict", {
        driveId: conflict.drive_id,
        fileId: conflict.id,
        path: conflict.local_path,
        action,
      });
      onResolved();
    } catch (error) {
      setError(String(error));
    } finally {
      setResolvingAction(null);
    }
  };

  const isResolving = resolvingAction !== null;

  return (
    <ListItem
      alignItems="flex-start"
      sx={{
        px: 2,
        py: 1.25,
        "&:hover": {
          bgcolor: "action.hover",
        },
      }}
    >
      <ListItemIcon sx={{ minWidth: 40, pt: 0.25 }}>
        <Box sx={{ position: "relative", width: 28, height: 28 }}>
          <FileIcon path={conflict.local_path} size={28} />
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
            <WarningIcon sx={{ fontSize: 14 }} color="warning" />
          </Box>
        </Box>
      </ListItemIcon>
      <ListItemText
        disableTypography
        primary={
          <Typography variant="body2" noWrap sx={{ fontWeight: 600 }}>
            {fileName}
          </Typography>
        }
        secondary={
          <Box sx={{ mt: 0.25 }}>
            <Typography
              variant="caption"
              color="text.secondary"
              component="span"
            >
              {t(
                "popup.conflictDescription",
                "Local file conflicts with the remote version"
              )}
            </Typography>
            <Stack
              direction="row"
              spacing={1}
              alignItems="center"
              justifyContent="space-between"
              sx={{ mt: 1, minWidth: 0 }}
            >
              <Link
                component="button"
                variant="caption"
                color="text.secondary"
                onClick={handleShowInExplorer}
                underline="always"
                sx={{
                  minWidth: 0,
                  overflow: "hidden",
                  textOverflow: "ellipsis",
                  whiteSpace: "nowrap",
                }}
              >
                {parentFolderName}
              </Link>
              <Button
                size="small"
                variant="contained"
                color="warning"
                disabled={isResolving}
                onClick={() => setActionsOpen((open) => !open)}
                sx={{ flexShrink: 0 }}
              >
                {t("popup.resolveConflict", "Resolve conflict")}
              </Button>
            </Stack>
            <Collapse in={actionsOpen} timeout="auto" unmountOnExit>
              <Box sx={{ display: "flex", gap: 0.75, flexWrap: "wrap", mt: 1 }}>
                <Button
                  size="small"
                  variant="outlined"
                  disabled={isResolving}
                  onClick={() => handleResolve("keep_remote")}
                >
                  {resolvingAction === "keep_remote"
                    ? t("popup.resolving", "Resolving...")
                    : t("popup.keepRemote", "Keep remote")}
                </Button>
                <Button
                  size="small"
                  variant="contained"
                  disabled={isResolving}
                  onClick={() => handleResolve("overwrite_remote")}
                >
                  {resolvingAction === "overwrite_remote"
                    ? t("popup.resolving", "Resolving...")
                    : t("popup.overwriteRemote", "Overwrite remote")}
                </Button>
                <Button
                  size="small"
                  variant="outlined"
                  disabled={isResolving}
                  onClick={() => handleResolve("save_as_new")}
                >
                  {resolvingAction === "save_as_new"
                    ? t("popup.resolving", "Resolving...")
                    : t("popup.saveAsNew", "Save as new")}
                </Button>
              </Box>
            </Collapse>
            {error && (
              <Alert severity="error" sx={{ mt: 1, py: 0 }}>
                {error}
              </Alert>
            )}
          </Box>
        }
      />
    </ListItem>
  );
}
