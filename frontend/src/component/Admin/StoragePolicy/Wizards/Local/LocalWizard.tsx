import { Button } from "@mui/material";
import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { StoragePolicy } from "../../../../../api/dashboard";
import { PolicyType } from "../../../../../api/explorer";
import { DenseFilledTextField } from "../../../../Common/StyledComponents";
import SettingForm from "../../../../Pages/Setting/SettingForm";
import { NoMarginHelperText } from "../../../Settings/Settings";
import { AddWizardProps } from "../../AddWizardDialog";

const LocalWizard = ({ onSubmit }: AddWizardProps) => {
  const { t } = useTranslation("dashboard");
  const formRef = useRef<HTMLFormElement>(null);
  const [policy, setPolicy] = useState<StoragePolicy>({
    id: 0,
    name: "",
    type: PolicyType.local,
    file_name_rule: "{uuid}_{originname}",
    settings: {
      chunk_size: 25 << 20,
      pre_allocate: true,
    },
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
      <SettingForm title={t("policy.name")} lgWidth={12}>
        <DenseFilledTextField
          fullWidth
          required
          value={policy.name}
          onChange={(e) => setPolicy({ ...policy, name: e.target.value })}
        />
        <NoMarginHelperText>{t("policy.policyName")}</NoMarginHelperText>
      </SettingForm>
      <Button variant="contained" color="primary" sx={{ mt: 2 }} onClick={handleSubmit}>
        {t("policy.create")}
      </Button>
    </form>
  );
};

export default LocalWizard;
