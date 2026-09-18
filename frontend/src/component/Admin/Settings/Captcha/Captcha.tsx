import { Box, Collapse, FormControl, FormControlLabel, ListItemText, Stack, Switch, Typography } from "@mui/material";
import { useContext } from "react";
import { useTranslation } from "react-i18next";
import { CaptchaType } from "../../../../api/site.ts";
import { isTrueVal } from "../../../../session/utils.ts";
import { DenseSelect } from "../../../Common/StyledComponents.tsx";
import { SquareMenuItem } from "../../../FileManager/ContextMenu/ContextMenu.tsx";
import SettingForm from "../../../Pages/Setting/SettingForm.tsx";
import { NoMarginHelperText, SettingSection, SettingSectionContent } from "../Settings.tsx";
import { SettingContext } from "../SettingWrapper.tsx";
import CapCaptcha from "./CapCaptcha.tsx";
import GraphicCaptcha from "./GraphicCaptcha.tsx";
import ReCaptcha from "./ReCaptcha.tsx";
import TurnstileCaptcha from "./TurnstileCaptcha.tsx";

const Captcha = () => {
  const { t } = useTranslation("dashboard");
  const { formRef, setSettings, values } = useContext(SettingContext);

  return (
    <Box component={"form"} ref={formRef} onSubmit={(e) => e.preventDefault()}>
      <Stack spacing={5}>
        <SettingSection>
          <Typography variant="h6" gutterBottom>
            {t("nav.captcha")}
          </Typography>
          <SettingSectionContent>
            <SettingForm lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel
                  control={
                    <Switch
                      checked={isTrueVal(values.login_captcha)}
                      onChange={(e) =>
                        setSettings({
                          login_captcha: e.target.checked ? "1" : "0",
                        })
                      }
                    />
                  }
                  label={t("settings.captchaForLogin")}
                />
                <NoMarginHelperText>{t("settings.captchaForLoginDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel
                  control={
                    <Switch
                      checked={isTrueVal(values.reg_captcha)}
                      onChange={(e) =>
                        setSettings({
                          reg_captcha: e.target.checked ? "1" : "0",
                        })
                      }
                    />
                  }
                  label={t("settings.captchaForSignup")}
                />
                <NoMarginHelperText>{t("settings.captchaForSignupDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel
                  control={
                    <Switch
                      checked={isTrueVal(values.forget_captcha)}
                      onChange={(e) =>
                        setSettings({
                          forget_captcha: e.target.checked ? "1" : "0",
                        })
                      }
                    />
                  }
                  label={t("settings.captchaForReset")}
                />
                <NoMarginHelperText>{t("settings.captchaForResetDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
          </SettingSectionContent>
        </SettingSection>
        <SettingSection>
          <Typography variant="h6" gutterBottom>
            {t("settings.captchaType")}
          </Typography>
          <SettingSectionContent>
            <SettingForm title={t("settings.captchaType")} lgWidth={5}>
              <FormControl>
                <DenseSelect
                  onChange={(e) =>
                    setSettings({
                      captcha_type: e.target.value as string,
                    })
                  }
                  value={values.captcha_type}
                >
                  <SquareMenuItem value={CaptchaType.NORMAL}>
                    <ListItemText
                      slotProps={{
                        primary: { variant: "body2" },
                      }}
                    >
                      {t("settings.plainCaptcha")}
                    </ListItemText>
                  </SquareMenuItem>
                  <SquareMenuItem value={CaptchaType.RECAPTCHA}>
                    <ListItemText
                      slotProps={{
                        primary: { variant: "body2" },
                      }}
                    >
                      {t("settings.reCaptchaV2")}
                    </ListItemText>
                  </SquareMenuItem>
                  <SquareMenuItem value={CaptchaType.TURNSTILE}>
                    <ListItemText
                      slotProps={{
                        primary: { variant: "body2" },
                      }}
                    >
                      {t("settings.turnstile")}
                    </ListItemText>
                  </SquareMenuItem>
                  <SquareMenuItem value={CaptchaType.CAP}>
                    <ListItemText
                      slotProps={{
                        primary: { variant: "body2" },
                      }}
                    >
                      {t("settings.cap")}
                    </ListItemText>
                  </SquareMenuItem>
                </DenseSelect>
                <NoMarginHelperText>{t("settings.captchaTypeDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <Collapse in={values.captcha_type === CaptchaType.NORMAL} unmountOnExit>
              <GraphicCaptcha setSettings={setSettings} values={values} />
            </Collapse>
            <Collapse in={values.captcha_type === CaptchaType.RECAPTCHA} unmountOnExit>
              <ReCaptcha setSettings={setSettings} values={values} />
            </Collapse>
            <Collapse in={values.captcha_type === CaptchaType.TURNSTILE} unmountOnExit>
              <TurnstileCaptcha setSettings={setSettings} values={values} />
            </Collapse>
            <Collapse in={values.captcha_type === CaptchaType.CAP} unmountOnExit>
              <CapCaptcha setSettings={setSettings} values={values} />
            </Collapse>
          </SettingSectionContent>
        </SettingSection>
      </Stack>
    </Box>
  );
};

export default Captcha;
