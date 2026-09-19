import { LoadingButton } from "@mui/lab";
import {
  DialogActions,
  DialogContent,
  FormControl,
  InputLabel,
  ListItemText,
  TextField,
  Typography,
} from "@mui/material";
import { enqueueSnackbar } from "notistack";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { sendReportAbuse } from "../../../../api/api.ts";
import { closeReportAbuseDialog } from "../../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../../redux/hooks.ts";
import { Captcha, CaptchaParams } from "../../../Common/Captcha/Captcha.tsx";
import { DenseSelect } from "../../../Common/StyledComponents.tsx";
import { DefaultCloseAction } from "../../../Common/Snackbar/snackbar.tsx";
import DraggableDialog from "../../../Dialogs/DraggableDialog.tsx";
import { SquareMenuItem } from "../../ContextMenu/ContextMenu.tsx";

const ReportAbuseDialog = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();

  const open = useAppSelector((state) => state.globalState.reportAbuseDialogOpen);
  const targetType = useAppSelector((state) => state.globalState.reportAbuseTargetType);
  const target = useAppSelector((state) => state.globalState.reportAbuseTarget);
  const captchaEnabled = useAppSelector((state) => state.siteConfig.basic.config.abuse_captcha);

  const reasonOptions = t("application:vas.reportReasonOptions", { returnObjects: true }) as string[];

  const [reason, setReason] = useState(0);
  const [description, setDescription] = useState("");
  const [loading, setLoading] = useState(false);
  const [captchaGen, setCaptchaGen] = useState(0);
  const captchaState = useRef<CaptchaParams>({});

  useEffect(() => {
    if (open) {
      setReason(0);
      setDescription("");
      captchaState.current = {};
    }
  }, [open]);

  const onClose = () => dispatch(closeReportAbuseDialog());

  const onSubmit = () => {
    if (!targetType || !target) {
      return;
    }
    setLoading(true);
    dispatch(
      sendReportAbuse({
        target_type: targetType as "share" | "user",
        target,
        reason,
        description: description || undefined,
        ...captchaState.current,
      }),
    )
      .then(() => {
        enqueueSnackbar({
          message: t("application:vas.reportAbuseSuccess"),
          variant: "success",
          action: DefaultCloseAction,
        });
        onClose();
      })
      .catch(() => {
        setCaptchaGen((g) => g + 1);
      })
      .finally(() => setLoading(false));
  };

  return (
    <DraggableDialog
      title={t("application:vas.report")}
      dialogProps={{
        open: open ?? false,
        onClose,
        fullWidth: true,
        maxWidth: "sm",
      }}
    >
      <DialogContent>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          {t("application:vas.reportTarget")}: {target}
        </Typography>
        <FormControl fullWidth sx={{ mb: 2 }}>
          <InputLabel shrink>{t("application:vas.reportReason")}</InputLabel>
          <DenseSelect
            notched
            label={t("application:vas.reportReason")}
            value={String(reason)}
            onChange={(e) => setReason(parseInt(e.target.value as string))}
          >
            {reasonOptions.map((r, i) => (
              <SquareMenuItem key={i} value={String(i)}>
                <ListItemText slotProps={{ primary: { variant: "body2" } }}>{r}</ListItemText>
              </SquareMenuItem>
            ))}
          </DenseSelect>
        </FormControl>
        <TextField
          fullWidth
          multiline
          minRows={3}
          label={t("application:vas.reportDescription")}
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          inputProps={{ maxLength: 2000 }}
        />
        {captchaEnabled && <Captcha onStateChange={(s) => (captchaState.current = s)} generation={captchaGen} />}
      </DialogContent>
      <DialogActions>
        <LoadingButton loading={loading} variant="contained" onClick={onSubmit}>
          {t("common:ok")}
        </LoadingButton>
      </DialogActions>
    </DraggableDialog>
  );
};

export default ReportAbuseDialog;
