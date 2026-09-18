import { Button, Collapse, FormControl, Link, Stack } from "@mui/material";
import { useSnackbar } from "notistack";
import { useMemo, useRef, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { createStoragePolicyCors } from "../../../../../api/api";
import { StoragePolicy } from "../../../../../api/dashboard";
import { PolicyType } from "../../../../../api/explorer";
import { useAppDispatch } from "../../../../../redux/hooks";
import { DefaultCloseAction } from "../../../../Common/Snackbar/snackbar";
import { DenseFilledTextField, SecondaryButton } from "../../../../Common/StyledComponents";
import SettingForm from "../../../../Pages/Setting/SettingForm";
import { Code } from "../../../../Common/Code.tsx";
import { NoMarginHelperText } from "../../../Settings/Settings";
import { AddWizardProps } from "../../AddWizardDialog";
import BucketACLInput from "../../EditStoragePolicy/BucketACLInput";
import BucketCorsTable from "../../EditStoragePolicy/BucketCorsTable";
import { PolicyPropsMap } from "../../StoragePolicySetting";
const OssWizard = ({ onSubmit }: AddWizardProps) => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const [loading, setLoading] = useState(false);
  const { enqueueSnackbar } = useSnackbar();
  const [corsAdded, setCorsAdded] = useState(false);
  const formRef = useRef<HTMLFormElement>(null);
  const [policy, setPolicy] = useState<StoragePolicy>({
    id: 0,
    node_id: 0,
    name: "",
    type: PolicyType.oss,
    is_private: true,
    dir_name_rule: "uploads/{uid}/{path}",
    settings: {
      chunk_size: 25 << 20,
      thumb_exts: ["png", "jpg", "jpeg", "gif", "bmp", "webp", "heic", "tiff", "avif"],
      media_meta_exts: ["jpg", "jpeg", "png", "gif", "bmp", "webp", "tiff", "heic", "heif"],
      media_meta_generator_proxy: true,
      thumb_generator_proxy: true,
      chunk_concurrency: 3,
    },
    file_name_rule: "{uuid}_{originname}",
    edges: {},
  });

  const policyProps = useMemo(() => {
    return PolicyPropsMap[PolicyType.oss];
  }, []);

  const hamdleCreateCors = () => {
    if (!formRef.current?.checkValidity()) {
      formRef.current?.reportValidity();
      return;
    }
    setLoading(true);
    dispatch(createStoragePolicyCors({ policy }))
      .then(() => {
        enqueueSnackbar(t("policy.corsPolicyAdded"), { variant: "success", action: DefaultCloseAction });
        setCorsAdded(true);
      })
      .finally(() => {
        setLoading(false);
      });
  };

  const handleSubmit = () => {
    if (!formRef.current?.checkValidity()) {
      formRef.current?.reportValidity();
      return;
    }
    onSubmit(policy);
  };

  return (
    <form ref={formRef} onSubmit={handleSubmit}>
      <Stack spacing={2}>
        <SettingForm title={t("policy.name")} lgWidth={12}>
          <DenseFilledTextField
            fullWidth
            required
            value={policy.name}
            onChange={(e) => setPolicy({ ...policy, name: e.target.value })}
          />
          <NoMarginHelperText>{t("policy.policyName")}</NoMarginHelperText>
        </SettingForm>
        <SettingForm title={t("policy.bucketName")} lgWidth={12}>
          <DenseFilledTextField
            fullWidth
            required
            value={policy.bucket_name}
            onChange={(e) => setPolicy({ ...policy, bucket_name: e.target.value })}
          />
          <NoMarginHelperText>
            <Trans
              i18nKey="policy.createOSSBucketDes"
              ns="dashboard"
              components={[<Link href="https://oss.console.aliyun.com/bucket" target="_blank" />, <Code />, <Code />]}
            />
          </NoMarginHelperText>
        </SettingForm>
        <SettingForm title={t("policy.bucketType")} lgWidth={12}>
          <FormControl fullWidth>
            <BucketACLInput
              value={policy.is_private ?? false}
              onChange={(value) => setPolicy({ ...policy, is_private: value })}
            />
            <NoMarginHelperText>{t("policy.bucketTypeDes")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>
        <SettingForm title={t("policy.endpoint")} lgWidth={12}>
          <DenseFilledTextField
            fullWidth
            required
            value={policy.server}
            onChange={(e) => setPolicy({ ...policy, server: e.target.value })}
          />
          <NoMarginHelperText>
            <Trans i18nKey="policy.ossEndpointDes" ns="dashboard" components={[<Code />, <Code />, <Code />]} />
            {t("policy.ossEndpointDesInternalHint")}
          </NoMarginHelperText>
        </SettingForm>
        <SettingForm title={t(policyProps.regionCode ?? "")} lgWidth={12}>
          <DenseFilledTextField
            fullWidth
            required
            value={policy.settings?.region}
            onChange={(e) => setPolicy({ ...policy, settings: { ...policy.settings, region: e.target.value } })}
          />
          <NoMarginHelperText>{policyProps.regionCodeDes}</NoMarginHelperText>
        </SettingForm>
        <SettingForm title={t("policy.accessCredential")} lgWidth={12}>
          <FormControl fullWidth>
            <DenseFilledTextField
              placeholder="AccessKey ID"
              fullWidth
              required
              value={policy.access_key}
              onChange={(e) => setPolicy({ ...policy, access_key: e.target.value })}
            />
            <DenseFilledTextField
              placeholder="AccessKey Secret"
              sx={{ mt: 1 }}
              fullWidth
              required
              value={policy.secret_key}
              onChange={(e) => setPolicy({ ...policy, secret_key: e.target.value })}
            />
            <NoMarginHelperText>
              <Trans
                i18nKey="policy.ossAKDes"
                ns="dashboard"
                components={[
                  <Link href="https://ram.console.aliyun.com/profile/access-keys" target="_blank" />,
                  <Link href="https://ram.console.aliyun.com/ram/overview" target="_blank" />,
                  <Code />,
                ]}
              />
            </NoMarginHelperText>
          </FormControl>
        </SettingForm>
        <SettingForm title={t("policy.corsSettingStep")} lgWidth={12}>
          <FormControl fullWidth>
            <BucketCorsTable />
            <NoMarginHelperText>{t("policy.ossCORSDes")}</NoMarginHelperText>
          </FormControl>
          <Collapse in={!corsAdded} sx={{ mt: 1 }}>
            <Button loading={loading} variant="contained" color="primary" sx={{ mr: 1 }} onClick={hamdleCreateCors}>
              {t("policy.letCloudreveHelpMe")}
            </Button>
            <SecondaryButton variant="contained" onClick={() => setCorsAdded(true)}>
              {t("policy.addedManually")}
            </SecondaryButton>
          </Collapse>
        </SettingForm>
      </Stack>
      <Collapse in={corsAdded}>
        <Button variant="contained" color="primary" sx={{ mt: 2 }} onClick={handleSubmit}>
          {t("policy.create")}
        </Button>
      </Collapse>
    </form>
  );
};

export default OssWizard;
