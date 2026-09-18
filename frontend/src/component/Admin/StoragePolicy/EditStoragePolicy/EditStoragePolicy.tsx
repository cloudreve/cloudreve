import { Container } from "@mui/material";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useParams } from "react-router-dom";
import PageContainer from "../../../Pages/PageContainer";
import PageHeader from "../../../Pages/PageHeader";
import StoragePolicyForm from "./StoragePolicyForm";
import StoragePolicySettingWrapper from "./StoragePolicySettingWrapper";

const EditStoragePolicy = () => {
  const { t } = useTranslation("dashboard");
  const { id } = useParams();
  const [name, setName] = useState("");
  return (
    <PageContainer>
      <Container maxWidth="xl">
        <PageHeader title={t("policy.editX", { name })} />
        <StoragePolicySettingWrapper policyID={parseInt(id ?? "1") ?? 1} onPolicyChange={(p) => setName(p.name)}>
          <StoragePolicyForm />
        </StoragePolicySettingWrapper>
      </Container>
    </PageContainer>
  );
};

export default EditStoragePolicy;
