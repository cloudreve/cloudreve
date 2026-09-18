import { Box, FormControl, FormControlLabel, Switch, Typography } from "@mui/material";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { GroupEnt } from "../../../../api/dashboard";
import { GroupPermission } from "../../../../api/user";
import Boolset from "../../../../util/boolset";
import SettingForm from "../../../Pages/Setting/SettingForm";
import { NoMarginHelperText, SettingSection, SettingSectionContent } from "../../Settings/Settings";
import { AnonymousGroupID } from "../GroupRow";
import { GroupSettingContext } from "./GroupSettingWrapper";

const ShareSection = () => {
  const { t } = useTranslation("dashboard");
  const { values, setGroup } = useContext(GroupSettingContext);

  const permission = useMemo(() => {
    return new Boolset(values.permissions ?? "");
  }, [values.permissions]);

  const onAllowCreateShareLinkChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setGroup((p: GroupEnt) => ({
        ...p,
        permissions: new Boolset(p.permissions).set(GroupPermission.share, e.target.checked).toString(),
      }));
    },
    [setGroup],
  );

  const onShareDownloadChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setGroup((p: GroupEnt) => ({
        ...p,
        permissions: new Boolset(p.permissions).set(GroupPermission.share_download, e.target.checked).toString(),
      }));
    },
    [setGroup],
  );

  return (
    <SettingSection>
      <Typography variant="h6" gutterBottom>
        {t("group.share")}
      </Typography>
      <SettingSectionContent>
        {values?.id != AnonymousGroupID && (
          <SettingForm lgWidth={5}>
            <FormControl fullWidth>
              <FormControlLabel
                control={
                  <Switch checked={permission.enabled(GroupPermission.share)} onChange={onAllowCreateShareLinkChange} />
                }
                label={t("group.allowCreateShareLink")}
              />
              <NoMarginHelperText>{t("group.allowCreateShareLinkDes")}</NoMarginHelperText>
            </FormControl>
          </SettingForm>
        )}
        <SettingForm lgWidth={5}>
          <FormControl fullWidth>
            <FormControlLabel
              control={<Switch checked={false} />}
              label={
                <Box sx={{ display: "flex", alignItems: "center" }}>
                  {t("group.shareFree")}
                </Box>
              }
            />
            <NoMarginHelperText>{t("group.shareFreeDes")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>
        <SettingForm lgWidth={5}>
          <FormControl fullWidth>
            <FormControlLabel
              control={
                <Switch checked={permission.enabled(GroupPermission.share_download)} onChange={onShareDownloadChange} />
              }
              label={t("group.allowDownloadShare")}
            />
            <NoMarginHelperText>{t("group.allowDownloadShareDes")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>
        {values?.id != AnonymousGroupID && (
          <SettingForm lgWidth={5}>
            <FormControl fullWidth>
              <FormControlLabel
                control={<Switch checked={false} />}
                label={
                  <Box sx={{ display: "flex", alignItems: "center" }}>
                    {t("group.esclateAnonymity")}
                  </Box>
                }
              />
              <NoMarginHelperText>{t("group.esclateAnonymityDes")}</NoMarginHelperText>
            </FormControl>
          </SettingForm>
        )}
      </SettingSectionContent>
    </SettingSection>
  );
};

export default ShareSection;
