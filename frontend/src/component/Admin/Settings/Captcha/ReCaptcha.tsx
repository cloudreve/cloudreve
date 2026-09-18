import { Trans, useTranslation } from "react-i18next";
import { FormControl, Link, Stack } from "@mui/material";
import SettingForm from "../../../Pages/Setting/SettingForm.tsx";
import { DenseFilledTextField } from "../../../Common/StyledComponents.tsx";
import * as React from "react";
import { NoMarginHelperText } from "../Settings.tsx";

export interface ReCaptchaProps {
  values: {
    [key: string]: string;
  };
  setSettings: (settings: { [key: string]: string }) => void;
}

const ReCaptcha = ({ values, setSettings }: ReCaptchaProps) => {
  const { t } = useTranslation("dashboard");
  return (
    <Stack spacing={3}>
      <SettingForm title={t("settings.siteKey")} lgWidth={5}>
        <FormControl fullWidth>
          <DenseFilledTextField
            value={values.captcha_ReCaptchaKey}
            onChange={(e) =>
              setSettings({
                captcha_ReCaptchaKey: e.target.value,
              })
            }
            required
          />
          <NoMarginHelperText>
            <Trans
              i18nKey="settings.siteKeyDes"
              ns={"dashboard"}
              components={[<Link key={0} href={"https://www.google.com/recaptcha/admin/create"} target={"_blank"} />]}
            />
          </NoMarginHelperText>
        </FormControl>
      </SettingForm>
      <SettingForm title={t("settings.siteSecret")} lgWidth={5}>
        <FormControl fullWidth>
          <DenseFilledTextField
            value={values.captcha_ReCaptchaSecret}
            onChange={(e) =>
              setSettings({
                captcha_ReCaptchaSecret: e.target.value,
              })
            }
            required
          />
          <NoMarginHelperText>
            <Trans
              i18nKey="settings.siteSecretDes"
              ns={"dashboard"}
              components={[<Link key={0} href={"https://www.google.com/recaptcha/admin/create"} target={"_blank"} />]}
            />
          </NoMarginHelperText>
        </FormControl>
      </SettingForm>
    </Stack>
  );
};

export default ReCaptcha;
