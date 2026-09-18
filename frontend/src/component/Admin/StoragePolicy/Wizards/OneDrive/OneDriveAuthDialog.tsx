import { Alert, Box, Button, DialogActions, DialogContent, Stack, Typography } from "@mui/material";
import { useEffect, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { getPolicyOauthRedirectUrl } from "../../../../../api/api";
import { useAppDispatch } from "../../../../../redux/hooks";
import { DenseFilledTextField } from "../../../../Common/StyledComponents";
import DraggableDialog from "../../../../Dialogs/DraggableDialog";
import { Code } from "../../../../Common/Code.tsx";
import { NoMarginHelperText } from "../../../Settings/Settings";

interface OneDriveAuthDialogProps {
  open: boolean;
  onClose: () => void;
  onConfirm: (appId: string, appSecret: string) => void;
  initialAppId?: string;
  initialAppSecret?: string;
}

const OneDriveAuthDialog = ({
  open,
  onClose,
  onConfirm,
  initialAppId = "",
  initialAppSecret = "",
}: OneDriveAuthDialogProps) => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const [loading, setLoading] = useState(false);
  const [redirectUrl, setRedirectUrl] = useState<string>("");
  const [appId, setAppId] = useState<string>(initialAppId);
  const [appSecret, setAppSecret] = useState<string>(initialAppSecret);
  const isHttps = window.location.protocol === "https:";

  useEffect(() => {
    if (open) {
      setLoading(true);
      dispatch(getPolicyOauthRedirectUrl()).then((res) => {
        setRedirectUrl(res);
        setLoading(false);
      });
    }
  }, [open, dispatch]);

  const handleConfirm = () => {
    onConfirm(appId, appSecret);
    onClose();
  };

  return (
    <DraggableDialog
      title={t("policy.authorizeOneDrive")}
      dialogProps={{ open, onClose, maxWidth: "sm", fullWidth: true }}
    >
      <DialogContent>
        <Stack spacing={3} sx={{ mt: 1 }}>
          <Typography variant="body2" gutterBottom>
            {t("policy.authorizeOneDriveDes")}
          </Typography>

          <Box>
            <Typography variant="subtitle2" gutterBottom>
              {t("policy.redirectUrl")}
            </Typography>
            <Typography component={"div"} variant="body2" color="textPrimary">
              {redirectUrl}
            </Typography>
            <Typography component={"div"} variant="caption" color="textSecondary">
              {t("policy.redirectUrlDes")}
            </Typography>
          </Box>

          <Box>
            <Typography variant="subtitle2" gutterBottom>
              {t("policy.aadAppID")}
            </Typography>
            <DenseFilledTextField fullWidth required value={appId} onChange={(e) => setAppId(e.target.value)} />
            <NoMarginHelperText>
              <Trans i18nKey="policy.aadAppIDDes" ns="dashboard" components={[<Code />, <Code />]} />
            </NoMarginHelperText>
          </Box>

          <Box>
            <Typography variant="subtitle2" gutterBottom>
              {t("policy.aadAppSecret")}
            </Typography>
            <DenseFilledTextField fullWidth required value={appSecret} onChange={(e) => setAppSecret(e.target.value)} />
            <NoMarginHelperText>
              <Trans i18nKey="policy.addAppSecretDes" ns="dashboard" components={[<Code />, <Code />, <Code />]} />
            </NoMarginHelperText>
          </Box>
          {!isHttps && <Alert severity="warning">{t("policy.httpsRequired")}</Alert>}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t("common:cancel")}</Button>
        <Button variant="contained" color="primary" onClick={handleConfirm} disabled={!appId || !appSecret}>
          {t("policy.authorizeMicrosoft")}
        </Button>
      </DialogActions>
    </DraggableDialog>
  );
};

export default OneDriveAuthDialog;
