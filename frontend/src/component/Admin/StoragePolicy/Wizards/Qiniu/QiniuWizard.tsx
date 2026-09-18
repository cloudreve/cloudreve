import { Button, FormControl, Link, Stack } from "@mui/material";
import { useRef, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { StoragePolicy } from "../../../../../api/dashboard";
import { PolicyType } from "../../../../../api/explorer";
import { useAppDispatch } from "../../../../../redux/hooks";
import { DenseFilledTextField } from "../../../../Common/StyledComponents";
import SettingForm from "../../../../Pages/Setting/SettingForm";
import { EndpointInput } from "../../../Common/EndpointInput";
import { NoMarginHelperText } from "../../../Settings/Settings";
import { AddWizardProps } from "../../AddWizardDialog";
import BucketACLInput from "../../EditStoragePolicy/BucketACLInput";
const QiniuWizard = ({ onSubmit }: AddWizardProps) => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const formRef = useRef<HTMLFormElement>(null);
  const [policy, setPolicy] = useState<StoragePolicy>({
    id: 0,
    node_id: 0,
    name: "",
    type: PolicyType.qiniu,
    is_private: true,
    dir_name_rule: "uploads/{uid}/{path}",
    settings: {
      chunk_size: 25 << 20,
      thumb_exts: ["psd", "jpeg", "png", "gif", "webp", "tiff", "bmp", "avif", "heic"],
      thumb_max_size: 20 << 20,
      media_meta_exts: [
        "avi",
        "mp4",
        "mkv",
        "mov",
        "webm",
        "opus",
        "flv",
        "hls",
        "ts",
        "dash",
        "mp3",
        "aac",
        "flac",
        "wav",
        "amr",
        "jpg",
        "jpeg",
        "png",
        "gif",
        "bmp",
        "webp",
        "tiff",
        "heic",
        "heif",
      ],
      media_meta_generator_proxy: true,
      thumb_generator_proxy: true,
      custom_proxy: true,
      chunk_concurrency: 3,
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
        <SettingForm title={t("policy.qiniuBucketName")} lgWidth={12}>
          <DenseFilledTextField
            fullWidth
            required
            value={policy.bucket_name}
            onChange={(e) => setPolicy({ ...policy, bucket_name: e.target.value })}
          />
          <NoMarginHelperText>
            <Trans
              i18nKey="policy.enterQiniuBucket"
              ns="dashboard"
              components={[<Link href="https://portal.qiniu.com/kodo/bucket" target="_blank" />]}
            />
          </NoMarginHelperText>
        </SettingForm>
        <SettingForm title={t("policy.aclType")} lgWidth={12}>
          <FormControl fullWidth>
            <BucketACLInput
              phraseVariant={PolicyType.qiniu}
              value={policy.is_private ?? false}
              onChange={(value) => setPolicy({ ...policy, is_private: value })}
            />
            <NoMarginHelperText>{t("policy.bucketTypeDes")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>
        <SettingForm title={t("policy.bucketDomain")} lgWidth={12}>
          <EndpointInput
            fullWidth
            enforceProtocol
            required
            value={policy.settings?.proxy_server ?? ""}
            onChange={(e) => setPolicy({ ...policy, settings: { ...policy.settings, proxy_server: e.target.value } })}
            variant={"outlined"}
          />
          <NoMarginHelperText>{t("policy.bucketDomainDes")}</NoMarginHelperText>
        </SettingForm>
        <SettingForm title={t("policy.accessCredential")} lgWidth={12}>
          <FormControl fullWidth>
            <DenseFilledTextField
              placeholder="AK"
              fullWidth
              required
              value={policy.access_key}
              onChange={(e) => setPolicy({ ...policy, access_key: e.target.value })}
            />
            <DenseFilledTextField
              placeholder="SK"
              sx={{ mt: 1 }}
              fullWidth
              required
              value={policy.secret_key}
              onChange={(e) => setPolicy({ ...policy, secret_key: e.target.value })}
            />
            <NoMarginHelperText>{t("policy.qiniuCredentialDes")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>
      </Stack>
      <Button variant="contained" color="primary" sx={{ mt: 2 }} onClick={handleSubmit}>
        {t("policy.create")}
      </Button>
    </form>
  );
};

export default QiniuWizard;
