import { Box, Stack } from "@mui/material";
import { useContext } from "react";
import { NodeSettingContext } from "./NodeSettingWrapper";

export interface NodeFormProps {
  children: React.ReactNode;
}

const NodeForm = ({ children }: NodeFormProps) => {
  const { formRef } = useContext(NodeSettingContext);

  return (
    <Box component="form" ref={formRef} noValidate>
      <Stack spacing={5}>{children}</Stack>
    </Box>
  );
};

export default NodeForm;
