import { Box, Collapse, FormControl, Link, ListItemText, Typography } from "@mui/material";
import * as React from "react";
import { useCallback, useContext } from "react";
import { Trans, useTranslation } from "react-i18next";
import SizeInput from "../../../Common/SizeInput.tsx";
import { DenseFilledTextField, DenseSelect } from "../../../Common/StyledComponents.tsx";
import { SquareMenuItem } from "../../../FileManager/ContextMenu/ContextMenu.tsx";
import SettingForm from "../../../Pages/Setting/SettingForm.tsx";
import { NoMarginHelperText, SettingSection, SettingSectionContent } from "../../Settings/Settings.tsx";
import { SettingContext } from "../../Settings/SettingWrapper.tsx";

const FileSystemSection = () => {
  const { t } = useTranslation("dashboard");
  const { setSettings, values } = useContext(SettingContext);

  const onMimeMappingChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setSettings({
      mime_mapping: e.target.value,
    });
  }, []);

  return (
    <SettingSection>
      <Typography variant="h6" gutterBottom>
        {t("nav.fileSystem")}
      </Typography>
      <SettingSectionContent>
        <SettingForm title={t("settings.textEditMaxSize")} lgWidth={5}>
          <FormControl>
            <SizeInput
              variant={"outlined"}
              required
              allowZero={false}
              value={parseInt(values.maxEditSize) ?? 0}
              onChange={(e) =>
                setSettings({
                  maxEditSize: e.toString(),
                })
              }
            />
            <NoMarginHelperText>{t("settings.textEditMaxSizeDes")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>
        <SettingForm title={t("settings.trashBinInterval")} lgWidth={5}>
          <FormControl fullWidth>
            <DenseFilledTextField
              value={values.cron_trash_bin_collect}
              onChange={(e) =>
                setSettings({
                  cron_trash_bin_collect: e.target.value,
                })
              }
              required
            />
            <NoMarginHelperText>
              <Trans
                i18nKey="settings.cronDes"
                values={{
                  des: t("settings.trashBinIntervalDes"),
                }}
                ns={"dashboard"}
                components={[<Link href="https://crontab.guru/" target="_blank" rel="noopener noreferrer" />]}
              />
            </NoMarginHelperText>
          </FormControl>
        </SettingForm>
        <SettingForm title={t("settings.entityCollectInterval")} lgWidth={5}>
          <FormControl fullWidth>
            <DenseFilledTextField
              value={values.cron_entity_collect}
              onChange={(e) =>
                setSettings({
                  cron_entity_collect: e.target.value,
                })
              }
              required
            />
            <NoMarginHelperText>
              <Trans
                i18nKey="settings.cronDes"
                ns={"dashboard"}
                values={{
                  des: t("settings.entityCollectIntervalDes"),
                }}
                components={[<Link href="https://crontab.guru/" target="_blank" rel="noopener noreferrer" />]}
              />
            </NoMarginHelperText>
          </FormControl>
        </SettingForm>
        <SettingForm title={t("settings.publicResourceMaxAge")} lgWidth={5}>
          <FormControl fullWidth>
            <DenseFilledTextField
              type="number"
              inputProps={{ min: 0, setp: 1 }}
              value={values.public_resource_maxage}
              onChange={(e) =>
                setSettings({
                  public_resource_maxage: e.target.value,
                })
              }
              required
            />
            <NoMarginHelperText>{t("settings.publicResourceMaxAgeDes")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>
        <SettingForm title={t("settings.defaultPagination")} lgWidth={5}>
          <FormControl fullWidth>
            <DenseSelect
              renderValue={(v) => (
                <ListItemText
                  slotProps={{
                    primary: { variant: "body2" },
                  }}
                >
                  {v == "0" ? t("settings.offsetPagination") : t("settings.cursorPagination")}
                </ListItemText>
              )}
              onChange={(e) =>
                setSettings({
                  use_cursor_pagination: e.target.value as string,
                })
              }
              MenuProps={{
                PaperProps: { sx: { maxWidth: 230 } },
                MenuListProps: {
                  sx: {
                    "& .MuiMenuItem-root": {
                      whiteSpace: "normal",
                    },
                  },
                },
              }}
              value={values.use_cursor_pagination}
            >
              <SquareMenuItem value={"0"}>
                <Box
                  sx={{
                    display: "flex",
                    flexDirection: "column",
                  }}
                >
                  <Typography variant={"body2"} fontWeight={600}>
                    {t("settings.offsetPagination")}
                  </Typography>
                  <Typography variant={"body2"} color={"textSecondary"}>
                    {t("settings.offsetPaginationDes")}
                  </Typography>
                </Box>
              </SquareMenuItem>
              <SquareMenuItem value={"1"}>
                <Box sx={{ display: "flex", flexDirection: "column" }}>
                  <Typography variant={"body2"} fontWeight={600}>
                    {t("settings.cursorPagination")}
                  </Typography>
                  <Typography variant={"body2"} color={"textSecondary"}>
                    {t("settings.cursorPaginationDes")}
                  </Typography>
                </Box>
              </SquareMenuItem>
            </DenseSelect>
            <NoMarginHelperText>{t("settings.defaultPaginationDes")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>
        <SettingForm title={t("settings.maxPageSize")} lgWidth={5}>
          <FormControl>
            <DenseFilledTextField
              type="number"
              inputProps={{ min: 0, setp: 1 }}
              value={values.max_page_size}
              onChange={(e) =>
                setSettings({
                  max_page_size: e.target.value,
                })
              }
              required
            />
            <NoMarginHelperText>{t("settings.maxPageSizeDes")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>
        <SettingForm title={t("settings.maxBatchSize")} lgWidth={5}>
          <FormControl>
            <DenseFilledTextField
              type="number"
              inputProps={{ min: 0, setp: 1 }}
              value={values.max_batched_file}
              onChange={(e) =>
                setSettings({
                  max_batched_file: e.target.value,
                })
              }
              required
            />
            <NoMarginHelperText>{t("settings.maxBatchSizeDes")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>
        <SettingForm title={t("settings.maxRecursiveSearch")} lgWidth={5}>
          <FormControl>
            <DenseFilledTextField
              type="number"
              inputProps={{ min: 0, setp: 1 }}
              value={values.max_recursive_searched_folder}
              onChange={(e) =>
                setSettings({
                  max_recursive_searched_folder: e.target.value,
                })
              }
              required
            />
            <NoMarginHelperText>{t("settings.maxRecursiveSearchDes")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>
        <SettingForm title={t("settings.mapProvider")} lgWidth={5}>
          <FormControl>
            <DenseSelect
              onChange={(e) =>
                setSettings({
                  map_provider: e.target.value as string,
                })
              }
              value={values.map_provider}
            >
              <SquareMenuItem value={"google"}>
                <ListItemText
                  slotProps={{
                    primary: { variant: "body2" },
                  }}
                >
                  {t("settings.mapGoogle")}
                </ListItemText>
              </SquareMenuItem>
              <SquareMenuItem value={"openstreetmap"}>
                <ListItemText
                  slotProps={{
                    primary: { variant: "body2" },
                  }}
                >
                  {t("settings.mapOpenStreetMap")}
                </ListItemText>
              </SquareMenuItem>
              <SquareMenuItem value={"mapbox"}>
                <ListItemText
                  slotProps={{
                    primary: { variant: "body2" },
                  }}
                >
                  {t("settings.mapboxMap")}
                </ListItemText>
              </SquareMenuItem>
            </DenseSelect>
            <NoMarginHelperText>{t("settings.mapProviderDes")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>
        <Collapse in={values.map_provider === "mapbox"} unmountOnExit>
          <SettingForm title={t("settings.mapboxAccessToken")} lgWidth={5}>
            <FormControl fullWidth>
              <DenseFilledTextField
                fullWidth
                required
                value={values.map_mapbox_ak ?? ""}
                onChange={(e) => setSettings({ map_mapbox_ak: e.target.value })}
              />
              <NoMarginHelperText>
                <Trans
                  i18nKey="settings.mapboxAccessTokenDes"
                  ns="dashboard"
                  components={[<Link href="https://account.mapbox.com/access-tokens" target="_blank" />]}
                />
              </NoMarginHelperText>
            </FormControl>
          </SettingForm>
        </Collapse>
        <Collapse in={values.map_provider === "google"} unmountOnExit>
          <SettingForm title={t("settings.tileType")} lgWidth={5}>
            <FormControl>
              <DenseSelect
                onChange={(e) =>
                  setSettings({
                    map_google_tile_type: e.target.value as string,
                  })
                }
                value={values.map_google_tile_type}
              >
                <SquareMenuItem value={"terrain"}>
                  <ListItemText
                    slotProps={{
                      primary: { variant: "body2" },
                    }}
                  >
                    {t("settings.tileTypeTerrain")}
                  </ListItemText>
                </SquareMenuItem>
                <SquareMenuItem value={"satellite"}>
                  <ListItemText
                    slotProps={{
                      primary: { variant: "body2" },
                    }}
                  >
                    {t("settings.tileTypeSatellite")}
                  </ListItemText>
                </SquareMenuItem>
                <SquareMenuItem value={"regular"}>
                  <ListItemText
                    slotProps={{
                      primary: { variant: "body2" },
                    }}
                  >
                    {t("settings.tileTypeGeneral")}
                  </ListItemText>
                </SquareMenuItem>
              </DenseSelect>
              <NoMarginHelperText>{t("settings.tileTypeDes")}</NoMarginHelperText>
            </FormControl>
          </SettingForm>
        </Collapse>
        <SettingForm title={t("settings.mimeMapping")} lgWidth={5}>
          <FormControl fullWidth>
            <DenseFilledTextField
              multiline
              rows={6}
              value={values.mime_mapping}
              onChange={onMimeMappingChange}
              required
            />
            <NoMarginHelperText>{t("settings.mimeMappingDes")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>
      </SettingSectionContent>
    </SettingSection>
  );
};

export default FileSystemSection;
