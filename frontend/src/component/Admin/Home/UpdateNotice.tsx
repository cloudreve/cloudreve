import {
  Alert,
  Button,
  DialogActions,
  DialogContent,
  Typography,
} from "@mui/material";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { applyServerUpdate, getDashboardSummary, getServerUpdateInfo } from "../../../api/api.ts";
import { UpdateInfo } from "../../../api/dashboard.ts";
import DraggableDialog from "../../Dialogs/DraggableDialog.tsx";
import { DefaultButton, DenseFilledTextField } from "../../Common/StyledComponents.tsx";

const UpdateNotice = () => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const [info, setInfo] = useState<UpdateInfo>();
  const [open, setOpen] = useState(false);
  const [updating, setUpdating] = useState(false);
  const [applyError, setApplyError] = useState<string>();

  useEffect(() => {
    dispatch(getServerUpdateInfo())
      .then(setInfo)
      .catch(() => {
        // Check failures (offline, rate limit) are silent — the banner
        // simply does not appear.
      });
  }, [dispatch]);

  const doUpdate = async () => {
    setApplyError(undefined);
    setUpdating(true);
    try {
      await dispatch(applyServerUpdate());
    } catch (e) {
      setApplyError(String(e));
      setUpdating(false);
      return;
    }

    // The process restarts itself once the binary is swapped; poll until it
    // answers again, then reload so the UI picks up the new build.
    const started = Date.now();
    const poll = async () => {
      try {
        const s = await dispatch(getDashboardSummary(false));
        if (s.version.version === info?.version) {
          window.location.reload();
          return;
        }
      } catch {
        // still restarting
      }
      if (Date.now() - started < 120_000) {
        setTimeout(poll, 3000);
      } else {
        setApplyError(t("update.timeout"));
        setUpdating(false);
      }
    };
    setTimeout(poll, 4000);
  };

  if (!info?.newer) {
    return null;
  }

  return (
    <>
      <Alert
        severity="info"
        sx={{ mb: 2 }}
        action={
          <Button color="inherit" size="small" onClick={() => setOpen(true)}>
            {t("update.title")}
          </Button>
        }
      >
        {t("update.available", { version: info.version, current: info.current })}
      </Alert>

      <DraggableDialog
        title={t("update.title")}
        dialogProps={{ open, onClose: () => setOpen(false), fullWidth: true, maxWidth: "sm" }}
      >
        <DialogContent>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
            {t("update.available", { version: info.version, current: info.current })}
          </Typography>
          {info.notes && (
            <DenseFilledTextField
              fullWidth
              multiline
              minRows={4}
              maxRows={10}
              value={info.notes}
              slotProps={{ input: { readOnly: true } }}
            />
          )}
          {updating && (
            <Alert severity="info" sx={{ mt: 2 }}>
              {t("update.updating")}
            </Alert>
          )}
          {applyError && (
            <Alert severity="error" sx={{ mt: 2 }}>
              {applyError}
            </Alert>
          )}
          {info.container && (
            <Alert severity="warning" sx={{ mt: 2 }}>
              {t("update.containerHint")}
            </Alert>
          )}
          {!info.self_update && !info.container && (
            <Alert severity="warning" sx={{ mt: 2 }}>
              {t("update.unsupported")}
            </Alert>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpen(false)} disabled={updating}>
            {t("common:close")}
          </Button>
          <Button onClick={() => window.open(info.url)}>{t("update.viewRelease")}</Button>
          {info.self_update && (
            <DefaultButton variant="contained" onClick={doUpdate} disabled={updating}>
              {updating ? t("update.updating") : t("update.updateNow")}
            </DefaultButton>
          )}
        </DialogActions>
      </DraggableDialog>
    </>
  );
};

export default UpdateNotice;
