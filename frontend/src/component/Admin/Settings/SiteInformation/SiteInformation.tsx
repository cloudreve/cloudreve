import { Box, FormControl, FormControlLabel, Stack, Switch, Typography } from "@mui/material";
import Grid from "@mui/material/Grid";
import { useContext } from "react";
import { useTranslation } from "react-i18next";
import { isTrueVal } from "../../../../session/utils.ts";
import { DenseFilledTextField } from "../../../Common/StyledComponents.tsx";
import SettingForm from "../../../Pages/Setting/SettingForm.tsx";
import { NoMarginHelperText, SettingSection, SettingSectionContent, StyledInputAdornment } from "../Settings.tsx";
import { SettingContext } from "../SettingWrapper.tsx";
import GeneralImagePreview from "./GeneralImagePreview.tsx";
import LogoPreview from "./LogoPreview.tsx";
import SiteURLInput from "./SiteURLInput.tsx";

const SiteInformation = () => {
  const { t } = useTranslation("dashboard");
  const { formRef, setSettings, values } = useContext(SettingContext);

  return (
    <Box component={"form"} ref={formRef} onSubmit={(e) => e.preventDefault()}>
      <Stack spacing={5}>
        <SettingSection>
          <Typography variant="h6" gutterBottom>
            {t("settings.basicInformation")}
          </Typography>
          <SettingSectionContent>
            <SettingForm title={t("settings.mainTitle")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  fullWidth
                  onChange={(e) => setSettings({ siteName: e.target.value })}
                  value={values.siteName}
                  required
                  inputProps={{ maxLength: 255 }}
                />
                <NoMarginHelperText>{t("settings.mainTitleDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("settings.siteDescription")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  fullWidth
                  onChange={(e) => setSettings({ siteDes: e.target.value })}
                  value={values.siteDes}
                  multiline
                  rows={4}
                />
                <NoMarginHelperText>{t("settings.siteDescriptionDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("settings.siteURL")} lgWidth={5}>
              <FormControl fullWidth>
                <SiteURLInput urls={values.siteURL} onChange={(v) => setSettings({ siteURL: v })} />
              </FormControl>
            </SettingForm>
            <SettingForm title={t("settings.customFooterHTML")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  fullWidth
                  onChange={(e) => setSettings({ siteScript: e.target.value })}
                  value={values.siteScript}
                  multiline
                  rows={4}
                />
                <NoMarginHelperText>{t("settings.customFooterHTMLDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("settings.announcement")} lgWidth={5} pro>
              <FormControl fullWidth>
                <DenseFilledTextField inputProps={{ readOnly: true }} fullWidth multiline rows={4} />
                <NoMarginHelperText>{t("settings.announcementDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("settings.tosUrl")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  fullWidth
                  onChange={(e) => setSettings({ tos_url: e.target.value })}
                  value={values.tos_url}
                />
                <NoMarginHelperText>{t("settings.tosUrlDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("settings.privacyUrl")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  fullWidth
                  onChange={(e) => setSettings({ privacy_policy_url: e.target.value })}
                  value={values.privacy_policy_url}
                />
                <NoMarginHelperText>{t("settings.privacyUrlDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
          </SettingSectionContent>
        </SettingSection>
        <SettingSection>
          <Typography variant="h6" gutterBottom>
            {t("settings.branding")}
          </Typography>
          <SettingSectionContent>
            <SettingForm
              title={t("settings.logo")}
              lgWidth={5}
              spacing={3}
              secondary={
                <Grid item md={7} xs={12}>
                  <LogoPreview logoDark={values.site_logo} logoLight={values.site_logo_light} />
                </Grid>
              }
            >
              <FormControl fullWidth>
                <DenseFilledTextField
                  fullWidth
                  onChange={(e) => setSettings({ site_logo: e.target.value })}
                  value={values.site_logo}
                  required
                  InputProps={{
                    startAdornment: (
                      <StyledInputAdornment disableTypography position="start">
                        {t("settings.light")}
                      </StyledInputAdornment>
                    ),
                  }}
                />
                <DenseFilledTextField
                  sx={{ mt: 1 }}
                  fullWidth
                  onChange={(e) => setSettings({ site_logo_light: e.target.value })}
                  value={values.site_logo_light}
                  required
                  InputProps={{
                    startAdornment: (
                      <StyledInputAdornment disableTypography position="start">
                        {t("settings.dark")}
                      </StyledInputAdornment>
                    ),
                  }}
                />
                <NoMarginHelperText>{t("settings.logoDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm
              title={t("settings.smallIcon")}
              lgWidth={5}
              spacing={3}
              secondary={
                <Grid item md={7} xs={12}>
                  <Box sx={{ maxWidth: 160 }}>
                    <GeneralImagePreview src={values.pwa_small_icon} debounce={250} />
                  </Box>
                </Grid>
              }
            >
              <FormControl fullWidth>
                <DenseFilledTextField
                  fullWidth
                  onChange={(e) => setSettings({ pwa_small_icon: e.target.value })}
                  value={values.pwa_small_icon}
                  required
                />
                <NoMarginHelperText>{t("settings.smallIconDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm
              title={t("settings.mediumIcon")}
              lgWidth={5}
              spacing={3}
              secondary={
                <Grid item md={7} xs={12}>
                  <Box sx={{ maxWidth: 160 }}>
                    <GeneralImagePreview src={values.pwa_medium_icon} debounce={250} />
                  </Box>
                </Grid>
              }
            >
              <FormControl fullWidth>
                <DenseFilledTextField
                  fullWidth
                  onChange={(e) => setSettings({ pwa_medium_icon: e.target.value })}
                  value={values.pwa_medium_icon}
                  required
                />
                <NoMarginHelperText>{t("settings.mediumIconDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm
              title={t("settings.largeIcon")}
              lgWidth={5}
              spacing={3}
              secondary={
                <Grid item md={7} xs={12}>
                  <Box sx={{ maxWidth: 160 }}>
                    <GeneralImagePreview src={values.pwa_large_icon} debounce={250} />
                  </Box>
                </Grid>
              }
            >
              <FormControl fullWidth>
                <DenseFilledTextField
                  fullWidth
                  onChange={(e) => setSettings({ pwa_large_icon: e.target.value })}
                  value={values.pwa_large_icon}
                  required
                />
                <NoMarginHelperText>{t("settings.largeIconDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
          </SettingSectionContent>
        </SettingSection>
        <SettingSection>
          <Typography variant="h6" gutterBottom>
            {t("vas.mobileApp")}
          </Typography>
          <SettingSectionContent>
            <SettingForm lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel
                  control={
                    <Switch
                      checked={isTrueVal(values.show_app_promotion)}
                      onChange={(e) =>
                        setSettings({
                          show_app_promotion: e.target.checked ? "1" : "0",
                        })
                      }
                    />
                  }
                  label={t("vas.showAppPromotion")}
                />
                <NoMarginHelperText>{t("vas.showAppPromotionDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("vas.appFeedback")} lgWidth={5} pro>
              <FormControl fullWidth>
                <DenseFilledTextField
                  fullWidth
                  slotProps={{
                    input: {
                      readOnly: true,
                    },
                  }}
                />
                <NoMarginHelperText>{t("vas.appLinkDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("vas.appForum")} lgWidth={5} pro>
              <FormControl fullWidth>
                <DenseFilledTextField fullWidth slotProps={{ input: { readOnly: true } }} />
                <NoMarginHelperText>{t("vas.appLinkDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
          </SettingSectionContent>
        </SettingSection>
        <SettingSection>
          <Typography variant="h6" gutterBottom>
            {t("vas.desktopApp")}
          </Typography>
          <SettingSectionContent>
            <SettingForm lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel
                  control={
                    <Switch
                      checked={isTrueVal(values.show_desktop_app_promotion)}
                      onChange={(e) =>
                        setSettings({
                          show_desktop_app_promotion: e.target.checked ? "1" : "0",
                        })
                      }
                    />
                  }
                  label={t("vas.showDesktopAppPromotion")}
                />
                <NoMarginHelperText>{t("vas.showDesktopAppPromotionDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
          </SettingSectionContent>
        </SettingSection>
      </Stack>
    </Box>
  );
};

export default SiteInformation;
