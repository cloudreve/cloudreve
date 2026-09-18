import { Trans, useTranslation } from "react-i18next";
import { FormControl, Link, Stack } from "@mui/material";
import SettingForm from "../../../Pages/Setting/SettingForm.tsx";
import { DenseFilledTextField } from "../../../Common/StyledComponents.tsx";
import * as React from "react";
import { NoMarginHelperText } from "../Settings.tsx";

export interface TurnstileCaptchaProps {
  values: {
    [key: string]: string;
  };
  setSettings: (settings: { [key: string]: string }) => void;
}

const Turnstile = ({ values, setSettings }: TurnstileCaptchaProps) => {
  const { t } = useTranslation("dashboard");
  return (
    <Stack spacing={3}>
      <SettingForm title={t("settings.turnstileSiteKey")} lgWidth={5}>
        <FormControl fullWidth>
          <DenseFilledTextField
            value={values.captcha_turnstile_site_key}
            onChange={(e) =>
              setSettings({
                captcha_turnstile_site_key: e.target.value,
              })
            }
            required
          />
          <NoMarginHelperText>
            <Trans
              i18nKey="settings.siteKeyDes"
              ns={"dashboard"}
              components={[<Link key={0} href={"https://dash.cloudflare.com/"} target={"_blank"} />]}
            />
          </NoMarginHelperText>
        </FormControl>
      </SettingForm>
      <SettingForm title={t("settings.turnstileSiteKSecret")} lgWidth={5}>
        <FormControl fullWidth>
          <DenseFilledTextField
            value={values.captcha_turnstile_site_secret}
            onChange={(e) =>
              setSettings({
                captcha_turnstile_site_secret: e.target.value,
              })
            }
            required
          />
          <NoMarginHelperText>
            <Trans
              i18nKey="settings.siteSecretDes"
              ns={"dashboard"}
              components={[<Link key={0} href={"https://dash.cloudflare.com/"} target={"_blank"} />]}
            />
          </NoMarginHelperText>
        </FormControl>
      </SettingForm>
    </Stack>
  );
};

export default Turnstile;
