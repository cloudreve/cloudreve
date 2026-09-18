import { Button, Collapse, FormControl, Link, Stack } from "@mui/material";
import { useSnackbar } from "notistack";
import { useRef, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { StoragePolicy } from "../../../../../api/dashboard";
import { PolicyType } from "../../../../../api/explorer";
import { useAppDispatch } from "../../../../../redux/hooks";
import { DenseFilledTextField } from "../../../../Common/StyledComponents";
import SettingForm from "../../../../Pages/Setting/SettingForm";
import { Code } from "../../../../Common/Code.tsx";
import { EndpointInput } from "../../../Common/EndpointInput";
import { NoMarginHelperText } from "../../../Settings/Settings";
import { AddWizardProps } from "../../AddWizardDialog";
import BucketACLInput from "../../EditStoragePolicy/BucketACLInput";

const UpyunWizard = ({ onSubmit }: AddWizardProps) => {
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
    type: PolicyType.upyun,
    is_private: true,
    dir_name_rule: "uploads/{uid}/{path}",
    settings: {
      thumb_exts: ["png", "jpg", "jpeg", "gif", "bmp", "webp", "svg"],
      media_meta_exts: ["png", "jpg", "jpeg", "gif", "bmp", "webp", "svg"],
      media_meta_generator_proxy: true,
      thumb_generator_proxy: true,
      custom_proxy: true,
    },
    file_name_rule: "{uuid}_{originname}",
    edges: {},
  });

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
        <SettingForm title={t("policy.storageServiceName")} lgWidth={12}>
          <DenseFilledTextField
            fullWidth
            required
            value={policy.bucket_name}
            onChange={(e) => setPolicy({ ...policy, bucket_name: e.target.value })}
          />
          <NoMarginHelperText>
            <Trans
              i18nKey="policy.createUpyunBucketDes"
              ns="dashboard"
              components={[<Link href="https://console.upyun.com/console/service" target="_blank" />]}
            />
          </NoMarginHelperText>
        </SettingForm>
        <SettingForm title={t("policy.tokenStatus")} lgWidth={12}>
          <FormControl fullWidth>
            <BucketACLInput
              phraseVariant={policy.type}
              value={policy.is_private ?? false}
              onChange={(value) => setPolicy({ ...policy, is_private: value })}
            />
            <NoMarginHelperText>
              <Trans i18nKey="policy.upyunTokenDes" ns="dashboard" components={[<Code />, <Code />]} />
            </NoMarginHelperText>
          </FormControl>
        </SettingForm>
        <Collapse in={policy.is_private}>
          <SettingForm title={t("policy.upyunTokenSecret")} lgWidth={12}>
            <DenseFilledTextField
              fullWidth
              required
              value={policy.settings?.token}
              onChange={(e) => setPolicy({ ...policy, settings: { ...policy.settings, token: e.target.value } })}
            />
            <NoMarginHelperText>{t("policy.upyunTokenSecretDes")}</NoMarginHelperText>
          </SettingForm>
        </Collapse>
        <SettingForm title={t("policy.bucketCDNDomain")} lgWidth={12}>
          <EndpointInput
            fullWidth
            enforceProtocol
            required
            value={policy.settings?.proxy_server}
            onChange={(e) => setPolicy({ ...policy, settings: { ...policy.settings, proxy_server: e.target.value } })}
            variant={"outlined"}
          />
          <NoMarginHelperText>{t("policy.bucketCDNDes")}</NoMarginHelperText>
        </SettingForm>
        <SettingForm title={t("policy.accessCredential")} lgWidth={12}>
          <FormControl fullWidth>
            <DenseFilledTextField
              placeholder={t("policy.operatorName")}
              fullWidth
              required
              value={policy.access_key}
              onChange={(e) => setPolicy({ ...policy, access_key: e.target.value })}
            />
            <DenseFilledTextField
              placeholder={t("policy.operatorPassword")}
              sx={{ mt: 1 }}
              fullWidth
              required
              value={policy.secret_key}
              onChange={(e) => setPolicy({ ...policy, secret_key: e.target.value })}
            />
          </FormControl>
        </SettingForm>
      </Stack>
      <Button variant="contained" color="primary" sx={{ mt: 2 }} onClick={handleSubmit}>
        {t("policy.create")}
      </Button>
    </form>
  );
};

export default UpyunWizard;
