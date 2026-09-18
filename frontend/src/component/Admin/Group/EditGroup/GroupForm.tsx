import { Box, Stack } from "@mui/material";
import { useContext } from "react";
import { useTranslation } from "react-i18next";
import BasicInfoSection from "./BasicInfoSection";
import DefaultPinnedSection from "./DefaultPinnedSection";
import FileManagementSection from "./FileManagementSection";
import { GroupSettingContext } from "./GroupSettingWrapper";
import ShareSection from "./ShareSection";
import UploadDownloadSection from "./UploadDownloadSection";

const GroupForm = () => {
  const { t } = useTranslation("dashboard");
  const { formRef, values } = useContext(GroupSettingContext);
  return (
    <Box component={"form"} ref={formRef} onSubmit={(e) => e.preventDefault()}>
      <Stack spacing={5}>
        <BasicInfoSection />
        <ShareSection />
        <DefaultPinnedSection />
        <FileManagementSection />
        <UploadDownloadSection />
      </Stack>
    </Box>
  );
};

export default GroupForm;
