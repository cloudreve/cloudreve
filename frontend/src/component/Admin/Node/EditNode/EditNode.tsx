import { Container } from "@mui/material";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useParams } from "react-router-dom";
import { Node } from "../../../../api/dashboard";
import PageContainer from "../../../Pages/PageContainer";
import PageHeader from "../../../Pages/PageHeader";
import BasicInfoSection from "./BasicInfoSection";
import CapabilitiesSection from "./CapabilitiesSection";
import NodeForm from "./NodeForm";
import NodeSettingWrapper from "./NodeSettingWrapper";

const EditNode = () => {
  const { t } = useTranslation("dashboard");
  const { id } = useParams<{ id: string }>();
  const [node, setNode] = useState<Node | null>(null);
  const nodeID = parseInt(id ?? "0");

  return (
    <PageContainer>
      <Container maxWidth="xl">
        <PageHeader title={t("node.editNode", { node: node?.name })} />
        <NodeSettingWrapper nodeID={nodeID} onNodeChange={setNode}>
          <NodeForm>
            <BasicInfoSection />
            <CapabilitiesSection />
          </NodeForm>
        </NodeSettingWrapper>
      </Container>
    </PageContainer>
  );
};

export default EditNode;
