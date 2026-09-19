import { useTranslation } from "react-i18next";
import { useSnackbar } from "notistack";
import { useAppDispatch } from "../../../redux/hooks.ts";
import React, { useState } from "react";
import { DialogContent, FormControl, FormHelperText, Typography } from "@mui/material";
import DraggableDialog from "../../Dialogs/DraggableDialog.tsx";
import { DenseFilledTextField } from "../../Common/StyledComponents.tsx";
import { sendRequestEmailChange } from "../../../api/api.ts";

export interface ChangeEmailDialogProps {
  open?: boolean;
  onClose: () => void;
  currentEmail?: string;
}

const ChangeEmailDialog = ({ open, onClose, currentEmail }: ChangeEmailDialogProps) => {
  const { t } = useTranslation();
  const { enqueueSnackbar } = useSnackbar();
  const dispatch = useAppDispatch();

  const [loading, setLoading] = useState(false);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");

  const submit = () => {
    setLoading(true);
    dispatch(sendRequestEmailChange(email, password))
      .then(() => {
        enqueueSnackbar({ message: t("setting.emailChangeSent"), variant: "success" });
        setEmail("");
        setPassword("");
        onClose();
      })
      .finally(() => setLoading(false));
  };

  return (
    <DraggableDialog
      title={t("setting.changeEmail")}
      showCancel
      onAccept={submit}
      loading={loading}
      dialogProps={{
        open: !!open,
        onClose: onClose,
        fullWidth: true,
        maxWidth: "xs",
      }}
    >
      <DialogContent>
        <Typography variant="body2" sx={{ mb: 2 }}>
          {t("setting.changeEmailDes", { email: currentEmail })}
        </Typography>
        <FormControl fullWidth sx={{ mb: 2 }}>
          <DenseFilledTextField
            required
            type="email"
            label={t("setting.newEmail")}
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            autoComplete="email"
          />
        </FormControl>
        <FormControl fullWidth>
          <DenseFilledTextField
            required
            type="password"
            label={t("login.password")}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="current-password"
          />
          <FormHelperText>{t("setting.changeEmailPasswordDes")}</FormHelperText>
        </FormControl>
      </DialogContent>
    </DraggableDialog>
  );
};

export default ChangeEmailDialog;
