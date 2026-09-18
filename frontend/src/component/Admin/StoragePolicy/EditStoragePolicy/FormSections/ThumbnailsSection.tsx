import { Checkbox, FormControl, FormControlLabel, Link, Switch, Typography } from "@mui/material";
import { useCallback, useContext, useMemo } from "react";
import { Trans, useTranslation } from "react-i18next";
import { Link as RouterLink } from "react-router-dom";
import { StoragePolicy } from "../../../../../api/dashboard";
import { PolicyType } from "../../../../../api/explorer";
import SizeInput from "../../../../Common/SizeInput";
import { DenseFilledTextField } from "../../../../Common/StyledComponents";
import SettingForm from "../../../../Pages/Setting/SettingForm";
import { NoMarginHelperText, SettingSection, SettingSectionContent } from "../../../Settings/Settings";
import { PolicyPropsMap } from "../../StoragePolicySetting";
import { StoragePolicySettingContext } from "../StoragePolicySettingWrapper";

const ThumbnailsSection = () => {
  const { t } = useTranslation("dashboard");
  const { values, setPolicy } = useContext(StoragePolicySettingContext);

  const policyProps = useMemo(() => {
    return PolicyPropsMap[values.type];
  }, [values.type]);

  const noNativeThumbnail = useMemo(() => {
    return values.type === PolicyType.local || values.type === PolicyType.s3;
  }, [values.type]);

  const onNativeThumbnailChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setPolicy((p: StoragePolicy) => ({
        ...p,
        settings: {
          ...p.settings,
          thumb_exts: e.target.value === "" ? undefined : e.target.value.split(",").map((ext) => ext.trim()),
        },
      }));
    },
    [setPolicy],
  );

  const onNativeThumbnailSupportAllExtsChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setPolicy((p: StoragePolicy) => ({
        ...p,
        settings: { ...p.settings, thumb_support_all_exts: e.target.checked ? true : undefined },
      }));
    },
    [setPolicy],
  );

  const onNativeThumbnailMaxSizeChange = useCallback(
    (size: number) => {
      setPolicy((p: StoragePolicy) => ({
        ...p,
        settings: { ...p.settings, thumb_max_size: size === 0 ? undefined : size },
      }));
    },
    [setPolicy],
  );

  const onThumbProxyChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setPolicy((p: StoragePolicy) => ({
        ...p,
        settings: { ...p.settings, thumb_generator_proxy: e.target.checked ? true : undefined },
      }));
    },
    [setPolicy],
  );

  if (values.type === PolicyType.local) {
    return null;
  }

  return (
    <SettingSection>
      <Typography variant="h6" gutterBottom>
        {t("settings.thumbnails")}
      </Typography>
      <SettingSectionContent>
        {!noNativeThumbnail && (
          <>
            <SettingForm title={t("policy.nativeThumbNails")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  multiline
                  fullWidth
                  disabled={values.settings?.thumb_support_all_exts}
                  maxRows={4}
                  value={values.settings?.thumb_exts?.join(",")}
                  onChange={onNativeThumbnailChange}
                />
                <FormControlLabel
                  slotProps={{
                    typography: {
                      variant: "body2",
                    },
                  }}
                  control={
                    <Checkbox
                      size={"small"}
                      checked={values.settings?.thumb_support_all_exts ?? false}
                      onChange={onNativeThumbnailSupportAllExtsChange}
                    />
                  }
                  label={t("policy.nativeThumbNailsSupportAllExts")}
                />
                <NoMarginHelperText>
                  {t("policy.nativeThumbNailsGeneralDes")}
                  {policyProps.nativeThumbDes && (
                    <Trans
                      i18nKey={policyProps.nativeThumbDes}
                      ns="dashboard"
                      components={
                        policyProps.nativeThumbDoc ? [<Link href={policyProps.nativeThumbDoc} target="_blank" />] : []
                      }
                    />
                  )}
                </NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("policy.nativeThumbnailMaxSize")} lgWidth={5}>
              <FormControl fullWidth>
                <SizeInput
                  variant={"outlined"}
                  value={values.settings?.thumb_max_size ?? 0}
                  onChange={onNativeThumbnailMaxSizeChange}
                />
                <NoMarginHelperText>{t("policy.nativeThumbnailMaxSizeDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
          </>
        )}
        <SettingForm lgWidth={5}>
          <FormControl fullWidth>
            <FormControlLabel
              control={
                <Switch checked={values.settings?.thumb_generator_proxy ?? false} onChange={onThumbProxyChange} />
              }
              label={t("policy.thumbProxy")}
            />
            <NoMarginHelperText>
              <Trans
                i18nKey="policy.thumbProxyDes"
                ns="dashboard"
                components={[<Link component={RouterLink} to="/admin/settings?tab=mediaProcessing" />]}
              />
            </NoMarginHelperText>
          </FormControl>
        </SettingForm>
      </SettingSectionContent>
    </SettingSection>
  );
};

export default ThumbnailsSection;
