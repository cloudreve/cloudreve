import { Collapse, FormControl, FormControlLabel, Link, ListItemText, Switch, Typography } from "@mui/material";
import { useContext } from "react";
import { Trans, useTranslation } from "react-i18next";
import { isTrueVal } from "../../../../session/utils.ts";
import { Code } from "../../../Common/Code.tsx";
import { DenseFilledTextField, DenseSelect } from "../../../Common/StyledComponents.tsx";
import { SquareMenuItem } from "../../../FileManager/ContextMenu/ContextMenu.tsx";
import SettingForm from "../../../Pages/Setting/SettingForm.tsx";
import { NoMarginHelperText, SettingSection, SettingSectionContent } from "../../Settings/Settings.tsx";
import { SettingContext } from "../../Settings/SettingWrapper.tsx";

const FileEncryptionSection = () => {
  const { t } = useTranslation("dashboard");
  const { setSettings, values } = useContext(SettingContext);

  return (
    <SettingSection>
      <Typography variant="h6" gutterBottom>
        {t("policy.fileEncryption")}
      </Typography>
      <SettingSectionContent>
        <SettingForm title={t("settings.masterEncryptionKeyVault")} lgWidth={5}>
          <FormControl>
            <DenseSelect
              onChange={(e) =>
                setSettings({
                  encrypt_master_key_vault: e.target.value as string,
                })
              }
              value={values.encrypt_master_key_vault}
            >
              <SquareMenuItem value={"setting"}>
                <ListItemText
                  slotProps={{
                    primary: { variant: "body2" },
                  }}
                >
                  {t("settings.masterEncryptionKeyVaultSetting")}
                </ListItemText>
              </SquareMenuItem>
              <SquareMenuItem value={"file"}>
                <ListItemText
                  slotProps={{
                    primary: { variant: "body2" },
                  }}
                >
                  {t("settings.masterEncryptionKeyVaultFile")}
                </ListItemText>
              </SquareMenuItem>
              <SquareMenuItem value={"env"}>
                <ListItemText
                  slotProps={{
                    primary: { variant: "body2" },
                  }}
                >
                  {<Trans i18nKey="settings.masterEncryptionKeyVaultEnv" ns="dashboard" components={[<Code />]} />}
                </ListItemText>
              </SquareMenuItem>
            </DenseSelect>
            <NoMarginHelperText>
              <Trans
                i18nKey="settings.masterEncryptionKeyVaultDes"
                ns="dashboard"
                components={[
                  <Link
                    href="https://docs.cloudreve.org/usage/file-encryption#rotate-master-encryption-key"
                    target="_blank"
                  />,
                ]}
              />
            </NoMarginHelperText>
          </FormControl>
        </SettingForm>
        <Collapse in={values.encrypt_master_key_vault === "file"} unmountOnExit>
          <SettingForm title={t("settings.masterEncryptionKeyVaultFilePath")} lgWidth={5}>
            <FormControl fullWidth>
              <DenseFilledTextField
                required
                value={values.encrypt_master_key_file}
                onChange={(e) => setSettings({ encrypt_master_key_file: e.target.value })}
              />
            </FormControl>
          </SettingForm>
        </Collapse>
        <SettingForm lgWidth={5}>
          <FormControl>
            <FormControlLabel
              control={
                <Switch
                  checked={isTrueVal(values.show_encryption_status)}
                  onChange={(e) => setSettings({ show_encryption_status: e.target.checked ? "1" : "0" })}
                />
              }
              label={t("settings.showEncryptionStatus")}
            />
            <NoMarginHelperText>
              <Trans i18nKey="settings.showEncryptionStatusDes" ns="dashboard" />
            </NoMarginHelperText>
          </FormControl>
        </SettingForm>
      </SettingSectionContent>
    </SettingSection>
  );
};

export default FileEncryptionSection;
