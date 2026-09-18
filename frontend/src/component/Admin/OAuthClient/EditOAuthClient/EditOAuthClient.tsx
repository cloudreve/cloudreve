import { Container } from "@mui/material";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useParams } from "react-router-dom";
import { GetOAuthClientResponse } from "../../../../api/dashboard";
import PageContainer from "../../../Pages/PageContainer";
import PageHeader from "../../../Pages/PageHeader";
import OAuthClientForm from "./OAuthClientForm";
import OAuthClientSettingWrapper from "./OAuthClientSettingWrapper";

const EditOAuthClient = () => {
  const { t } = useTranslation("dashboard");
  const { id } = useParams<{ id: string }>();
  const [client, setClient] = useState<GetOAuthClientResponse | null>(null);
  const clientID = parseInt(id ?? "0");
  const isNew = clientID === 0;

  return (
    <PageContainer>
      <Container maxWidth="xl">
        <PageHeader
          title={isNew ? t("oauth.newOAuthClient") : t("oauth.editOAuthClient", { name: t(client?.name ?? "") })}
        />
        <OAuthClientSettingWrapper clientID={clientID} onClientChange={setClient}>
          <OAuthClientForm />
        </OAuthClientSettingWrapper>
      </Container>
    </PageContainer>
  );
};

export default EditOAuthClient;
