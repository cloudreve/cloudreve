import { Box, Stack, Typography } from "@mui/material";
import { useSnackbar } from "notistack";
import { useCallback, useContext, useState } from "react";
import { useTranslation } from "react-i18next";
import { useAppDispatch } from "../../../../redux/hooks";
import SettingForm from "../../../Pages/Setting/SettingForm";
import { SettingSection, SettingSectionContent } from "../../Settings/Settings";
import { SettingContext } from "../../Settings/SettingWrapper";
import DefaultViewerApps from "./DefaultViewerApps";
import FileViewerList from "./FileViewerList";

const ViewerSetting = () => {
  const { t } = useTranslation("dashboard");
  const { formRef, setSettings, values } = useContext(SettingContext);
  const [loading, setLoading] = useState(false);
  const dispatch = useAppDispatch();
  const { enqueueSnackbar } = useSnackbar();

  const viewerOnChange = useCallback(
    (s: string) =>
      setSettings({
        file_viewers: s,
      }),
    [],
  );

  const defaultAppsOnChange = useCallback(
    (s: string) =>
      setSettings({
        viewer_default_apps: s,
      }),
    [],
  );

  return (
    <Box component={"form"} ref={formRef} onSubmit={(e) => e.preventDefault()}>
      <Stack spacing={5}>
        <SettingSection>
          <Typography variant="h6" gutterBottom>
            {t("settings.viewerList")}
          </Typography>
          <SettingSectionContent>
            <FileViewerList config={values.file_viewers} onChange={viewerOnChange} />
          </SettingSectionContent>
        </SettingSection>
        <SettingSection>
          <Typography variant="h6" gutterBottom>
            {t("settings.defaultViewerApps")}
          </Typography>
          <Typography variant="body2" color="text.secondary" gutterBottom>
            {t("settings.defaultViewerAppsDes")}
          </Typography>
          <SettingSectionContent>
            <SettingForm lgWidth={5}>
              <DefaultViewerApps
                config={values.viewer_default_apps}
                fileViewers={values.file_viewers}
                onChange={defaultAppsOnChange}
              />
            </SettingForm>
          </SettingSectionContent>
        </SettingSection>
      </Stack>
    </Box>
  );
};

export default ViewerSetting;
