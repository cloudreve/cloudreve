import { Collapse, FormControl, FormControlLabel, ListItemText, Stack, Switch } from "@mui/material";
import { useTranslation } from "react-i18next";
import { isTrueVal } from "../../../../session/utils.ts";
import { DenseFilledTextField, DenseSelect } from "../../../Common/StyledComponents.tsx";
import { SquareMenuItem } from "../../../FileManager/ContextMenu/ContextMenu.tsx";
import SettingForm from "../../../Pages/Setting/SettingForm.tsx";
import { NoMarginHelperText } from "../Settings.tsx";

export interface GraphicCaptchaProps {
  values: {
    [key: string]: string;
  };
  setSettings: (settings: { [key: string]: string }) => void;
}

const GraphicCaptcha = ({ values, setSettings }: GraphicCaptchaProps) => {
  const { t } = useTranslation("dashboard");
  return (
    <Stack spacing={3}>
      <SettingForm title={t("settings.captchaMode")} lgWidth={5}>
        <FormControl>
          <DenseSelect
            onChange={(e) =>
              setSettings({
                captcha_mode: e.target.value as string,
              })
            }
            value={values.captcha_mode}
          >
            {["captchaModeNumber", "captchaModeLetter", "captchaModeMath", "captchaModeNumberLetter"].map((k, i) => (
              <SquareMenuItem key={k} value={i.toString()}>
                <ListItemText
                  slotProps={{
                    primary: { variant: "body2" },
                  }}
                >
                  {t(`settings.${k}`)}
                </ListItemText>
              </SquareMenuItem>
            ))}
          </DenseSelect>
        </FormControl>
      </SettingForm>
      <Collapse in={values.captcha_mode !== "2"} unmountOnExit>
        <SettingForm title={t("settings.captchaLength")} lgWidth={5}>
          <FormControl>
            <DenseFilledTextField
              value={values.captcha_CaptchaLen}
              slotProps={{
                htmlInput: {
                  type: "number",
                  min: 1,
                  max: 10,
                },
              }}
              onChange={(e) =>
                setSettings({
                  captcha_CaptchaLen: e.target.value,
                })
              }
            />
            <NoMarginHelperText>{t("settings.captchaLengthDes")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>
      </Collapse>
      {[
        {
          name: "complexOfNoiseText",
          field: "captcha_ComplexOfNoiseText",
        },
        {
          name: "complexOfNoiseDot",
          field: "captcha_ComplexOfNoiseDot",
        },
        {
          name: "showHollowLine",
          field: "captcha_IsShowHollowLine",
        },
        {
          name: "showNoiseDot",
          field: "captcha_IsShowNoiseDot",
        },
        {
          name: "showNoiseText",
          field: "captcha_IsShowNoiseText",
        },
        {
          name: "showSlimeLine",
          field: "captcha_IsShowSlimeLine",
        },
        {
          name: "showSineLine",
          field: "captcha_IsShowSineLine",
        },
      ].map((v) => (
        <SettingForm lgWidth={5}>
          <FormControl fullWidth>
            <FormControlLabel
              control={
                <Switch
                  checked={isTrueVal(values[v.field])}
                  onChange={(e) =>
                    setSettings({
                      [v.field]: e.target.checked ? "1" : "0",
                    })
                  }
                />
              }
              label={t(`settings.${v.name}`)}
            />
          </FormControl>
        </SettingForm>
      ))}
    </Stack>
  );
};

export default GraphicCaptcha;
