import { Trans, useTranslation } from "react-i18next";
import { FormControl, Link, Stack, ListItemText } from "@mui/material";
import SettingForm from "../../../Pages/Setting/SettingForm.tsx";
import { DenseFilledTextField, DenseSelect } from "../../../Common/StyledComponents.tsx";
import { SquareMenuItem } from "../../../FileManager/ContextMenu/ContextMenu.tsx";
import * as React from "react";
import { NoMarginHelperText } from "../Settings.tsx";

export interface CapCaptchaProps {
  values: {
    [key: string]: string;
  };
  setSettings: (settings: { [key: string]: string }) => void;
}

const CapCaptcha = ({ values, setSettings }: CapCaptchaProps) => {
  const { t } = useTranslation("dashboard");
  return (
    <Stack spacing={3}>
      <SettingForm title={t("settings.capInstanceURL")} lgWidth={5}>
        <FormControl fullWidth>
          <DenseFilledTextField
            value={values.captcha_cap_instance_url}
            onChange={(e) =>
              setSettings({
                captcha_cap_instance_url: e.target.value,
              })
            }
            placeholder="https://cap.example.com"
            required
          />
          <NoMarginHelperText>
            <Trans
              i18nKey="settings.capInstanceURLDes"
              ns={"dashboard"}
              components={[<Link key={0} href={"https://capjs.js.org/guide/standalone/"} target={"_blank"} />]}
            />
          </NoMarginHelperText>
        </FormControl>
      </SettingForm>
      <SettingForm title={t("settings.capSiteKey")} lgWidth={5}>
        <FormControl fullWidth>
          <DenseFilledTextField
            value={values.captcha_cap_site_key}
            onChange={(e) =>
              setSettings({
                captcha_cap_site_key: e.target.value,
              })
            }
            required
          />
          <NoMarginHelperText>
            <Trans
              i18nKey="settings.capSiteKeyDes"
              ns={"dashboard"}
              components={[<Link key={0} href={"https://capjs.js.org/guide/standalone/"} target={"_blank"} />]}
            />
          </NoMarginHelperText>
        </FormControl>
      </SettingForm>
      <SettingForm title={t("settings.capSecretKey")} lgWidth={5}>
        <FormControl fullWidth>
          <DenseFilledTextField
            value={values.captcha_cap_secret_key}
            onChange={(e) =>
              setSettings({
                captcha_cap_secret_key: e.target.value,
              })
            }
            type="password"
            required
          />
          <NoMarginHelperText>
            <Trans
              i18nKey="settings.capSecretKeyDes"
              ns={"dashboard"}
              components={[<Link key={0} href={"https://capjs.js.org/guide/standalone/"} target={"_blank"} />]}
            />
          </NoMarginHelperText>
        </FormControl>
      </SettingForm>
      <SettingForm title={t("settings.capAssetServer")} lgWidth={5}>
        <FormControl>
          <DenseSelect
            value={values.captcha_cap_asset_server || "jsdelivr"}
            onChange={(e) =>
              setSettings({
                captcha_cap_asset_server: e.target.value,
              })
            }
          >
            <SquareMenuItem value="jsdelivr">
              <ListItemText
                slotProps={{
                  primary: { variant: "body2" },
                }}
              >
                {t("settings.capAssetServerJsdelivr")}
              </ListItemText>
            </SquareMenuItem>
            <SquareMenuItem value="unpkg">
              <ListItemText
                slotProps={{
                  primary: { variant: "body2" },
                }}
              >
                {t("settings.capAssetServerUnpkg")}
              </ListItemText>
            </SquareMenuItem>
            <SquareMenuItem value="instance">
              <ListItemText
                slotProps={{
                  primary: { variant: "body2" },
                }}
              >
                {t("settings.capAssetServerInstance")}
              </ListItemText>
            </SquareMenuItem>
          </DenseSelect>
          <NoMarginHelperText>
            <Trans
              i18nKey="settings.capAssetServerDes"
              ns={"dashboard"}
              components={[
                <Link
                  key={0}
                  href={"https://capjs.js.org/guide/standalone/options.html#asset-server"}
                  target={"_blank"}
                />,
              ]}
            />
          </NoMarginHelperText>
        </FormControl>
      </SettingForm>
    </Stack>
  );
};

export default CapCaptcha;
