import { Box, Stack } from "@mui/material";
import { useContext } from "react";
import { SettingContext } from "../../Settings/SettingWrapper.tsx";
import AdvancedOptionsSection from "./AdvancedOptionsSection.tsx";
import FileEncryptionSection from "./FileEncryptionSection.tsx";
import FileEventPushSection from "./FileEventPushSection.tsx";
import FileSystemSection from "./FileSystemSection.tsx";
import SearchQuerySection from "./SearchQuerySection.tsx";

const Parameters = () => {
  const { formRef } = useContext(SettingContext);

  return (
    <Box component={"form"} ref={formRef} onSubmit={(e) => e.preventDefault()}>
      <Stack spacing={5}>
        <FileSystemSection />
        <SearchQuerySection />
        <FileEncryptionSection />
        <FileEventPushSection />
        <AdvancedOptionsSection />
      </Stack>
    </Box>
  );
};

export default Parameters;
