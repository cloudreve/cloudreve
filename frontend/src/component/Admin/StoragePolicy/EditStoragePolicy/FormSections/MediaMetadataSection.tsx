import { FormControl, FormControlLabel, Link, Switch, Typography } from "@mui/material";
import { useCallback, useContext, useMemo } from "react";
import { Trans, useTranslation } from "react-i18next";
import { Link as RouterLink } from "react-router-dom";
import { StoragePolicy } from "../../../../../api/dashboard";
import { PolicyType } from "../../../../../api/explorer";
import { DenseFilledTextField } from "../../../../Common/StyledComponents";
import SettingForm from "../../../../Pages/Setting/SettingForm";
import { NoMarginHelperText, SettingSection, SettingSectionContent } from "../../../Settings/Settings";
import { PolicyPropsMap } from "../../StoragePolicySetting";
import { StoragePolicySettingContext } from "../StoragePolicySettingWrapper";

const MediaMetadataSection = () => {
  const { t } = useTranslation("dashboard");
  const { values, setPolicy } = useContext(StoragePolicySettingContext);

  const policyProps = useMemo(() => {
    return PolicyPropsMap[values.type];
  }, [values.type]);

  const onNativeMediaMetaExtsChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setPolicy((p: StoragePolicy) => ({
        ...p,
        settings: {
          ...p.settings,
          media_meta_exts: e.target.value === "" ? undefined : e.target.value.split(",").map((ext) => ext.trim()),
        },
      }));
    },
    [setPolicy],
  );

  const onMediaMetaGeneratorProxyChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setPolicy((p: StoragePolicy) => ({
        ...p,
        settings: { ...p.settings, media_meta_generator_proxy: e.target.checked ? true : undefined },
      }));
    },
    [setPolicy],
  );

  const noNativeExtractor = useMemo(() => {
    return values.type === PolicyType.s3 || values.type === PolicyType.ks3 || values.type === PolicyType.onedrive;
  }, [values.type]);

  if (values.type === PolicyType.local) {
    return null;
  }

  return (
    <SettingSection>
      <Typography variant="h6" gutterBottom>
        {t("settings.extractMediaMeta")}
      </Typography>
      <SettingSectionContent>
        {!noNativeExtractor && (
          <SettingForm
            title={
              <Trans
                i18nKey="policy.nativeMediaMetaExts"
                ns="dashboard"
                values={{ name: t(policyProps.nativeExtractorName ?? "") }}
                components={[<span />]}
              />
            }
            lgWidth={5}
          >
            <FormControl fullWidth>
              <DenseFilledTextField
                multiline
                fullWidth
                value={values.settings?.media_meta_exts?.join(",")}
                onChange={onNativeMediaMetaExtsChange}
              />
              <NoMarginHelperText>
                <Trans
                  i18nKey="policy.nativeMediaMetaExtsGeneralDes"
                  values={{ name: t(policyProps.nativeExtractorName ?? "") }}
                  ns="dashboard"
                  components={
                    policyProps.nativeExtractorDoc
                      ? [<Link target="_blank" href={policyProps.nativeExtractorDoc} />]
                      : []
                  }
                />
                {policyProps.nativeExtractorDes && (
                  <Trans
                    i18nKey={policyProps.nativeExtractorDes}
                    ns="dashboard"
                    components={[<Link target="_blank" href={policyProps.nativeExtractorDesDoc} />]}
                  />
                )}
              </NoMarginHelperText>
            </FormControl>
          </SettingForm>
        )}
        <SettingForm lgWidth={5}>
          <FormControl fullWidth>
            <FormControlLabel
              control={
                <Switch
                  checked={values.settings?.media_meta_generator_proxy ?? false}
                  onChange={onMediaMetaGeneratorProxyChange}
                />
              }
              label={t("policy.mediaExtractorProxy")}
            />
            <NoMarginHelperText>
              <Trans
                i18nKey="policy.mediaExtractorProxyDes"
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

export default MediaMetadataSection;
