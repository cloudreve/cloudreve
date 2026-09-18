import { FormControl, FormControlLabel, Switch, Typography } from "@mui/material";
import { useContext } from "react";
import { useTranslation } from "react-i18next";
import { isTrueVal } from "../../../../session/utils.ts";
import { DenseFilledTextField } from "../../../Common/StyledComponents.tsx";
import SettingForm from "../../../Pages/Setting/SettingForm.tsx";
import { NoMarginHelperText, SettingSection, SettingSectionContent } from "../../Settings/Settings.tsx";
import { SettingContext } from "../../Settings/SettingWrapper.tsx";

const FileEventPushSection = () => {
  const { t } = useTranslation("dashboard");
  const { setSettings, values } = useContext(SettingContext);

  return (
    <SettingSection>
      <Typography variant="h6" gutterBottom>
        {t("settings.fileEventPush")}
      </Typography>
      <SettingSectionContent>
        <SettingForm lgWidth={5}>
          <FormControl fullWidth>
            <FormControlLabel
              control={
                <Switch
                  checked={isTrueVal(values.fs_event_push_enabled)}
                  onChange={(e) =>
                    setSettings({
                      fs_event_push_enabled: e.target.checked ? "1" : "0",
                    })
                  }
                />
              }
              label={t("settings.enableFileEventPush")}
            />
            <NoMarginHelperText>{t("settings.enableFileEventPushDes")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>
        <SettingForm title={t("settings.fileEventPushMaxAge")} lgWidth={5}>
          <DenseFilledTextField
            type="number"
            inputProps={{ min: 0, step: 1 }}
            value={values.fs_event_push_max_age}
            onChange={(e) =>
              setSettings({
                fs_event_push_max_age: e.target.value,
              })
            }
            required
          />
          <NoMarginHelperText>{t("settings.fileEventPushMaxAgeDes")}</NoMarginHelperText>
        </SettingForm>
        <SettingForm title={t("settings.fileEventPushDebounce")} lgWidth={5}>
          <DenseFilledTextField
            type="number"
            inputProps={{ min: 0, step: 1 }}
            value={values.fs_event_push_debounce}
            onChange={(e) =>
              setSettings({
                fs_event_push_debounce: e.target.value,
              })
            }
            required
          />
          <NoMarginHelperText>{t("settings.fileEventPushDebounceDes")}</NoMarginHelperText>
        </SettingForm>
      </SettingSectionContent>
    </SettingSection>
  );
};

export default FileEventPushSection;
