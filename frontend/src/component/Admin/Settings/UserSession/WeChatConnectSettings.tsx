import { ExpandMoreRounded } from "@mui/icons-material";
import { AccordionDetails, FormControl, FormControlLabel, Switch, Typography } from "@mui/material";
import { useContext, useMemo } from "react";
import { Trans, useTranslation } from "react-i18next";
import { isTrueVal } from "../../../../session/utils.ts";
import { Code } from "../../../Common/Code.tsx";
import { DenseFilledTextField } from "../../../Common/StyledComponents.tsx";
import { NoMarginHelperText, SettingSectionContent } from "../Settings.tsx";
import { SettingContext } from "../SettingWrapper.tsx";
import { AccordionSummary, StyledAccordion } from "./SSOSettings.tsx";

const WeChatConnectSettings = () => {
  const { t } = useTranslation("dashboard");
  const { setSettings, values } = useContext(SettingContext);

  const callbackURL = useMemo(() => {
    const primary = (values.siteURL ?? "").split(",")[0]?.trim().replace(/\/+$/, "");
    return primary ? `${primary}/api/v4/session/wechat/callback` : "";
  }, [values.siteURL]);

  const enabled = isTrueVal(values.wechat_connect_enabled);

  return (
    <StyledAccordion disableGutters>
      <AccordionSummary expandIcon={<ExpandMoreRounded />}>
        <FormControlLabel
          control={
            <Switch
              size="small"
              checked={enabled}
              onChange={(e) =>
                setSettings({
                  wechat_connect_enabled: e.target.checked ? "1" : "0",
                })
              }
              onClick={(e) => e.stopPropagation()}
            />
          }
          label={t("settings.wechatConnect")}
        />
      </AccordionSummary>
      <AccordionDetails sx={{ display: "block" }}>
        <SettingSectionContent>
          <FormControl fullWidth>
            <DenseFilledTextField
              label={t("settings.wechatAppID")}
              value={values.wechat_connect_app_id}
              onChange={(e) => setSettings({ wechat_connect_app_id: e.target.value })}
              required={enabled}
            />
            <NoMarginHelperText>
              <Trans i18nKey="settings.wechatAppIDDes" ns="dashboard" components={[<Code key="0" />]} />
            </NoMarginHelperText>
          </FormControl>
          <FormControl fullWidth>
            <DenseFilledTextField
              label={t("settings.wechatAppSecret")}
              value={values.wechat_connect_app_secret ?? ""}
              onChange={(e) => setSettings({ wechat_connect_app_secret: e.target.value })}
              type="password"
              placeholder={t("oauth.secretRedactedPlaceholder")}
            />
            <NoMarginHelperText>{t("oauth.clientSecretDesExisting")}</NoMarginHelperText>
          </FormControl>
          {callbackURL && (
            <FormControl fullWidth>
              <DenseFilledTextField
                label={t("settings.wechatCallbackUrl")}
                value={callbackURL}
                slotProps={{ input: { readOnly: true } }}
              />
              <NoMarginHelperText>
                <Trans
                  i18nKey="settings.wechatCallbackUrlDes"
                  ns="dashboard"
                  values={{ url: callbackURL }}
                  components={[<Code key="0" />]}
                />
              </NoMarginHelperText>
            </FormControl>
          )}
          <FormControl fullWidth>
            <FormControlLabel
              control={
                <Switch
                  checked={isTrueVal(values.wechat_connect_register_enabled)}
                  onChange={(e) =>
                    setSettings({
                      wechat_connect_register_enabled: e.target.checked ? "1" : "0",
                    })
                  }
                />
              }
              label={<Typography variant="body2">{t("settings.wechatRegisterEnabled")}</Typography>}
            />
            <NoMarginHelperText>{t("settings.wechatRegisterEnabledDes")}</NoMarginHelperText>
          </FormControl>
        </SettingSectionContent>
      </AccordionDetails>
    </StyledAccordion>
  );
};

export default WeChatConnectSettings;
