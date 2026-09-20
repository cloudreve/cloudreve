import { ExpandMoreRounded } from "@mui/icons-material";
import { AccordionDetails, FormControl, FormControlLabel, ListItemText, Switch, Typography } from "@mui/material";
import { useContext } from "react";
import { Trans, useTranslation } from "react-i18next";
import { isTrueVal } from "../../../../session/utils.ts";
import { Code } from "../../../Common/Code.tsx";
import { DenseFilledTextField, DenseSelect } from "../../../Common/StyledComponents.tsx";
import { SquareMenuItem } from "../../../FileManager/ContextMenu/ContextMenu.tsx";
import { NoMarginHelperText, SettingSectionContent } from "../Settings.tsx";
import { SettingContext } from "../SettingWrapper.tsx";
import { AccordionSummary, StyledAccordion } from "./SSOSettings.tsx";

const SmsGatewaySettings = () => {
  const { t } = useTranslation("dashboard");
  const { setSettings, values } = useContext(SettingContext);

  const enabled = isTrueVal(values.sms_enabled);

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
                  sms_enabled: e.target.checked ? "1" : "0",
                })
              }
              onClick={(e) => e.stopPropagation()}
            />
          }
          label={t("settings.smsSignIn")}
        />
      </AccordionSummary>
      <AccordionDetails sx={{ display: "block" }}>
        <SettingSectionContent>
          <FormControl fullWidth>
            <DenseFilledTextField
              label={t("settings.smsEndpoint")}
              value={values.sms_endpoint ?? ""}
              onChange={(e) => setSettings({ sms_endpoint: e.target.value })}
              required={enabled}
              placeholder="https://sms.example.com/send?phone={phone}&code={code}"
            />
            <NoMarginHelperText>
              <Trans i18nKey="settings.smsEndpointDes" ns="dashboard" components={[<Code key="0" />]} />
            </NoMarginHelperText>
          </FormControl>
          <FormControl fullWidth>
            <DenseSelect
              value={values.sms_method ?? "POST"}
              onChange={(e) => setSettings({ sms_method: e.target.value as string })}
            >
              {["POST", "GET"].map((m) => (
                <SquareMenuItem key={m} value={m}>
                  <ListItemText slotProps={{ primary: { variant: "body2" } }}>{m}</ListItemText>
                </SquareMenuItem>
              ))}
            </DenseSelect>
            <NoMarginHelperText>{t("settings.smsMethodDes")}</NoMarginHelperText>
          </FormControl>
          <FormControl fullWidth>
            <DenseFilledTextField
              label={t("settings.smsHeaders")}
              value={values.sms_headers ?? ""}
              onChange={(e) => setSettings({ sms_headers: e.target.value })}
              multiline
              minRows={2}
              placeholder={"Authorization: Bearer <token>"}
            />
            <NoMarginHelperText>{t("settings.smsHeadersDes")}</NoMarginHelperText>
          </FormControl>
          {(values.sms_method ?? "POST") !== "GET" && (
            <FormControl fullWidth>
              <DenseFilledTextField
                label={t("settings.smsBodyTemplate")}
                value={values.sms_body_tpl ?? ""}
                onChange={(e) => setSettings({ sms_body_tpl: e.target.value })}
                multiline
                minRows={2}
                placeholder={'{"phone":"{phone}","code":"{code}"}'}
              />
              <NoMarginHelperText>
                <Trans i18nKey="settings.smsBodyTemplateDes" ns="dashboard" components={[<Code key="0" />]} />
              </NoMarginHelperText>
            </FormControl>
          )}
          <FormControl fullWidth>
            <FormControlLabel
              control={
                <Switch
                  checked={isTrueVal(values.sms_register_enabled ?? "1")}
                  onChange={(e) =>
                    setSettings({
                      sms_register_enabled: e.target.checked ? "1" : "0",
                    })
                  }
                />
              }
              label={<Typography variant="body2">{t("settings.smsRegisterEnabled")}</Typography>}
            />
            <NoMarginHelperText>{t("settings.smsRegisterEnabledDes")}</NoMarginHelperText>
          </FormControl>
        </SettingSectionContent>
      </AccordionDetails>
    </StyledAccordion>
  );
};

export default SmsGatewaySettings;
