import { Box, FormControl, FormControlLabel, IconButton, Switch, Typography } from "@mui/material";
import { useCallback, useContext } from "react";
import { Trans, useTranslation } from "react-i18next";
import { StoragePolicy } from "../../../../../api/dashboard";
import QuestionCircle from "../../../../Icons/QuestionCircle";
import SettingForm from "../../../../Pages/Setting/SettingForm";
import { NoMarginHelperText, SettingSection, SettingSectionContent } from "../../../Settings/Settings";
import { StoragePolicySettingContext } from "../StoragePolicySettingWrapper";

const EncryptionSection = () => {
  const { t } = useTranslation("dashboard");
  const { values, setPolicy } = useContext(StoragePolicySettingContext);

  const onEncryptionChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setPolicy((p: StoragePolicy) => ({
        ...p,
        settings: { ...p.settings, encryption: e.target.checked ? true : undefined },
      }));
    },
    [setPolicy],
  );

  return (
    <SettingSection>
      <Box display="flex" alignItems="center" gap={1}>
        <Typography variant="h6">{t("policy.fileEncryption")}</Typography>
        <IconButton onClick={() => window.open("https://docs.cloudreve.org/usage/file-encryption", "_blank")}>
          <QuestionCircle />
        </IconButton>
      </Box>
      <SettingSectionContent>
        <SettingForm lgWidth={5}>
          <FormControl fullWidth>
            <FormControlLabel
              control={<Switch checked={values.settings?.encryption ?? false} onChange={onEncryptionChange} />}
              label={t("policy.enableFileEncryption")}
            />
            <NoMarginHelperText>
              <Trans i18nKey="policy.enableFileEncryptionDes" ns="dashboard" />
            </NoMarginHelperText>
          </FormControl>
        </SettingForm>
      </SettingSectionContent>
    </SettingSection>
  );
};

export default EncryptionSection;
