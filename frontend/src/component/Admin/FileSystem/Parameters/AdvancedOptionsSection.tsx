import { DeleteOutline } from "@mui/icons-material";
import { Box, FormControl, FormControlLabel, Link, Switch, Typography } from "@mui/material";
import { useSnackbar } from "notistack";
import { useContext, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { sendClearBlobUrlCache } from "../../../../api/api.ts";
import { useAppDispatch } from "../../../../redux/hooks.ts";
import { isTrueVal } from "../../../../session/utils.ts";
import { DefaultCloseAction } from "../../../Common/Snackbar/snackbar.tsx";
import { DenseFilledTextField, SecondaryButton } from "../../../Common/StyledComponents.tsx";
import SettingForm from "../../../Pages/Setting/SettingForm.tsx";
import { NoMarginHelperText, SettingSection, SettingSectionContent } from "../../Settings/Settings.tsx";
import { SettingContext } from "../../Settings/SettingWrapper.tsx";

const AdvancedOptionsSection = () => {
  const { t } = useTranslation("dashboard");
  const { setSettings, values } = useContext(SettingContext);
  const [loading, setLoading] = useState(false);
  const dispatch = useAppDispatch();
  const { enqueueSnackbar } = useSnackbar();

  const clearBlobUrlCache = () => {
    setLoading(true);
    dispatch(sendClearBlobUrlCache())
      .then(() => {
        setLoading(false);
        enqueueSnackbar(t("settings.cacheCleared"), { variant: "success", action: DefaultCloseAction });
      })
      .catch(() => {
        setLoading(false);
      });
  };

  return (
    <SettingSection>
      <Typography variant="h6" gutterBottom>
        {t("settings.advanceOptions")}
      </Typography>
      <SettingSectionContent>
        <SettingForm title={t("settings.archiveTimeout")} lgWidth={5}>
          <DenseFilledTextField
            type="number"
            inputProps={{ min: 1, setp: 1 }}
            value={values.archive_timeout}
            onChange={(e) =>
              setSettings({
                archive_timeout: e.target.value,
              })
            }
            required
          />
        </SettingForm>
        <SettingForm title={t("settings.uploadSessionTimeout")} lgWidth={5}>
          <DenseFilledTextField
            type="number"
            inputProps={{ min: 1, setp: 1 }}
            value={values.upload_session_timeout}
            onChange={(e) =>
              setSettings({
                upload_session_timeout: e.target.value,
              })
            }
            required
          />
          <NoMarginHelperText>{t("settings.uploadSessionDes")}</NoMarginHelperText>
        </SettingForm>
        <SettingForm title={t("settings.slaveAPIExpiration")} lgWidth={5}>
          <DenseFilledTextField
            type="number"
            inputProps={{ min: 1, setp: 1 }}
            value={values.slave_api_timeout}
            onChange={(e) =>
              setSettings({
                slave_api_timeout: e.target.value,
              })
            }
            required
          />
          <NoMarginHelperText>{t("settings.slaveAPIExpirationDes")}</NoMarginHelperText>
        </SettingForm>
        <SettingForm title={t("settings.folderPropsTimeout")} lgWidth={5}>
          <DenseFilledTextField
            type="number"
            inputProps={{ min: 1, setp: 1 }}
            value={values.folder_props_timeout}
            onChange={(e) =>
              setSettings({
                folder_props_timeout: e.target.value,
              })
            }
            required
          />
          <NoMarginHelperText>{t("settings.folderPropsTimeoutDes")}</NoMarginHelperText>
        </SettingForm>
        <SettingForm title={t("settings.failedChunkRetry")} lgWidth={5}>
          <DenseFilledTextField
            type="number"
            inputProps={{ min: 0, setp: 1 }}
            value={values.chunk_retries}
            onChange={(e) =>
              setSettings({
                chunk_retries: e.target.value,
              })
            }
            required
          />
          <NoMarginHelperText>{t("settings.failedChunkRetryDes")}</NoMarginHelperText>
        </SettingForm>
        <SettingForm lgWidth={5}>
          <FormControl fullWidth>
            <FormControlLabel
              control={
                <Switch
                  checked={isTrueVal(values.use_temp_chunk_buffer)}
                  onChange={(e) =>
                    setSettings({
                      use_temp_chunk_buffer: e.target.checked ? "1" : "0",
                    })
                  }
                />
              }
              label={t("settings.cacheChunks")}
            />
            <NoMarginHelperText>{t("settings.cacheChunksDes")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>
        <SettingForm title={t("settings.transitParallelNum")} lgWidth={5}>
          <DenseFilledTextField
            type="number"
            inputProps={{ min: 1, setp: 1 }}
            value={values.max_parallel_transfer}
            onChange={(e) =>
              setSettings({
                max_parallel_transfer: e.target.value,
              })
            }
            required
          />
          <NoMarginHelperText>{t("settings.transitParallelNumDes")}</NoMarginHelperText>
        </SettingForm>
        <SettingForm title={t("settings.oauthRefresh")} lgWidth={5}>
          <DenseFilledTextField
            fullWidth
            required
            value={values.cron_oauth_cred_refresh}
            onChange={(e) =>
              setSettings({
                cron_oauth_cred_refresh: e.target.value,
              })
            }
          />
          <NoMarginHelperText>
            <Trans
              i18nKey="settings.cronDes"
              values={{
                des: t("settings.oauthRefreshDes"),
              }}
              ns={"dashboard"}
              components={[<Link href="https://crontab.guru/" target="_blank" rel="noopener noreferrer" />]}
            />
          </NoMarginHelperText>
        </SettingForm>
        <SettingForm title={t("settings.wopiSessionTimeout")} lgWidth={5}>
          <DenseFilledTextField
            type="number"
            inputProps={{ min: 1, setp: 1 }}
            value={values.viewer_session_timeout}
            onChange={(e) =>
              setSettings({
                viewer_session_timeout: e.target.value,
              })
            }
            required
          />
          <NoMarginHelperText>{t("settings.wopiSessionTimeoutDes")}</NoMarginHelperText>
        </SettingForm>
        <SettingForm title={t("settings.fileBlobTimeout")} lgWidth={5}>
          <DenseFilledTextField
            type="number"
            inputProps={{ min: 1, setp: 1 }}
            value={values.entity_url_default_ttl}
            onChange={(e) =>
              setSettings({
                entity_url_default_ttl: e.target.value,
              })
            }
            required
          />
          <NoMarginHelperText>{t("settings.fileBlobTimeoutDes")}</NoMarginHelperText>
        </SettingForm>
        <SettingForm title={t("settings.fileBlobMargin")} lgWidth={5}>
          <DenseFilledTextField
            type="number"
            inputProps={{ min: 1, setp: 1 }}
            value={values.entity_url_cache_margin}
            onChange={(e) =>
              setSettings({
                entity_url_cache_margin: e.target.value,
              })
            }
            required
          />
          <NoMarginHelperText>{t("settings.fileBlobMarginDes")}</NoMarginHelperText>
        </SettingForm>
        <SettingForm title={t("settings.blobUrlCache")} lgWidth={5} anchorId="clearBlobUrlCache">
          <FormControl fullWidth>
            <Box>
              <SecondaryButton
                startIcon={<DeleteOutline />}
                variant="contained"
                loading={loading}
                color="primary"
                onClick={clearBlobUrlCache}
              >
                {t("settings.clearBlobUrlCache")}
              </SecondaryButton>
            </Box>
            <NoMarginHelperText>{t("settings.clearBlobUrlCacheDes")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>
      </SettingSectionContent>
    </SettingSection>
  );
};

export default AdvancedOptionsSection;
