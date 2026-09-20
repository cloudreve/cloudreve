import { FormControl, Stack } from "@mui/material";
import { Trans, useTranslation } from "react-i18next";
import SettingForm from "../../../Pages/Setting/SettingForm.tsx";
import { DenseFilledTextField } from "../../../Common/StyledComponents.tsx";
import { NoMarginHelperText } from "../Settings.tsx";

export interface TCaptchaProps {
  values: {
    [key: string]: string;
  };
  setSettings: (settings: { [key: string]: string }) => void;
}

const TCaptcha = ({ values, setSettings }: TCaptchaProps) => {
  const { t } = useTranslation("dashboard");
  return (
    <Stack spacing={3}>
      <SettingForm title={t("settings.tcaptchaAppId")} lgWidth={5}>
        <FormControl fullWidth>
          <DenseFilledTextField
            value={values.captcha_TCaptcha_CaptchaAppId}
            onChange={(e) => setSettings({ captcha_TCaptcha_CaptchaAppId: e.target.value })}
            required
          />
          <NoMarginHelperText>
            <Trans
              i18nKey="settings.tcaptchaDes"
              ns={"dashboard"}
              components={[<a key={0} href="https://console.cloud.tencent.com/captcha/graphical" target="_blank" rel="noreferrer" />]}
            />
          </NoMarginHelperText>
        </FormControl>
      </SettingForm>
      <SettingForm title={t("settings.tcaptchaAppSecretKey")} lgWidth={5}>
        <FormControl fullWidth>
          <DenseFilledTextField
            value={values.captcha_TCaptcha_AppSecretKey}
            onChange={(e) => setSettings({ captcha_TCaptcha_AppSecretKey: e.target.value })}
            required
          />
        </FormControl>
      </SettingForm>
      <SettingForm title={t("settings.tcaptchaSecretId")} lgWidth={5}>
        <FormControl fullWidth>
          <DenseFilledTextField
            value={values.captcha_TCaptcha_SecretId}
            onChange={(e) => setSettings({ captcha_TCaptcha_SecretId: e.target.value })}
            required
          />
          <NoMarginHelperText>{t("settings.tcaptchaSecretDes")}</NoMarginHelperText>
        </FormControl>
      </SettingForm>
      <SettingForm title={t("settings.tcaptchaSecretKey")} lgWidth={5}>
        <FormControl fullWidth>
          <DenseFilledTextField
            value={values.captcha_TCaptcha_SecretKey}
            onChange={(e) => setSettings({ captcha_TCaptcha_SecretKey: e.target.value })}
            required
          />
        </FormControl>
      </SettingForm>
    </Stack>
  );
};

export default TCaptcha;
