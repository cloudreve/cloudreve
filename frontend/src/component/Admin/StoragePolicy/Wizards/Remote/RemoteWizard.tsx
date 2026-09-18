import { Button, Link, Stack } from "@mui/material";
import { useRef, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { Link as RouterLink } from "react-router-dom";
import { StoragePolicy } from "../../../../../api/dashboard";
import { PolicyType } from "../../../../../api/explorer";
import { DenseFilledTextField } from "../../../../Common/StyledComponents";
import SettingForm from "../../../../Pages/Setting/SettingForm";
import NodeSelectionInput from "../../../Common/NodeSelectionInput";
import { NoMarginHelperText } from "../../../Settings/Settings";
import { AddWizardProps } from "../../AddWizardDialog";
const RemoteWizard = ({ onSubmit }: AddWizardProps) => {
  const { t } = useTranslation("dashboard");
  const formRef = useRef<HTMLFormElement>(null);
  const [policy, setPolicy] = useState<StoragePolicy>({
    id: 0,
    node_id: 0,
    name: "",
    type: PolicyType.remote,
    dir_name_rule: "data/uploads/{uid}/{path}",
    settings: {
      chunk_size: 25 << 20,
      pre_allocate: true,
      thumb_exts: ["jpg", "png", "gif", "jpeg", "mp3", "m4a", "ogg", "flag"],
      thumb_generator_proxy: true,
      media_meta_exts: [
        "mp3",
        "m4a",
        "ogg",
        "flac",
        "jpg",
        "jpeg",
        "png",
        "heic",
        "heif",
        "tiff",
        "avif",
        "3fr",
        "ari",
        "arw",
        "bay",
        "braw",
        "crw",
        "cr2",
        "cr3",
        "cap",
        "data",
        "dcs",
        "dcr",
        "dng",
        "drf",
        "eip",
        "erf",
        "fff",
        "gpr",
        "iiq",
        "k25",
        "kdc",
        "mdc",
        "mef",
        "mos",
        "mrw",
        "nef",
        "nrw",
        "obm",
        "orf",
        "pef",
        "ptx",
        "pxn",
        "r3d",
        "raf",
        "raw",
        "rwl",
        "rw2",
        "rwz",
        "sr2",
        "srf",
        "srw",
        "tif",
        "x3f",
      ],
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
        <SettingForm title={t("policy.node")} lgWidth={12}>
          <NodeSelectionInput
            fullWidth
            required
            value={policy.node_id ?? 0}
            onChange={(value) => setPolicy({ ...policy, node_id: value })}
          />
          <NoMarginHelperText>
            <Trans
              i18nKey="policy.nodeDes"
              ns="dashboard"
              components={[<Link component={RouterLink} to="/admin/node" />]}
            />
          </NoMarginHelperText>
        </SettingForm>
      </Stack>
      <Button disabled={!policy.node_id} variant="contained" color="primary" sx={{ mt: 2 }} onClick={handleSubmit}>
        {t("policy.create")}
      </Button>
    </form>
  );
};

export default RemoteWizard;
