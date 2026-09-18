import { Alert, Box, Link, Stack } from "@mui/material";
import { useContext } from "react";
import { Trans, useTranslation } from "react-i18next";
import { Link as RouterLink } from "react-router-dom";
import {
  BasicInfoSection,
  DownloadSection,
  EncryptionSection,
  MediaMetadataSection,
  StorageAndUploadSection,
  ThumbnailsSection,
} from "./FormSections";
import { StoragePolicySettingContext } from "./StoragePolicySettingWrapper";

const StoragePolicyForm = () => {
  const { t } = useTranslation("dashboard");
  const { formRef, values } = useContext(StoragePolicySettingContext);

  return (
    <Box component={"form"} ref={formRef} onSubmit={(e) => e.preventDefault()}>
      {!values.edges?.groups?.length && (
        <Alert severity="warning" sx={{ mb: 3 }}>
          <Trans
            i18nKey="policy.noBindedGroupWarning"
            ns="dashboard"
            components={[<Link component={RouterLink} to="/admin/group" />]}
          />
        </Alert>
      )}
      <Stack spacing={5}>
        <BasicInfoSection />
        <StorageAndUploadSection />
        <DownloadSection />
        <ThumbnailsSection />
        <MediaMetadataSection />
        <EncryptionSection />
      </Stack>
    </Box>
  );
};

export default StoragePolicyForm;
