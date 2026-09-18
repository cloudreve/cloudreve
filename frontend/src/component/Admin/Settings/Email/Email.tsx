import { Box, DialogContent, FormControl, FormControlLabel, Stack, Switch, Typography } from "@mui/material";
import { useSnackbar } from "notistack";
import { useContext, useState } from "react";
import { useTranslation } from "react-i18next";
import { sendTestSMTP } from "../../../../api/api.ts";
import { useAppDispatch } from "../../../../redux/hooks.ts";
import { isTrueVal } from "../../../../session/utils.ts";
import { DefaultCloseAction } from "../../../Common/Snackbar/snackbar.tsx";
import { DenseFilledTextField, SecondaryButton } from "../../../Common/StyledComponents.tsx";
import DraggableDialog, { StyledDialogContentText } from "../../../Dialogs/DraggableDialog.tsx";
import MailOutlined from "../../../Icons/MailOutlined.tsx";
import SettingForm from "../../../Pages/Setting/SettingForm.tsx";
import { NoMarginHelperText, SettingSection, SettingSectionContent } from "../Settings.tsx";
import { SettingContext } from "../SettingWrapper.tsx";
import EmailTemplates from "./EmailTemplates.tsx";

const Email = () => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const { enqueueSnackbar } = useSnackbar();
  const { formRef, setSettings, values } = useContext(SettingContext);
  const [testEmailOpen, setTestEmailOpen] = useState(false);
  const [testEmailAddress, setTestEmailAddress] = useState("");
  const [sending, setSending] = useState(false);

  const handleTestEmail = async () => {
    setSending(true);
    try {
      await dispatch(
        sendTestSMTP({
          to: testEmailAddress,
          settings: values,
        }),
      );
      enqueueSnackbar({
        message: t("settings.testMailSent"),
        variant: "success",
        action: DefaultCloseAction,
      });
      setTestEmailOpen(false);
    } catch (error) {
    } finally {
      setSending(false);
    }
  };

  return (
    <Box component={"form"} ref={formRef} onSubmit={(e) => e.preventDefault()}>
      <Stack spacing={5}>
        <DraggableDialog
          dialogProps={{
            open: testEmailOpen,
            onClose: () => setTestEmailOpen(false),
          }}
          loading={sending}
          showActions
          showCancel
          onAccept={handleTestEmail}
          title={t("settings.testSMTPSettings")}
        >
          <DialogContent>
            <StyledDialogContentText sx={{ mb: 2 }}>{t("settings.testSMTPTooltip")}</StyledDialogContentText>
            <SettingForm title={t("settings.recipient")} lgWidth={12}>
              <DenseFilledTextField
                required
                autoFocus
                value={testEmailAddress}
                onChange={(e) => setTestEmailAddress(e.target.value)}
                type="email"
                fullWidth
              />
            </SettingForm>
          </DialogContent>
        </DraggableDialog>

        <SettingSection>
          <Typography variant="h6" gutterBottom>
            {t("settings.smtp")}
          </Typography>
          <SettingSectionContent>
            <SettingForm title={t("settings.senderName")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  required
                  value={values.fromName ?? ""}
                  onChange={(e) => setSettings({ fromName: e.target.value })}
                />
                <NoMarginHelperText>{t("settings.senderNameDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>

            <SettingForm title={t("settings.senderAddress")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  type="email"
                  required
                  value={values.fromAdress ?? ""}
                  onChange={(e) => setSettings({ fromAdress: e.target.value })}
                />
                <NoMarginHelperText>{t("settings.senderAddressDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>

            <SettingForm title={t("settings.smtpServer")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  required
                  value={values.smtpHost ?? ""}
                  onChange={(e) => setSettings({ smtpHost: e.target.value })}
                />
                <NoMarginHelperText>{t("settings.smtpServerDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>

            <SettingForm title={t("settings.smtpPort")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  type="number"
                  required
                  inputProps={{ min: 1, step: 1 }}
                  value={values.smtpPort ?? ""}
                  onChange={(e) => setSettings({ smtpPort: e.target.value })}
                />
                <NoMarginHelperText>{t("settings.smtpPortDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>

            <SettingForm title={t("settings.smtpUsername")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  required
                  value={values.smtpUser ?? ""}
                  onChange={(e) => setSettings({ smtpUser: e.target.value })}
                />
                <NoMarginHelperText>{t("settings.smtpUsernameDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>

            <SettingForm title={t("settings.smtpPassword")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  type="password"
                  required
                  value={values.smtpPass ?? ""}
                  onChange={(e) => setSettings({ smtpPass: e.target.value })}
                />
                <NoMarginHelperText>{t("settings.smtpPasswordDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>

            <SettingForm title={t("settings.replyToAddress")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  required
                  value={values.replyTo ?? ""}
                  onChange={(e) => setSettings({ replyTo: e.target.value })}
                />
                <NoMarginHelperText>{t("settings.replyToAddressDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>

            <SettingForm lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel
                  control={
                    <Switch
                      checked={isTrueVal(values.smtpEncryption)}
                      onChange={(e) => setSettings({ smtpEncryption: e.target.checked ? "1" : "0" })}
                    />
                  }
                  label={t("settings.enforceSSL")}
                />
                <NoMarginHelperText>{t("settings.enforceSSLDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>

            <SettingForm title={t("settings.smtpTTL")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  type="number"
                  required
                  inputProps={{ min: 1, step: 1 }}
                  value={values.mail_keepalive ?? "30"}
                  onChange={(e) => setSettings({ mail_keepalive: e.target.value })}
                />
                <NoMarginHelperText>{t("settings.smtpTTLDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>

            <Box display="flex" gap={2} mt={2}>
              <SecondaryButton variant="contained" startIcon={<MailOutlined />} onClick={() => setTestEmailOpen(true)}>
                {t("settings.sendTestEmail")}
              </SecondaryButton>
            </Box>
          </SettingSectionContent>
        </SettingSection>

        {/* Email Templates Section */}
        <EmailTemplates />
      </Stack>
    </Box>
  );
};

export default Email;
