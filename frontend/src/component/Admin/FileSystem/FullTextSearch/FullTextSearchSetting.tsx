import { BuildOutlined } from "@mui/icons-material";
import {
  Box,
  CircularProgress,
  Collapse,
  FormControl,
  FormControlLabel,
  IconButton,
  Stack,
  Switch,
  Tooltip,
  Typography,
  useTheme,
} from "@mui/material";
import { useSnackbar } from "notistack";
import { lazy, Suspense, useContext, useState } from "react";
import { useTranslation } from "react-i18next";
import { sendRebuildFTSIndex } from "../../../../api/api.ts";
import { useAppDispatch } from "../../../../redux/hooks.ts";
import { confirmOperation } from "../../../../redux/thunks/dialog.ts";
import { isTrueVal } from "../../../../session/utils.ts";
import SizeInput from "../../../Common/SizeInput.tsx";
import { DefaultCloseAction } from "../../../Common/Snackbar/snackbar.tsx";
import { DenseFilledTextField, SecondaryButton } from "../../../Common/StyledComponents.tsx";
import QuestionCircle from "../../../Icons/QuestionCircle.tsx";
import SettingForm from "../../../Pages/Setting/SettingForm.tsx";
import { NoMarginHelperText, SettingSection, SettingSectionContent } from "../../Settings/Settings.tsx";
import { SettingContext } from "../../Settings/SettingWrapper.tsx";

const MonacoEditor = lazy(() => import("../../../Viewers/CodeViewer/MonacoEditor"));

