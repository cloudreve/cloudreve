import { ExpandMoreRounded } from "@mui/icons-material";
import { Accordion, AccordionDetails, FormControl, FormControlLabel, styled, Switch, Typography } from "@mui/material";
import MuiAccordionSummary, { AccordionSummaryProps } from "@mui/material/AccordionSummary";
import { useContext, useMemo } from "react";
import { Trans, useTranslation } from "react-i18next";
import { isTrueVal } from "../../../../session/utils.ts";
import { Code } from "../../../Common/Code.tsx";
import { DenseFilledTextField } from "../../../Common/StyledComponents.tsx";
import { NoMarginHelperText, SettingSectionContent } from "../Settings.tsx";
import { SettingContext } from "../SettingWrapper.tsx";

export const AccordionSummary = styled((props: AccordionSummaryProps) => <MuiAccordionSummary {...props} />)(
  ({ theme }) => ({
    fontSize: theme.typography.body2.fontSize,
    paddingLeft: theme.spacing(4),
    "& .MuiFormControlLabel-label": {
      fontSize: theme.typography.body2.fontSize,
    },
    "& .MuiCheckbox-root": {
      marginRight: theme.spacing(2),
    },
  }),
);

export const StyledAccordion = styled(Accordion)(({ theme }) => ({
  boxShadow: "none",
  border: `1px solid ${theme.palette.divider}`,
  "&::before": {
    display: "none",
  },
}));

const SSOSettings = () => {
  const { t } = useTranslation("dashboard");
  const { setSettings, values } = useContext(SettingContext);

  const callbackURL = useMemo(() => {
    const primary = (values.siteURL ?? "").split(",")[0]?.trim().replace(/\/+$/, "");
    return primary ? `${primary}/api/v4/session/sso/callback` : "";
  }, [values.siteURL]);

  const enabled = isTrueVal(values.sso_enabled);

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
                  sso_enabled: e.target.checked ? "1" : "0",
                })
              }
              onClick={(e) => e.stopPropagation()}
            />
          }
          label={t("settings.oidc")}
        />
      </AccordionSummary>
      <AccordionDetails sx={{ display: "block" }}>
        <SettingSectionContent>
          <FormControl fullWidth>
            <DenseFilledTextField
              label={t("settings.displayName")}
              value={values.sso_display_name}
              onChange={(e) => setSettings({ sso_display_name: e.target.value })}
              required
            />
            <NoMarginHelperText>{t("settings.displayNameDes")}</NoMarginHelperText>
          </FormControl>
          <FormControl fullWidth>
            <DenseFilledTextField
              label={t("settings.ssoIssuer")}
              value={values.sso_issuer}
              onChange={(e) => setSettings({ sso_issuer: e.target.value })}
              placeholder="https://keycloak.example.com/realms/master"
              required={enabled}
            />
            <NoMarginHelperText>
              <Trans i18nKey="settings.ssoIssuerDes" ns="dashboard" components={[<Code key="0" />, <Code key="1" />]} />
            </NoMarginHelperText>
          </FormControl>
          <FormControl fullWidth>
            <DenseFilledTextField
              label={t("settings.clientID")}
              value={values.sso_client_id}
              onChange={(e) => setSettings({ sso_client_id: e.target.value })}
              required={enabled}
            />
            <NoMarginHelperText>{t("settings.clientIDDes")}</NoMarginHelperText>
          </FormControl>
          <FormControl fullWidth>
            <DenseFilledTextField
              label={t("settings.clientSecret")}
              value={values.sso_client_secret ?? ""}
              onChange={(e) => setSettings({ sso_client_secret: e.target.value })}
              type="password"
              placeholder={t("oauth.secretRedactedPlaceholder")}
            />
            <NoMarginHelperText>{t("oauth.clientSecretDesExisting")}</NoMarginHelperText>
          </FormControl>
          <FormControl fullWidth>
            <DenseFilledTextField
              label={t("settings.scope")}
              value={values.sso_scopes}
              onChange={(e) => setSettings({ sso_scopes: e.target.value })}
              placeholder="groups, roles"
            />
            <NoMarginHelperText>
              <Trans i18nKey="settings.scopeDes" ns="dashboard" components={[<Code key="0" />]} />
            </NoMarginHelperText>
          </FormControl>
          {callbackURL && (
            <FormControl fullWidth>
              <DenseFilledTextField
                label={t("settings.ssoCallbackUrl")}
                value={callbackURL}
                slotProps={{ input: { readOnly: true } }}
              />
              <NoMarginHelperText>
                <Trans i18nKey="settings.ssoCallbackUrlDes" ns="dashboard" values={{ url: callbackURL }} components={[<Code key="0" />]} />
              </NoMarginHelperText>
            </FormControl>
          )}
          <FormControl fullWidth>
            <FormControlLabel
              control={
                <Switch
                  checked={isTrueVal(values.sso_register_enabled)}
                  onChange={(e) =>
                    setSettings({
                      sso_register_enabled: e.target.checked ? "1" : "0",
                    })
                  }
                />
              }
              label={<Typography variant="body2">{t("settings.ssoRegisterEnabled")}</Typography>}
            />
            <NoMarginHelperText>{t("settings.ssoRegisterEnabledDes")}</NoMarginHelperText>
          </FormControl>
          <FormControl fullWidth>
            <FormControlLabel
              control={
                <Switch
                  checked={isTrueVal(values.sso_auto_redirect)}
                  onChange={(e) =>
                    setSettings({
                      sso_auto_redirect: e.target.checked ? "1" : "0",
                    })
                  }
                />
              }
              label={<Typography variant="body2">{t("settings.ssoAutoRedirect")}</Typography>}
            />
            <NoMarginHelperText>
              <Trans
                i18nKey="settings.ssoAutoRedirectDes"
                ns="dashboard"
                components={[<Code key="0" />]}
              />
            </NoMarginHelperText>
          </FormControl>
        </SettingSectionContent>
      </AccordionDetails>
    </StyledAccordion>
  );
};

export default SSOSettings;
