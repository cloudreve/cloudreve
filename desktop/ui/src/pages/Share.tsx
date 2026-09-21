import {
  Alert,
  Box,
  Button,
  Checkbox,
  CircularProgress,
  FormControlLabel,
  IconButton,
  InputAdornment,
  MenuItem,
  Snackbar,
  Typography,
} from "@mui/material";
import CloseIcon from "@mui/icons-material/Close";
import ContentCopyIcon from "@mui/icons-material/ContentCopy";
import { invoke } from "@tauri-apps/api/core";
import { getCurrentWindow } from "@tauri-apps/api/window";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useSearchParams } from "react-router-dom";
import { DenseFilledTextField } from "../common/StyledComponent";

type Permission = "view" | "preview" | "edit" | "upload";

const EXPIRE_OPTIONS = [
  { value: 0, label: "never" },
  { value: 3600, label: "oneHour" },
  { value: 86400, label: "oneDay" },
  { value: 604800, label: "sevenDays" },
  { value: 2592000, label: "thirtyDays" },
] as const;

export default function Share() {
  const { t } = useTranslation();
  const [params] = useSearchParams();
  const driveId = params.get("drive") ?? "";
  const uri = params.get("uri") ?? "";
  const name = params.get("name") ?? "";
  const isDir = params.get("dir") === "1";

  const [permission, setPermission] = useState<Permission>("view");
  const [expire, setExpire] = useState<number>(0);
  const [usePassword, setUsePassword] = useState(false);
  const [password, setPassword] = useState("");
  const [downloads, setDownloads] = useState("");
  const [creating, setCreating] = useState(false);
  const [shareUrl, setShareUrl] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const createShare = async () => {
    setCreating(true);
    setError(null);
    try {
      const url = await invoke<string>("create_share", {
        driveId,
        uri,
        options: {
          password: usePassword && password ? password : null,
          downloads: downloads ? parseInt(downloads, 10) : null,
          expire: expire > 0 ? expire : null,
          previewOnly: permission === "preview",
          allowEdit: permission === "edit",
          allowUpload: permission === "upload",
          uploadOnly: permission === "upload",
        },
      });
      setShareUrl(url);
    } catch (e) {
      setError(String(e));
    } finally {
      setCreating(false);
    }
  };

  const copyLink = async () => {
    if (!shareUrl) return;
    await navigator.clipboard.writeText(shareUrl);
    setCopied(true);
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
      {/* Title bar with drag region */}
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
          {t("share.title", { name })}
        </Typography>
        <IconButton
          size="small"
          onClick={() => getCurrentWindow().close()}
          sx={{ WebkitAppRegion: "no-drag", appRegion: "no-drag" }}
        >
          <CloseIcon fontSize="small" />
        </IconButton>
      </Box>

      <Box sx={{ flex: 1, overflow: "auto", px: 3, pb: 2 }}>
        {shareUrl ? (
          <>
            <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
              {t("share.linkReady")}
            </Typography>
            <DenseFilledTextField
              fullWidth
              value={shareUrl}
              slotProps={{
                input: {
                  readOnly: true,
                  endAdornment: (
                    <InputAdornment position="end">
                      <IconButton size="small" onClick={copyLink} edge="end">
                        <ContentCopyIcon fontSize="small" />
                      </IconButton>
                    </InputAdornment>
                  ),
                },
              }}
              sx={{ mt: 1.5 }}
              onFocus={(e) => e.target.select()}
            />
          </>
        ) : (
          <>
            <DenseFilledTextField
              select
              fullWidth
              label={t("share.permission")}
              value={permission}
              onChange={(e) => setPermission(e.target.value as Permission)}
              sx={{ mt: 1 }}
            >
              <MenuItem value="view">{t("share.permViewDownload")}</MenuItem>
              <MenuItem value="preview">{t("share.permPreviewOnly")}</MenuItem>
              <MenuItem value="edit">{t("share.permAllowEdit")}</MenuItem>
              {isDir && (
                <MenuItem value="upload">{t("share.permUploadOnly")}</MenuItem>
              )}
            </DenseFilledTextField>

            <DenseFilledTextField
              select
              fullWidth
              label={t("share.expires")}
              value={expire}
              onChange={(e) => setExpire(Number(e.target.value))}
              sx={{ mt: 2 }}
            >
              {EXPIRE_OPTIONS.map((o) => (
                <MenuItem key={o.value} value={o.value}>
                  {t(`share.expire_${o.label}`)}
                </MenuItem>
              ))}
            </DenseFilledTextField>

            <FormControlLabel
              control={
                <Checkbox
                  checked={usePassword}
                  onChange={(e) => setUsePassword(e.target.checked)}
                  size="small"
                />
              }
              label={t("share.passwordProtect")}
              sx={{ mt: 1.5, display: "flex" }}
            />
            {usePassword && (
              <DenseFilledTextField
                fullWidth
                label={t("share.password")}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                slotProps={{ htmlInput: { maxLength: 32 } }}
                sx={{ mt: 1 }}
              />
            )}

            <DenseFilledTextField
              fullWidth
              label={t("share.downloadLimit")}
              type="number"
              value={downloads}
              onChange={(e) => setDownloads(e.target.value)}
              placeholder={t("share.unlimited")}
              slotProps={{ htmlInput: { min: 1 } }}
              sx={{ mt: 2 }}
            />
          </>
        )}

        {error && (
          <Alert severity="error" sx={{ mt: 2 }}>
            {error}
          </Alert>
        )}
      </Box>

      {/* Footer */}
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
        {shareUrl ? (
          <Button variant="contained" onClick={() => getCurrentWindow().close()}>
            {t("share.done")}
          </Button>
        ) : (
          <>
            <Button onClick={() => getCurrentWindow().close()}>
              {t("share.cancel")}
            </Button>
            <Button
              variant="contained"
              onClick={createShare}
              disabled={creating || (usePassword && !password)}
              startIcon={
                creating ? <CircularProgress size={16} color="inherit" /> : null
              }
            >
              {t("share.create")}
            </Button>
          </>
        )}
      </Box>

      <Snackbar
        open={copied}
        autoHideDuration={2000}
        onClose={() => setCopied(false)}
        message={t("share.copied")}
        anchorOrigin={{ vertical: "bottom", horizontal: "center" }}
      />
    </Box>
  );
}