const FullTextSearchSetting = () => {
  const { t } = useTranslation("dashboard");
  const { setSettings, values, formRef } = useContext(SettingContext);
  const dispatch = useAppDispatch();
  const { enqueueSnackbar } = useSnackbar();
  const theme = useTheme();
  const ftsEnabled = isTrueVal(values.fts_enabled);
  const [rebuildLoading, setRebuildLoading] = useState(false);

  return (
    <Box component={"form"} ref={formRef} onSubmit={(e) => e.preventDefault()}>
      <Stack spacing={5}>
        {/* Master Switch */}
        <SettingSection>
          <Typography variant="h6" gutterBottom>
            {t("settings.ftsTitle")}
            <IconButton
              onClick={() => {
                window.open("https://docs.cloudreve.org/usage/search/fts", "_blank");
              }}
            >
              <QuestionCircle />
            </IconButton>
          </Typography>
          <SettingSectionContent>
            <SettingForm lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel
                  control={
                    <Switch
                      checked={ftsEnabled}
                      onChange={(e) =>
                        setSettings({
                          fts_enabled: e.target.checked ? "1" : "0",
                        })
                      }
                    />
                  }
                  label={t("settings.ftsEnable")}
                />
                <NoMarginHelperText>{t("settings.ftsEnableDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
          </SettingSectionContent>
        </SettingSection>

        <Collapse in={ftsEnabled} unmountOnExit>
          <Stack spacing={5}>
            {/* Indexer (Meilisearch) Section */}
            <SettingSection>
              <Typography variant="h6" gutterBottom>
                {t("settings.ftsIndexer")}
              </Typography>
              <SettingSectionContent>
                <SettingForm title={t("settings.ftsMeilisearchEndpoint")} lgWidth={5}>
                  <DenseFilledTextField
                    fullWidth
                    required
                    placeholder="http://localhost:7700"
                    value={values.fts_meilisearch_endpoint}
                    onChange={(e) =>
                      setSettings({
                        fts_meilisearch_endpoint: e.target.value,
                      })
                    }
                  />
                  <NoMarginHelperText>{t("settings.ftsMeilisearchEndpointDes")}</NoMarginHelperText>
                </SettingForm>
                <SettingForm title={t("settings.ftsMeilisearchApiKey")} lgWidth={5}>
                  <DenseFilledTextField
                    fullWidth
                    required
                    type="password"
                    value={values.fts_meilisearch_api_key}
                    onChange={(e) =>
                      setSettings({
                        fts_meilisearch_api_key: e.target.value,
                      })
                    }
                  />
                  <NoMarginHelperText>{t("settings.ftsMeilisearchApiKeyDes")}</NoMarginHelperText>
                </SettingForm>
                <SettingForm title={t("settings.ftsMeilisearchPageSize")} lgWidth={5}>
                  <DenseFilledTextField
                    type="number"
                    inputProps={{ min: 1, step: 1 }}
                    value={values.fts_meilisearch_page_size}
                    onChange={(e) =>
                      setSettings({
                        fts_meilisearch_page_size: e.target.value,
                      })
                    }
                    required
                  />
                  <NoMarginHelperText>{t("settings.ftsMeilisearchPageSizeDes")}</NoMarginHelperText>
                </SettingForm>
                <SettingForm lgWidth={5}>
                  <FormControl fullWidth>
                    <FormControlLabel
                      control={
                        <Switch
                          checked={isTrueVal(values.fts_meilisearch_embed_enabled)}
                          onChange={(e) =>
                            setSettings({
                              fts_meilisearch_embed_enabled: e.target.checked ? "1" : "0",
                            })
                          }
                        />
                      }
                      label={t("settings.ftsMeilisearchAISearch")}
                    />
                    <NoMarginHelperText>{t("settings.ftsMeilisearchAISearchDes")}</NoMarginHelperText>
                  </FormControl>
                </SettingForm>
                <Collapse in={isTrueVal(values.fts_meilisearch_embed_enabled)} unmountOnExit>
                  <SettingForm title={t("settings.ftsMeilisearchEmbedConfig")} lgWidth={9}>
                    <Suspense fallback={<CircularProgress />}>
                      <MonacoEditor
                        theme={theme.palette.mode === "dark" ? "vs-dark" : "vs"}
                        language="json"
                        value={values.fts_meilisearch_embed_config}
                        onChange={(value) =>
                          setSettings({
                            fts_meilisearch_embed_config: value || "{}",
                          })
                        }
                        height="200px"
                        minHeight="200px"
                        options={{
                          wordWrap: "on",
                          minimap: { enabled: false },
                          scrollBeyondLastLine: false,
                        }}
                      />
                    </Suspense>
                    <NoMarginHelperText>{t("settings.ftsMeilisearchEmbedConfigDes")}</NoMarginHelperText>
                  </SettingForm>
                </Collapse>

                {/* Action Buttons */}
                <SettingForm title={t("settings.ftsMeilisearchActions")} lgWidth={5}>
                  <FormControl fullWidth>
                    <Box sx={{ display: "flex", gap: 1 }}>
                      <Tooltip title={t("settings.ftsRebuildIndexTooltip")}>
                        <SecondaryButton
                          startIcon={<BuildOutlined />}
                          variant="contained"
                          color="primary"
                          disabled={rebuildLoading}
                          onClick={() => {
                            dispatch(confirmOperation(t("settings.ftsRebuildIndexConfirm"))).then(() => {
                              setRebuildLoading(true);
                              dispatch(sendRebuildFTSIndex({}))
                                .then(() => {
                                  enqueueSnackbar(t("settings.ftsRebuildIndexSubmitted"), {
                                    variant: "success",
                                    action: DefaultCloseAction,
                                  });
                                })
                                .finally(() => {
                                  setRebuildLoading(false);
                                });
                            });
                          }}
                        >
                          {t("settings.ftsRebuildIndex")}
                        </SecondaryButton>
                      </Tooltip>
                    </Box>
                    <NoMarginHelperText>{t("settings.ftsMeilisearchActionsDes")}</NoMarginHelperText>
                  </FormControl>
                </SettingForm>
              </SettingSectionContent>
            </SettingSection>

            {/* Extractor (Tika) Section */}
            <SettingSection>
              <Typography variant="h6" gutterBottom>
                {t("settings.ftsExtractor")}
              </Typography>
              <SettingSectionContent>
                <SettingForm title={t("settings.ftsTikaEndpoint")} lgWidth={5}>
                  <DenseFilledTextField
                    fullWidth
                    placeholder="http://localhost:9998"
                    value={values.fts_tika_endpoint}
                    onChange={(e) =>
                      setSettings({
                        fts_tika_endpoint: e.target.value,
                      })
                    }
                  />
                  <NoMarginHelperText>{t("settings.ftsTikaEndpointDes")}</NoMarginHelperText>
                </SettingForm>
                <SettingForm title={t("settings.ftsTikaExts")} lgWidth={5}>
                  <DenseFilledTextField
                    fullWidth
                    value={values.fts_tika_exts}
                    onChange={(e) =>
                      setSettings({
                        fts_tika_exts: e.target.value,
                      })
                    }
                  />
                  <NoMarginHelperText>{t("settings.ftsTikaExtsDes")}</NoMarginHelperText>
                </SettingForm>
                <SettingForm title={t("settings.ftsTikaMaxFileSize")} lgWidth={5}>
                  <FormControl>
                    <SizeInput
                      variant={"outlined"}
                      required
                      allowZero={false}
                      value={parseInt(values.fts_tika_max_file_size) || 0}
                      onChange={(e) =>
                        setSettings({
                          fts_tika_max_file_size: e.toString(),
                        })
                      }
                    />
                  </FormControl>
                  <NoMarginHelperText>{t("settings.ftsTikaMaxFileSizeDes")}</NoMarginHelperText>
                </SettingForm>
              </SettingSectionContent>
            </SettingSection>

            {/* Chunker Section */}
            <SettingSection>
              <Typography variant="h6" gutterBottom>
                {t("settings.ftsChunker")}
              </Typography>
              <SettingSectionContent>
                <SettingForm title={t("settings.ftsChunkSize")} lgWidth={5}>
                  <FormControl>
                    <SizeInput
                      variant={"outlined"}
                      required
                      allowZero={false}
                      value={parseInt(values.fts_chunk_size) || 0}
                      onChange={(e) =>
                        setSettings({
                          fts_chunk_size: e.toString(),
                        })
                      }
                    />
                  </FormControl>
                  <NoMarginHelperText>{t("settings.ftsChunkSizeDes")}</NoMarginHelperText>
                </SettingForm>
              </SettingSectionContent>
            </SettingSection>
          </Stack>
        </Collapse>
      </Stack>
    </Box>
  );
};

export default FullTextSearchSetting;
