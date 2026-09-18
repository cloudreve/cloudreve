import { Button, Collapse, FormControl, Link, Stack } from "@mui/material";
import { useSnackbar } from "notistack";
import { useRef, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { createStoragePolicyCors } from "../../../../../api/api";
import { StoragePolicy } from "../../../../../api/dashboard";
import { PolicyType } from "../../../../../api/explorer";
import { useAppDispatch } from "../../../../../redux/hooks";
import { DefaultCloseAction } from "../../../../Common/Snackbar/snackbar";
import { DenseFilledTextField, SecondaryButton } from "../../../../Common/StyledComponents";
import SettingForm from "../../../../Pages/Setting/SettingForm";
import { Code } from "../../../../Common/Code.tsx";
import { EndpointInput } from "../../../Common/EndpointInput";
import { NoMarginHelperText } from "../../../Settings/Settings";
import { AddWizardProps } from "../../AddWizardDialog";
import BucketACLInput from "../../EditStoragePolicy/BucketACLInput";
import BucketCorsTable from "../../EditStoragePolicy/BucketCorsTable";

const ObsWizard = ({ onSubmit }: AddWizardProps) => {
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
    type: PolicyType.obs,
    is_private: true,
    dir_name_rule: "uploads/{uid}/{path}",
    settings: {
      chunk_size: 25 << 20,
      thumb_exts: ["png", "jpg", "jpeg", "gif", "bmp", "webp", "tiff"],
      media_meta_exts: ["png", "jpg", "jpeg", "gif", "bmp", "webp", "tiff"],
      media_meta_generator_proxy: true,
      thumb_generator_proxy: true,
      chunk_concurrency: 3,
    },
    file_name_rule: "{uuid}_{originname}",
    edges: {},
  });

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
        <SettingForm title={t("policy.cosObsBucketName")} lgWidth={12}>
          <DenseFilledTextField
            fullWidth
            required
            value={policy.bucket_name}
            onChange={(e) => setPolicy({ ...policy, bucket_name: e.target.value })}
          />
          <NoMarginHelperText>
            <Trans
              i18nKey="policy.obsBucketDes"
              ns="dashboard"
              components={[
                <Link href="https://console.huaweicloud.com/console/#/obs/create" target="_blank" />,
                <Code />,
                <Code />,
                <Code />,
              ]}
            />
          </NoMarginHelperText>
        </SettingForm>
        <SettingForm title={t("policy.bucketPolicy")} lgWidth={12}>
          <FormControl fullWidth>
            <BucketACLInput
              phraseVariant={policy.type}
              value={policy.is_private ?? false}
              onChange={(value) => setPolicy({ ...policy, is_private: value })}
            />
            <NoMarginHelperText>{t("policy.bucketTypeDes")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>
        <SettingForm title={t("policy.policyEndpoint")} lgWidth={12}>
          <EndpointInput
            fullWidth
            required
            value={policy.server}
            enforceProtocol
            enforcePrefix={false}
            onChange={(e) => setPolicy({ ...policy, server: e.target.value })}
            variant={"outlined"}
          />
          <NoMarginHelperText>
            <Trans i18nKey="policy.obsEndpointDes" ns="dashboard" components={[<Code />, <Code />]} />
            {t("policy.obsEndpointCnameHint")}
          </NoMarginHelperText>
        </SettingForm>
        <SettingForm title={t("policy.accessCredential")} lgWidth={12}>
          <FormControl fullWidth>
            <DenseFilledTextField
              placeholder="Access Key Id"
              fullWidth
              required
              value={policy.access_key}
              onChange={(e) => setPolicy({ ...policy, access_key: e.target.value })}
            />
            <DenseFilledTextField
              placeholder="Secret Access Key"
              sx={{ mt: 1 }}
              fullWidth
              required
              value={policy.secret_key}
              onChange={(e) => setPolicy({ ...policy, secret_key: e.target.value })}
            />
            <NoMarginHelperText>
              <Trans
                i18nKey="policy.obsCredentialDes"
                ns="dashboard"
                components={[
                  <Link href="https://console.huaweicloud.com/iam/#/mine/accessKey" target="_blank" />,
                  <Code />,
                  <Code />,
                ]}
              />
            </NoMarginHelperText>
          </FormControl>
        </SettingForm>
        <SettingForm title={t("policy.corsSettingStep")} lgWidth={12}>
          <FormControl fullWidth>
            <BucketCorsTable exposedHeaders={["ETag"]} />
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

export default ObsWizard;
