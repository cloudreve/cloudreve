import { Box, Checkbox, Collapse, FormControl, FormControlLabel, Switch, Typography } from "@mui/material";
import { useCallback, useContext } from "react";
import { Trans, useTranslation } from "react-i18next";
import { StoragePolicy } from "../../../../../api/dashboard";
import { PolicyType } from "../../../../../api/explorer";
import SettingForm from "../../../../Pages/Setting/SettingForm";
import { EndpointInput } from "../../../Common/EndpointInput";
import { NoMarginHelperText, SettingSection, SettingSectionContent } from "../../../Settings/Settings";
import { TrafficDiagram } from "../../TrafficDiagram";
import { StoragePolicySettingContext } from "../StoragePolicySettingWrapper";

const DownloadSection = () => {
  const { t } = useTranslation("dashboard");
  const { values, setPolicy } = useContext(StoragePolicySettingContext);

  const onDownloadCdnChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setPolicy((p: StoragePolicy) => ({
        ...p,
        settings: { ...p.settings, custom_proxy: e.target.checked ? true : undefined },
      }));
    },
    [setPolicy],
  );

  const onProxyServerChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setPolicy((p: StoragePolicy) => ({
        ...p,
        settings: { ...p.settings, proxy_server: e.target.value },
      }));
    },
    [setPolicy],
  );

  const onInternalProxyChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setPolicy((p: StoragePolicy) => ({
        ...p,
        settings: { ...p.settings, internal_proxy: e.target.checked ? true : undefined },
      }));
    },
    [setPolicy],
  );

  const onStreamSaverChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setPolicy((p: StoragePolicy) => ({
        ...p,
        settings: { ...p.settings, stream_saver: e.target.checked ? true : undefined },
      }));
    },
    [setPolicy],
  );

  const onSkipSignChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setPolicy((p: StoragePolicy) => ({
        ...p,
        settings: { ...p.settings, source_auth: e.target.checked ? true : undefined },
      }));
    },
    [setPolicy],
  );

  return (
    <SettingSection>
      <Typography variant="h6" gutterBottom>
        {t("policy.download")}
      </Typography>
      <SettingSectionContent>
        <SettingForm title={t("policy.downloadCdn")} lgWidth={5}>
          <FormControl fullWidth>
            <FormControlLabel
              slotProps={{
                typography: {
                  variant: "body2",
                },
              }}
              control={
                <Checkbox
                  disabled={values.type === PolicyType.upyun || values.type === PolicyType.qiniu}
                  size={"small"}
                  checked={values.settings?.custom_proxy ?? false}
                  onChange={onDownloadCdnChange}
                />
              }
              label={t("policy.useDownloadCdn")}
            />
            <Collapse in={values.settings?.custom_proxy} unmountOnExit>
              <Box>
                <EndpointInput
                  fullWidth
                  required
                  enforceProtocol
                  variant={"outlined"}
                  value={values.settings?.proxy_server ?? ""}
                  onChange={onProxyServerChange}
                />
                <NoMarginHelperText>{t("policy.downloadCdnDes")}</NoMarginHelperText>
                {values.type == PolicyType.cos && values.is_private && (
                  <FormControl fullWidth>
                    <FormControlLabel
                      slotProps={{
                        typography: {
                          variant: "body2",
                        },
                      }}
                      control={
                        <Checkbox
                          size={"small"}
                          checked={values.settings?.source_auth ?? false}
                          onChange={onSkipSignChange}
                        />
                      }
                      label={t("policy.skipSign")}
                    />
                    <NoMarginHelperText>{t("policy.skipSignDes")}</NoMarginHelperText>
                  </FormControl>
                )}
              </Box>
            </Collapse>
          </FormControl>
        </SettingForm>
        {values.type !== PolicyType.local && (
          <SettingForm lgWidth={5}>
            <FormControl fullWidth>
              <FormControlLabel
                control={<Switch checked={values.settings?.internal_proxy ?? false} onChange={onInternalProxyChange} />}
                label={t("policy.downloadRelay")}
              />
              <NoMarginHelperText>
                <Trans i18nKey="policy.downloadRelayDes" ns="dashboard" />
              </NoMarginHelperText>
            </FormControl>
          </SettingForm>
        )}
        {values.type === PolicyType.onedrive && (
          <SettingForm lgWidth={5}>
            <FormControl fullWidth>
              <FormControlLabel
                control={<Switch checked={values.settings?.stream_saver ?? false} onChange={onStreamSaverChange} />}
                label={t("policy.streamSaver")}
              />
              <NoMarginHelperText>
                <Trans i18nKey="policy.streamSaverDes" ns="dashboard" />
              </NoMarginHelperText>
            </FormControl>
          </SettingForm>
        )}
        {values.type !== PolicyType.local && (
          <SettingForm lgWidth={5} title={t("policy.downloadTrafficDiagram")}>
            <TrafficDiagram
              variant="download"
              internalEndpoint={!!values.settings?.server_side_endpoint}
              cdn={values.settings?.custom_proxy}
              proxyed={values.settings?.internal_proxy}
              proxyNodeTitle={values.type === PolicyType.qiniu ? t("policy.cdnOrCustomDomain") : undefined}
              storageNodeTitle={t("policy.node")}
            />
          </SettingForm>
        )}
      </SettingSectionContent>
    </SettingSection>
  );
};

export default DownloadSection;
