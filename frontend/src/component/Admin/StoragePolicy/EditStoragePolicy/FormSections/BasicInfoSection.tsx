import {
  Checkbox,
  CircularProgress,
  Collapse,
  FormControl,
  FormControlLabel,
  Link,
  ListItemText,
  SelectChangeEvent,
  Typography,
} from "@mui/material";
import { useSnackbar } from "notistack";
import { useCallback, useContext, useMemo, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { Link as RouterLink } from "react-router-dom";
import { createStoragePolicyCors, getOneDriveDriverRoot } from "../../../../../api/api";
import { StoragePolicy } from "../../../../../api/dashboard";
import { PolicyType } from "../../../../../api/explorer";
import { useAppDispatch } from "../../../../../redux/hooks";
import { DefaultCloseAction } from "../../../../Common/Snackbar/snackbar";
import { DenseFilledTextField, DenseSelect, SecondaryButton } from "../../../../Common/StyledComponents";
import { SquareMenuItem } from "../../../../FileManager/ContextMenu/ContextMenu";
import SettingForm from "../../../../Pages/Setting/SettingForm";
import { Code } from "../../../../Common/Code.tsx";
import { EndpointInput } from "../../../Common/EndpointInput";
import NodeSelectionInput from "../../../Common/NodeSelectionInput";
import { NoMarginHelperText, SettingSection, SettingSectionContent } from "../../../Settings/Settings";
import { PolicyPropsMap } from "../../StoragePolicySetting";
import GraphEndpointSelection from "../../Wizards/OneDrive/GraphEndpointSelection";
import BucketACLInput from "../BucketACLInput";
import BucketCorsTable from "../BucketCorsTable";
import { StoragePolicySettingContext } from "../StoragePolicySettingWrapper";
import OdSignInStatus from "./OdSignInStatus";

export const SharePointDriverPending = "sharepoint_pending";

const BasicInfoSection = () => {
  const { t } = useTranslation("dashboard");
  const { values, setPolicy } = useContext(StoragePolicySettingContext);
  const [corsLoading, setCorsLoading] = useState(false);
  const { enqueueSnackbar } = useSnackbar();
  const dispatch = useAppDispatch();

  // Extract sharepoint URL from od_driver if it exists and is not the default
  const initialSharepointUrl = useMemo(() => {
    if (values.settings?.od_driver && values.settings.od_driver !== "me/drive") {
      return values.settings.od_driver;
    }
    return "";
  }, [values.settings?.od_driver]);

  const [sharepointUrl, setSharepointUrl] = useState(initialSharepointUrl);
  const [sharepointLoading, setSharepointLoading] = useState(false);

  const onNameChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setPolicy((p: StoragePolicy) => ({ ...p, name: e.target.value }));
    },
    [setPolicy],
  );

  const onNodeChange = useCallback(
    (value: number) => {
      setPolicy((p: StoragePolicy) => ({ ...p, node_id: value > 0 ? value : undefined }));
    },
    [setPolicy],
  );

  const onBucketNameChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setPolicy((p: StoragePolicy) => ({ ...p, bucket_name: e.target.value }));
    },
    [setPolicy],
  );

  const showBucket = useMemo(() => {
    return (
      values.type === PolicyType.oss ||
      values.type === PolicyType.cos ||
      values.type === PolicyType.obs ||
      values.type === PolicyType.qiniu ||
      values.type === PolicyType.s3 ||
      values.type === PolicyType.ks3 ||
      values.type === PolicyType.upyun
    );
  }, [values.type]);

  const showEndpoint = useMemo(() => {
    return values.type === PolicyType.cos || values.type === PolicyType.obs || values.type === PolicyType.s3;
  }, [values.type]);

  const showCors = useMemo(() => {
    return (
      values.type === PolicyType.oss ||
      values.type === PolicyType.cos ||
      values.type === PolicyType.obs ||
      values.type === PolicyType.s3 ||
      values.type === PolicyType.ks3
    );
  }, [values.type]);

  const policyProps = useMemo(() => {
    return PolicyPropsMap[values.type];
  }, [values.type]);

  const onBucketTypeChange = useCallback(
    (value: boolean) => {
      setPolicy((p: StoragePolicy) => ({ ...p, is_private: value ? true : undefined }));
    },
    [setPolicy],
  );

  const onEndpointChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setPolicy((p: StoragePolicy) => ({ ...p, server: e.target.value }));
    },
    [setPolicy],
  );

  const onIntranetEndpointChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setPolicy((p: StoragePolicy) => ({
        ...p,
        settings: { ...p.settings, server_side_endpoint: e.target.value ? e.target.value : undefined },
      }));
    },
    [setPolicy],
  );

  const onUseCnameChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setPolicy((p: StoragePolicy) => ({
        ...p,
        settings: { ...p.settings, use_cname: e.target.checked ? true : undefined },
      }));
    },
    [setPolicy],
  );

  const onAccessKeyChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setPolicy((p: StoragePolicy) => ({ ...p, access_key: e.target.value }));
    },
    [setPolicy],
  );

  const onSecretKeyChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setPolicy((p: StoragePolicy) => ({ ...p, secret_key: e.target.value }));
    },
    [setPolicy],
  );

  const onS3ForcePathStyleChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setPolicy((p: StoragePolicy) => ({
        ...p,
        settings: { ...p.settings, s3_path_style: e.target.checked ? true : undefined },
      }));
    },
    [setPolicy],
  );

  const onS3RegionChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setPolicy((p: StoragePolicy) => ({ ...p, settings: { ...p.settings, region: e.target.value } }));
    },
    [setPolicy],
  );

  const onS3DeleteBatchSizeChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setPolicy((p: StoragePolicy) => ({
        ...p,
        settings: {
          ...p.settings,
          s3_delete_batch_size: parseInt(e.target.value) ? parseInt(e.target.value) : undefined,
        },
      }));
    },
    [setPolicy],
  );

  const onGraphEndpointChange = useCallback(
    (value: string) => {
      setPolicy((p: StoragePolicy) => ({ ...p, server: value }));
    },
    [setPolicy],
  );

  const onDriverTypeChange = useCallback(
    (e: SelectChangeEvent<unknown>) => {
      const value = e.target.value as string;
      if (value === "default") {
        setPolicy((p: StoragePolicy) => ({
          ...p,
          settings: { ...p.settings, od_driver: "me/drive" },
        }));
      } else {
        // When switching to SharePoint, set an empty URL initially
        setSharepointUrl("");
        setPolicy((p: StoragePolicy) => ({
          ...p,
          settings: { ...p.settings, od_driver: SharePointDriverPending }, // Temporary value to trigger the collapse
        }));
      }
    },
    [setPolicy, setSharepointUrl],
  );

  const onSharepointUrlChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setSharepointUrl(e.target.value);
  }, []);

  const handleSharepointUrlBlur = useCallback(() => {
    if (!sharepointUrl || sharepointUrl.startsWith("sites/")) return;

    setSharepointLoading(true);
    dispatch(getOneDriveDriverRoot(values.id, sharepointUrl))
      .then((res) => {
        setPolicy((p: StoragePolicy) => ({
          ...p,
          settings: { ...p.settings, od_driver: res },
        }));
        setSharepointUrl(res);
      })
      .finally(() => {
        setSharepointLoading(false);
      });
  }, [sharepointUrl, setPolicy]);

  const handleCreateCors = useCallback(() => {
    setCorsLoading(true);
    dispatch(
      createStoragePolicyCors({
        policy: values,
      }),
    )
      .then(() => {
        enqueueSnackbar(t("policy.corsPolicyAdded"), {
          variant: "success",
          action: DefaultCloseAction,
        });
      })
      .finally(() => {
        setCorsLoading(false);
      });
  }, [dispatch, values, t]);

  const onOdTpsLimitChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setPolicy((p: StoragePolicy) => ({
        ...p,
        settings: { ...p.settings, tps_limit: parseFloat(e.target.value) ? parseFloat(e.target.value) : undefined },
      }));
    },
    [setPolicy],
  );

  const onOdTpsLimitBurstChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setPolicy((p: StoragePolicy) => ({
        ...p,
        settings: {
          ...p.settings,
          tps_limit_burst: parseInt(e.target.value) ? parseInt(e.target.value) : undefined,
        },
      }));
    },
    [setPolicy],
  );

  const onTokenChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setPolicy((p: StoragePolicy) => ({
        ...p,
        settings: { ...p.settings, token: e.target.value ? e.target.value : undefined },
      }));
    },
    [setPolicy],
  );
  return (
    <SettingSection>
      <Typography variant="h6" gutterBottom>
        {t("policy.basicInfo")}
      </Typography>
      <SettingSectionContent>
        <SettingForm title={t("policy.name")} lgWidth={5}>
          <FormControl fullWidth>
            <DenseFilledTextField required value={values.name} onChange={onNameChange} />
            <NoMarginHelperText>{t("policy.policyName")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>
        {showBucket && (
          <>
            <SettingForm title={t(policyProps.bucketName ?? "")} lgWidth={5}>
              <DenseFilledTextField fullWidth required value={values.bucket_name} onChange={onBucketNameChange} />
              <NoMarginHelperText>{policyProps.bucketNameDes}</NoMarginHelperText>
            </SettingForm>
            <SettingForm title={t(policyProps.bucketType ?? "")} lgWidth={5}>
              <FormControl fullWidth>
                <BucketACLInput
                  phraseVariant={values.type}
                  value={values.is_private ?? false}
                  onChange={onBucketTypeChange}
                />
                <NoMarginHelperText>{t("policy.bucketTypeDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            {values.type === PolicyType.upyun && (
              <Collapse in={values.is_private}>
                <SettingForm title={t("policy.upyunTokenSecret")} lgWidth={5}>
                  <DenseFilledTextField fullWidth required value={values.settings?.token} onChange={onTokenChange} />
                  <NoMarginHelperText>{t("policy.upyunTokenSecretDes")}</NoMarginHelperText>
                </SettingForm>
              </Collapse>
            )}
            {showEndpoint && (
              <SettingForm title={t(policyProps.endpointName ?? "")} lgWidth={5}>
                <EndpointInput
                  fullWidth
                  enforceProtocol
                  required
                  value={values.server}
                  onChange={onEndpointChange}
                  variant={"outlined"}
                  enforcePrefix={!policyProps.endpointNotEnforcePrefix}
                />
                <NoMarginHelperText>{policyProps.endpointDes}</NoMarginHelperText>
                {values.type == PolicyType.obs && (
                  <>
                    <FormControlLabel
                      slotProps={{
                        typography: {
                          variant: "body2",
                        },
                      }}
                      control={
                        <Checkbox
                          size={"small"}
                          checked={values.settings?.use_cname ?? false}
                          onChange={onUseCnameChange}
                        />
                      }
                      label={t("policy.thisIsACustomDomain")}
                    />
                    <NoMarginHelperText>{t("policy.thisIsACustomDomainDes")}</NoMarginHelperText>
                  </>
                )}
                {(values.type === PolicyType.s3 || values.type === PolicyType.ks3) && (
                  <>
                    <FormControlLabel
                      sx={{ mt: 1, mb: -1 }}
                      slotProps={{
                        typography: {
                          variant: "body2",
                        },
                      }}
                      control={
                        <Checkbox
                          size={"small"}
                          checked={values.settings?.s3_path_style ?? false}
                          onChange={onS3ForcePathStyleChange}
                        />
                      }
                      label={t("policy.usePathEndpoint")}
                    />
                    <NoMarginHelperText>
                      <Trans
                        i18nKey={
                          values.type === PolicyType.s3 ? "policy.s3EndpointPathStyle" : "policy.ks3EndpointPathStyle"
                        }
                        ns="dashboard"
                        components={[<Code />]}
                      />
                    </NoMarginHelperText>
                  </>
                )}
              </SettingForm>
            )}
            {values.type === PolicyType.oss && (
              <>
                <SettingForm title={t("policy.endpoint")} lgWidth={5}>
                  <DenseFilledTextField fullWidth required value={values.server} onChange={onEndpointChange} />
                  <NoMarginHelperText>
                    <Trans i18nKey="policy.ossEndpointDes" ns="dashboard" components={[<Code />, <Code />, <Code />]} />
                  </NoMarginHelperText>
                  <FormControlLabel
                    sx={{ mt: 1, mb: -1 }}
                    slotProps={{
                      typography: {
                        variant: "body2",
                      },
                    }}
                    control={
                      <Checkbox
                        size={"small"}
                        checked={values.settings?.use_cname ?? false}
                        onChange={onUseCnameChange}
                      />
                    }
                    label={t("policy.thisIsACustomDomain")}
                  />
                  <NoMarginHelperText>{t("policy.thisIsACustomDomainDes")}</NoMarginHelperText>
                </SettingForm>
                <SettingForm title={t("policy.intranetEndPoint")} lgWidth={5}>
                  <DenseFilledTextField
                    fullWidth
                    value={values.settings?.server_side_endpoint}
                    onChange={onIntranetEndpointChange}
                  />
                  <NoMarginHelperText>{t("policy.ossLANEndpointDes")}</NoMarginHelperText>
                </SettingForm>
              </>
            )}
            {policyProps.regionCode && (
              <SettingForm title={t(policyProps.regionCode ?? "")} lgWidth={5}>
                <DenseFilledTextField fullWidth required value={values.settings?.region} onChange={onS3RegionChange} />
                <NoMarginHelperText>{policyProps.regionCodeDes}</NoMarginHelperText>
              </SettingForm>
            )}
            <SettingForm title={t("policy.accessCredential")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  placeholder={t(policyProps.akName ?? "")}
                  fullWidth
                  required
                  value={values.access_key}
                  onChange={onAccessKeyChange}
                />
                <DenseFilledTextField
                  placeholder={t(policyProps.skName ?? "")}
                  sx={{ mt: 1 }}
                  fullWidth
                  required
                  value={values.secret_key}
                  onChange={onSecretKeyChange}
                />
                <NoMarginHelperText>{policyProps.credentialDes}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
          </>
        )}
        {values.type === PolicyType.remote && (
          <SettingForm title={t("policy.node")} lgWidth={5}>
            <NodeSelectionInput fullWidth required value={values.node_id ?? 0} onChange={onNodeChange} />
            <NoMarginHelperText>
              <Trans
                i18nKey="policy.nodeDes"
                ns="dashboard"
                components={[<Link component={RouterLink} to="/admin/node" />]}
              />
            </NoMarginHelperText>
          </SettingForm>
        )}
        {showCors && (
          <SettingForm title={t("policy.corsSettingStep")} lgWidth={5}>
            <FormControl fullWidth>
              <BucketCorsTable exposedHeaders={policyProps.corsExposedHeaders} />
              <NoMarginHelperText>{t("policy.ossCORSDes")}</NoMarginHelperText>
            </FormControl>
            <SecondaryButton
              loading={corsLoading}
              variant="contained"
              color="primary"
              sx={{ mt: 1 }}
              onClick={handleCreateCors}
            >
              {t("policy.letCloudreveHelpMe")}
            </SecondaryButton>
          </SettingForm>
        )}
        {(values.type === PolicyType.s3 || values.type === PolicyType.ks3) && (
          <SettingForm title={t("policy.batchDeleteSize")} lgWidth={5}>
            <DenseFilledTextField
              fullWidth
              slotProps={{
                htmlInput: {
                  min: 0,
                },
              }}
              value={values.settings?.s3_delete_batch_size}
              onChange={onS3DeleteBatchSizeChange}
              type="number"
            />
            <NoMarginHelperText>
              <Trans i18nKey="policy.batchDeleteSizeDes" ns="dashboard" components={[<Code />]} />
            </NoMarginHelperText>
          </SettingForm>
        )}
        {values.type == PolicyType.onedrive && (
          <>
            <SettingForm title={t("policy.aadAccountCloud")} lgWidth={5}>
              <GraphEndpointSelection fullWidth value={values.server ?? ""} required onChange={onGraphEndpointChange} />
              <NoMarginHelperText>{t("policy.aadAccountCloudDes")}</NoMarginHelperText>
            </SettingForm>
            <SettingForm title={t("policy.grantAccess")} lgWidth={5}>
              <OdSignInStatus />
            </SettingForm>
            <SettingForm title={t("policy.driverRoot")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseSelect
                  value={
                    values.settings?.od_driver === undefined ||
                    values.settings?.od_driver === "" ||
                    values.settings?.od_driver === "me/drive"
                      ? "default"
                      : "sharepoint"
                  }
                  onChange={onDriverTypeChange}
                >
                  <SquareMenuItem value={"default"}>
                    <ListItemText
                      slotProps={{
                        primary: { variant: "body2" },
                      }}
                    >
                      {t("policy.saveToDefaultOneDrive")}
                    </ListItemText>
                  </SquareMenuItem>
                  <SquareMenuItem value="sharepoint">
                    <ListItemText
                      slotProps={{
                        primary: { variant: "body2" },
                      }}
                    >
                      {t("policy.saveToSharePoint")}
                    </ListItemText>
                  </SquareMenuItem>
                </DenseSelect>
                <NoMarginHelperText>{t("policy.driverRootDes")}</NoMarginHelperText>
              </FormControl>
              <Collapse
                in={
                  values.settings?.od_driver !== undefined &&
                  values.settings?.od_driver !== "" &&
                  values.settings?.od_driver !== "me/drive"
                }
              >
                <FormControl fullWidth sx={{ mt: 2 }}>
                  <DenseFilledTextField
                    fullWidth
                    placeholder="https://example.sharepoint.com/sites/yoursite"
                    value={sharepointUrl}
                    onChange={onSharepointUrlChange}
                    onBlur={handleSharepointUrlBlur}
                    disabled={sharepointLoading}
                    slotProps={{
                      input: {
                        endAdornment: sharepointLoading ? <CircularProgress size={20} color="inherit" /> : null,
                      },
                    }}
                  />
                  <NoMarginHelperText>{t("policy.sharePointUrlDes")}</NoMarginHelperText>
                </FormControl>
              </Collapse>
            </SettingForm>
            <SettingForm title={t("policy.limitOdTPSDes")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  placeholder={t("policy.tps")}
                  slotProps={{
                    htmlInput: {
                      min: 0,
                      step: 0.1,
                    },
                  }}
                  fullWidth
                  value={values.settings?.tps_limit}
                  onChange={onOdTpsLimitChange}
                />
                <NoMarginHelperText>{t("policy.tpsDes")}</NoMarginHelperText>
                <DenseFilledTextField
                  sx={{ mt: 1 }}
                  placeholder={t("policy.tpsBurst")}
                  slotProps={{
                    htmlInput: {
                      min: 0,
                      step: 0.1,
                    },
                  }}
                  fullWidth
                  value={values.settings?.tps_limit_burst}
                  onChange={onOdTpsLimitBurstChange}
                />
                <NoMarginHelperText>{t("policy.tpsBurstDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
          </>
        )}
      </SettingSectionContent>
    </SettingSection>
  );
};

export default BasicInfoSection;
