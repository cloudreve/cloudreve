import { Box, Button, Link, Stack, Typography } from "@mui/material";
import { useEffect, useRef, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { getPolicyOauthRedirectUrl } from "../../../../../api/api";
import { StoragePolicy } from "../../../../../api/dashboard";
import { PolicyType } from "../../../../../api/explorer";
import { useAppDispatch } from "../../../../../redux/hooks";
import { DenseFilledTextField } from "../../../../Common/StyledComponents";
import SettingForm from "../../../../Pages/Setting/SettingForm";
import { Code } from "../../../../Common/Code.tsx";
import { NoMarginHelperText } from "../../../Settings/Settings";
import { AddWizardProps } from "../../AddWizardDialog";
import GraphEndpointSelection from "./GraphEndpointSelection";

const wwGraph = "https://graph.microsoft.com/v1.0";

const OneDriveWizard = ({ onSubmit }: AddWizardProps) => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const [redirectUrl, setRedirectUrl] = useState<string>("Loading...");
  const formRef = useRef<HTMLFormElement>(null);
  const [policy, setPolicy] = useState<StoragePolicy>({
    id: 0,
    node_id: 0,
    name: "",
    type: PolicyType.onedrive,
    server: wwGraph,
    is_private: true,
    dir_name_rule: "uploads/{uid}/{path}",
    settings: {
      chunk_size: 50 << 20,
      thumb_support_all_exts: true,
      media_meta_generator_proxy: true,
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

  useEffect(() => {
    dispatch(getPolicyOauthRedirectUrl()).then((res) => {
      setRedirectUrl(res);
    });
  }, []);

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
        <SettingForm title={t("policy.aadAccountCloud")} lgWidth={12}>
          <GraphEndpointSelection
            fullWidth
            value={policy.server ?? ""}
            required
            onChange={(value) => setPolicy({ ...policy, server: value })}
          />
          <NoMarginHelperText>{t("policy.aadAccountCloudDes")}</NoMarginHelperText>
        </SettingForm>
        <SettingForm title={t("policy.entraIdApp")} lgWidth={12}>
          <Typography component={"div"} variant="caption" color="textSecondary">
            <Trans
              i18nKey="policy.creatAadAppDes"
              ns="dashboard"
              components={[
                <Link
                  href={
                    wwGraph === policy.server
                      ? "https://portal.azure.com/#blade/Microsoft_AAD_IAM/ActiveDirectoryMenuBlade/Overview"
                      : "https://portal.azure.cn/#blade/Microsoft_AAD_IAM/ActiveDirectoryMenuBlade/Overview"
                  }
                  target="_blank"
                />,
                <Code />,
              ]}
            />
          </Typography>
          <Typography sx={{ mt: 1 }} component={"div"} variant="caption" color="textSecondary">
            <Trans
              i18nKey="policy.createAadAppDes2"
              ns="dashboard"
              values={{ url: redirectUrl }}
              components={[<Code />, <Code />, <Code />, <Code />, <Code />, <Code />, <Code />]}
            />
          </Typography>
          <DenseFilledTextField
            sx={{ mt: 2 }}
            fullWidth
            required
            placeholder={t("policy.aadAppID")}
            value={policy.bucket_name}
            onChange={(e) => setPolicy({ ...policy, bucket_name: e.target.value })}
          />
          <NoMarginHelperText>
            <Trans i18nKey="policy.aadAppIDDes" ns="dashboard" components={[<Code />, <Code />]} />
          </NoMarginHelperText>
          <DenseFilledTextField
            sx={{ mt: 2 }}
            fullWidth
            required
            placeholder={t("policy.aadAppSecret")}
            value={policy.secret_key}
            onChange={(e) => setPolicy({ ...policy, secret_key: e.target.value })}
          />
          <NoMarginHelperText>
            <Trans i18nKey="policy.addAppSecretDes" ns="dashboard" components={[<Code />, <Code />, <Code />]} />
          </NoMarginHelperText>
        </SettingForm>
        <SettingForm title={t("policy.grantAccess")} lgWidth={12}>
          <Typography component={"div"} variant="caption" color="textSecondary">
            {t("policy.grantAccessLater")}
          </Typography>
        </SettingForm>
        <Box>
          <Button variant="contained" color="primary" sx={{ mt: 2 }} onClick={handleSubmit}>
            {t("policy.create")}
          </Button>
        </Box>
      </Stack>
    </form>
  );
};

export default OneDriveWizard;
