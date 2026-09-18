import { Box, Container } from "@mui/material";
import { useQueryState } from "nuqs";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { CSSTransition, SwitchTransition } from "react-transition-group";
import ResponsiveTabs, { Tab } from "../../Common/ResponsiveTabs.tsx";
import AppGeneric from "../../Icons/AppGeneric.tsx";
import Icons from "../../Icons/Icons.tsx";
import Search from "../../Icons/Search.tsx";
import SettingsOutlined from "../../Icons/SettingsOutlined.tsx";
import TextBulletListSquareEdit from "../../Icons/TextBulletListSquareEdit.tsx";
import PageContainer from "../../Pages/PageContainer.tsx";
import PageHeader, { PageTabQuery } from "../../Pages/PageHeader.tsx";
import SettingsWrapper from "../Settings/SettingWrapper.tsx";
import CustomPropsSetting from "./CustomProps/CustomPropsSetting.tsx";
import FullTextSearchSetting from "./FullTextSearch/FullTextSearchSetting.tsx";
import FileIcons from "./Icons/FileIcons.tsx";
import Parameters from "./Parameters/Parameters.tsx";
import ViewerSetting from "./ViewerSetting/ViewerSetting.tsx";

export enum SettingsPageTab {
  Parameters = "parameters",
  FullTextSearch = "fullTextSearch",
  CustomProps = "customProps",
  Icon = "icon",
  FileApp = "fileApp",
}

const FileSystem = () => {
  const { t } = useTranslation("dashboard");
  const [tab, setTab] = useQueryState(PageTabQuery);

  const tabs: Tab<SettingsPageTab>[] = useMemo(() => {
    const res = [];
    res.push(
      ...[
        {
          label: t("nav.settings"),
          value: SettingsPageTab.Parameters,
          icon: <SettingsOutlined />,
        },
        {
          label: t("nav.fullTextSearch"),
          value: SettingsPageTab.FullTextSearch,
          icon: <Search />,
        },
        {
          label: t("settings.fileIcons"),
          value: SettingsPageTab.Icon,
          icon: <Icons />,
        },
        {
          label: t("settings.fileViewers"),
          value: SettingsPageTab.FileApp,
          icon: <AppGeneric />,
        },
        {
          label: t("nav.customProps"),
          value: SettingsPageTab.CustomProps,
          icon: <TextBulletListSquareEdit />,
        },
      ],
    );
    return res;
  }, [t]);

  return (
    <PageContainer>
      <Container maxWidth="xl">
        <PageHeader title={t("dashboard:nav.fileSystem")} />
        <ResponsiveTabs
          value={tab ?? SettingsPageTab.Parameters}
          onChange={(_e, newValue) => setTab(newValue)}
          tabs={tabs}
        />
        <SwitchTransition>
          <CSSTransition
            addEndListener={(node, done) => node.addEventListener("transitionend", done, false)}
            classNames="fade"
            key={`${tab}`}
          >
            <Box>
              {(!tab || tab === SettingsPageTab.Parameters) && (
                <SettingsWrapper
                  settings={[
                    "maxEditSize",
                    "cron_trash_bin_collect",
                    "cron_entity_collect",
                    "public_resource_maxage",
                    "use_cursor_pagination",
                    "max_page_size",
                    "max_recursive_searched_folder",
                    "max_batched_file",
                    "map_provider",
                    "map_google_tile_type",
                    "map_mapbox_ak",
                    "mime_mapping",
                    "explorer_category_image_query",
                    "explorer_category_video_query",
                    "explorer_category_audio_query",
                    "explorer_category_document_query",
                    "archive_timeout",
                    "upload_session_timeout",
                    "slave_api_timeout",
                    "folder_props_timeout",
                    "chunk_retries",
                    "use_temp_chunk_buffer",
                    "max_parallel_transfer",
                    "cron_oauth_cred_refresh",
                    "viewer_session_timeout",
                    "entity_url_default_ttl",
                    "entity_url_cache_margin",
                    "encrypt_master_key_vault",
                    "encrypt_master_key_file",
                    "show_encryption_status",
                    "fs_event_push_enabled",
                    "fs_event_push_max_age",
                    "fs_event_push_debounce",
                  ]}
                >
                  <Parameters />
                </SettingsWrapper>
              )}
              {tab === SettingsPageTab.FullTextSearch && (
                <SettingsWrapper
                  settings={[
                    "fts_enabled",
                    "fts_meilisearch_endpoint",
                    "fts_meilisearch_api_key",
                    "fts_meilisearch_page_size",
                    "fts_meilisearch_embed_enabled",
                    "fts_meilisearch_embed_config",
                    "fts_chunk_size",
                    "fts_tika_endpoint",
                    "fts_tika_exts",
                    "fts_tika_max_file_size",
                  ]}
                >
                  <FullTextSearchSetting />
                </SettingsWrapper>
              )}
              {tab === SettingsPageTab.Icon && (
                <SettingsWrapper settings={["explorer_icons", "emojis"]}>
                  <FileIcons />
                </SettingsWrapper>
              )}
              {tab === SettingsPageTab.FileApp && (
                <SettingsWrapper settings={["file_viewers", "viewer_default_apps"]}>
                  <ViewerSetting />
                </SettingsWrapper>
              )}
              {tab === SettingsPageTab.CustomProps && (
                <SettingsWrapper settings={["custom_props"]}>
                  <CustomPropsSetting />
                </SettingsWrapper>
              )}
            </Box>
          </CSSTransition>
        </SwitchTransition>
      </Container>
    </PageContainer>
  );
};

export default FileSystem;
