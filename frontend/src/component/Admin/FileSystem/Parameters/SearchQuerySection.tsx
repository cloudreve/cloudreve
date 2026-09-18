import { Typography } from "@mui/material";
import { useContext } from "react";
import { useTranslation } from "react-i18next";
import { DenseFilledTextField } from "../../../Common/StyledComponents.tsx";
import SettingForm from "../../../Pages/Setting/SettingForm.tsx";
import { SettingSection, SettingSectionContent } from "../../Settings/Settings.tsx";
import { SettingContext } from "../../Settings/SettingWrapper.tsx";

const SearchQuerySection = () => {
  const { t } = useTranslation("dashboard");
  const { setSettings, values } = useContext(SettingContext);

  return (
    <SettingSection>
      <Typography variant="h6" gutterBottom>
        {t("settings.searchQuery")}
      </Typography>
      <SettingSectionContent>
        <SettingForm title={t("application:navbar.photos")}>
          <DenseFilledTextField
            fullWidth
            value={values.explorer_category_image_query}
            onChange={(e) =>
              setSettings({
                explorer_category_image_query: e.target.value,
              })
            }
            required
          />
        </SettingForm>
        <SettingForm title={t("application:navbar.videos")}>
          <DenseFilledTextField
            fullWidth
            value={values.explorer_category_video_query}
            onChange={(e) =>
              setSettings({
                explorer_category_video_query: e.target.value,
              })
            }
            required
          />
        </SettingForm>
        <SettingForm title={t("application:navbar.music")}>
          <DenseFilledTextField
            fullWidth
            value={values.explorer_category_audio_query}
            onChange={(e) =>
              setSettings({
                explorer_category_audio_query: e.target.value,
              })
            }
            required
          />
        </SettingForm>
        <SettingForm title={t("application:navbar.documents")}>
          <DenseFilledTextField
            fullWidth
            value={values.explorer_category_document_query}
            onChange={(e) =>
              setSettings({
                explorer_category_document_query: e.target.value,
              })
            }
            required
          />
        </SettingForm>
      </SettingSectionContent>
    </SettingSection>
  );
};

export default SearchQuerySection;
