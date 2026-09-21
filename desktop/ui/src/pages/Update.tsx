import {
  Alert,
  Box,
  Button,
  CircularProgress,
  IconButton,
  LinearProgress,
  Typography,
} from "@mui/material";
import CloseIcon from "@mui/icons-material/Close";
import { invoke } from "@tauri-apps/api/core";
import { listen } from "@tauri-apps/api/event";
import { getCurrentWindow } from "@tauri-apps/api/window";
import { openUrl } from "@tauri-apps/plugin-opener";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { DenseFilledTextField } from "../common/StyledComponent";

type UpdateInfo = {
  version: string;
  current_version: string;
  notes?: string | null;
  date?: string | null;
};

type Phase = "checking" | "available" | "downloading" | "ready" | "uptodate" | "error";

const RELEASES_URL = "https://github.com/Dvorinka/cloudreve/releases";

export default function Update() {
  const { t } = useTranslation();
  const [phase, setPhase] = useState<Phase>("checking");
  const [info, setInfo] = useState<UpdateInfo | null>(null);
  const [progress, setProgress] = useState(0);
  const [error, setError] = useState<string | null>(null);
  const downloadedRef = useRef(0);

  useEffect(() => {
    const unProgress = listen<{ downloaded: number; total: number | null }>(
      "update-progress",
      (e) => {
        downloadedRef.current += e.payload.downloaded;
        if (e.payload.total) {
          setProgress(
            Math.min(100, (downloadedRef.current / e.payload.total) * 100)
          );
        }
      }
    );
    const unFinished = listen("update-finished", () => setPhase("ready"));

    invoke<UpdateInfo | null>("check_update")
      .then((u) => {
        setInfo(u);
        setPhase(u ? "available" : "uptodate");
      })
      .catch((e) => {
        setError(String(e));
        setPhase("error");
      });

    return () => {
      unProgress.then((f) => f());
      unFinished.then((f) => f());
    };
  }, []);

  const startUpdate = async () => {
    setPhase("downloading");
    setError(null);
    downloadedRef.current = 0;
    setProgress(0);
    try {
      await invoke("install_update");
      // The ready phase is set by the update-finished listener; if the
      // event raced past us, still land on ready.
      setPhase((p) => (p === "downloading" ? "ready" : p));
    } catch (e) {
      setError(String(e));
      setPhase("error");
    }
  };

  return (
    <Box
      sx={{
        height: "100vh",
        display: "flex",
        flexDirection: "column",
        overflow: "hidden",
      }}
    >
      <Box
        data-tauri-drag-region
        sx={{
          px: 2,
          pt: 1.5,
          pb: 1,
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          flexShrink: 0,
        }}
      >
        <Typography variant="subtitle1" fontWeight={600} noWrap>
          {t("update.title")}
        </Typography>
        <IconButton
          size="small"
          onClick={() => getCurrentWindow().close()}
          sx={{ WebkitAppRegion: "no-drag", appRegion: "no-drag" }}
        >
          <CloseIcon fontSize="small" />
        </IconButton>
      </Box>

      <Box
        sx={{
          flex: 1,
          overflow: "auto",
          px: 3,
          pb: 2,
          display: "flex",
          flexDirection: "column",
          justifyContent: "center",
          gap: 1.5,
        }}
      >
        {phase === "checking" && (
          <Box sx={{ display: "flex", justifyContent: "center", py: 4 }}>
            <CircularProgress size={28} />
          </Box>
        )}

        {phase === "available" && info && (
          <>
            <Typography variant="body1" fontWeight={600}>
              {t("update.available", {
                version: info.version,
                current: info.current_version,
              })}
            </Typography>
            {info.notes && (
              <DenseFilledTextField
                fullWidth
                multiline
                minRows={3}
                maxRows={6}
                value={info.notes}
                slotProps={{ input: { readOnly: true } }}
              />
            )}
          </>
        )}

        {phase === "downloading" && (
          <>
            <Typography variant="body2" color="text.secondary">
              {t("update.downloading")}
            </Typography>
            <LinearProgress variant="determinate" value={progress} />
            <Typography variant="caption" color="text.secondary">
              {Math.round(progress)}%
            </Typography>
          </>
        )}

        {phase === "ready" && (
          <Alert severity="success">{t("update.ready")}</Alert>
        )}

        {phase === "uptodate" && (
          <Alert severity="info">{t("update.upToDate")}</Alert>
        )}

        {phase === "error" && (
          <Alert severity="error">{error ?? t("update.failed")}</Alert>
        )}
      </Box>

      <Box
        sx={{
          px: 3,
          py: 2,
          display: "flex",
          justifyContent: "flex-end",
          gap: 1,
          flexShrink: 0,
        }}
      >
        {phase === "available" && (
          <>
            <Button onClick={() => getCurrentWindow().close()}>
              {t("update.later")}
            </Button>
            <Button variant="contained" onClick={startUpdate}>
              {t("update.updateNow")}
            </Button>
          </>
        )}
        {phase === "ready" && (
          <Button
            variant="contained"
            onClick={() => invoke("restart_app")}
          >
            {t("update.restart")}
          </Button>
        )}
        {(phase === "uptodate" || phase === "checking") && (
          <Button onClick={() => getCurrentWindow().close()}>
            {t("update.close")}
          </Button>
        )}
        {phase === "error" && (
          <>
            <Button onClick={() => getCurrentWindow().close()}>
              {t("update.close")}
            </Button>
            <Button variant="outlined" onClick={() => openUrl(RELEASES_URL)}>
              {t("update.downloadManually")}
            </Button>
          </>
        )}
      </Box>
    </Box>
  );
}
