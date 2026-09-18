import { Box, FormControl, Link, Stack, Typography } from "@mui/material";
import { useContext } from "react";
import { Trans, useTranslation } from "react-i18next";
import { DenseFilledTextField, SecondaryButton } from "../../../Common/StyledComponents";
import ArrowSync from "../../../Icons/ArrowSync";
import SettingForm from "../../../Pages/Setting/SettingForm";
import { NoMarginHelperText, SettingSection, SettingSectionContent } from "../Settings";
import { SettingContext } from "../SettingWrapper";

const ServerSetting = () => {
  const { t } = useTranslation("dashboard");
  const { formRef, setSettings, values } = useContext(SettingContext);

  const rotateSecretKey = () => {
    setSettings({ secret_key: "[Placeholder]" });
  };

  return (
    <Box component={"form"} ref={formRef} onSubmit={(e) => e.preventDefault()}>
      <Stack spacing={5}>
        <SettingSection>
          <Typography variant="h6" gutterBottom>
            {t("settings.server")}
          </Typography>
          <SettingSectionContent>
            <SettingForm title={t("settings.tempPath")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  required
                  value={values.temp_path ?? ""}
                  onChange={(e) => setSettings({ temp_path: e.target.value })}
                />
                <NoMarginHelperText>{t("settings.tempPathDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("settings.siteID")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  required
                  value={values.siteID ?? ""}
                  onChange={(e) => setSettings({ siteID: e.target.value })}
                />
                <NoMarginHelperText>{t("settings.siteIDDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("settings.siteSecretKey")} lgWidth={5}>
              <SecondaryButton onClick={rotateSecretKey} startIcon={<ArrowSync />} variant="contained">
                {t("settings.rotateSecretKey")}
              </SecondaryButton>
              <NoMarginHelperText>{t("settings.siteSecretKeyDes")}</NoMarginHelperText>
            </SettingForm>
            <SettingForm title={t("settings.hashidSalt")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  required
                  value={values.hash_id_salt ?? ""}
                  onChange={(e) => setSettings({ hash_id_salt: e.target.value })}
                />
                <NoMarginHelperText>{t("settings.hashidSaltDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("settings.accessTokenTTL")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  required
                  slotProps={{
                    input: {
                      type: "number",
                      inputProps: {
                        min: 100,
                      },
                    },
                  }}
                  value={values.access_token_ttl ?? ""}
                  onChange={(e) => setSettings({ access_token_ttl: e.target.value })}
                />
                <NoMarginHelperText>{t("settings.accessTokenTTLDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("settings.refreshTokenTTL")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  required
                  slotProps={{
                    input: {
                      type: "number",
                      inputProps: {
                        min: parseInt(values.access_token_ttl ?? "1000") + 100,
                      },
                    },
                  }}
                  value={values.refresh_token_ttl ?? ""}
                  onChange={(e) => setSettings({ refresh_token_ttl: e.target.value })}
                />
                <NoMarginHelperText>{t("settings.refreshTokenTTLDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("settings.cronGarbageCollect")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  required
                  value={values.cron_garbage_collect ?? ""}
                  onChange={(e) => setSettings({ cron_garbage_collect: e.target.value })}
                />
                <NoMarginHelperText>
                  <Trans
                    i18nKey="settings.cronDes"
                    values={{
                      des: t("settings.cronGarbageCollectDes"),
                    }}
                    ns={"dashboard"}
                    components={[<Link href="https://crontab.guru/" target="_blank" rel="noopener noreferrer" />]}
                  />
                </NoMarginHelperText>
              </FormControl>
            </SettingForm>
          </SettingSectionContent>
        </SettingSection>
      </Stack>
    </Box>
  );
};

export default ServerSetting;
