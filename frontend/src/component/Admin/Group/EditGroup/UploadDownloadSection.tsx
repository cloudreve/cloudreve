import { Box, Collapse, FormControl, FormControlLabel, Stack, Switch, Typography } from "@mui/material";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { GroupEnt } from "../../../../api/dashboard";
import { GroupPermission } from "../../../../api/user";
import Boolset from "../../../../util/boolset";
import SizeInput from "../../../Common/SizeInput";
import { DenseFilledTextField } from "../../../Common/StyledComponents";
import SettingForm, { ProChip } from "../../../Pages/Setting/SettingForm";
import { NoMarginHelperText, SettingSection, SettingSectionContent } from "../../Settings/Settings";
import { AnonymousGroupID } from "../GroupRow";
import { GroupSettingContext } from "./GroupSettingWrapper";

const UploadDownloadSection = () => {
  const { t } = useTranslation("dashboard");
  const { values, setGroup } = useContext(GroupSettingContext);

  const permission = useMemo(() => {
    return new Boolset(values.permissions ?? "");
  }, [values.permissions]);

  const onAllowArchiveDownloadChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setGroup((p: GroupEnt) => ({
        ...p,
        permissions: new Boolset(p.permissions).set(GroupPermission.archive_download, e.target.checked).toString(),
      }));
    },
    [setGroup],
  );

  const onAllowDirectLinkChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setGroup((p: GroupEnt) => ({
        ...p,
        settings: {
          ...p.settings,
          source_batch: e.target.checked ? 1 : 0,
        },
      }));
    },
    [setGroup],
  );

  const onSourceBatchChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setGroup((p: GroupEnt) => ({
        ...p,
        settings: { ...p.settings, source_batch: parseInt(e.target.value) ? parseInt(e.target.value) : undefined },
      }));
    },
    [setGroup],
  );

  const onRedirectedSourceChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setGroup((p: GroupEnt) => ({
        ...p,
        settings: { ...p.settings, redirected_source: e.target.checked ? true : undefined },
      }));
    },
    [setGroup],
  );

  const onDownloadSpeedLimitChange = useCallback(
    (e: number) => {
      setGroup((p: GroupEnt) => ({ ...p, speed_limit: e ? e : undefined }));
    },
    [setGroup],
  );

  const onReuseDirectLinkChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setGroup((p: GroupEnt) => ({
        ...p,
        permissions: new Boolset(p.permissions).set(GroupPermission.unique_direct_link, !e.target.checked).toString(),
      }));
    },
    [setGroup],
  );

  return (
    <SettingSection>
      <Typography variant="h6" gutterBottom>
        {t("group.uploadDownload")}
      </Typography>
      <SettingSectionContent>
        <SettingForm lgWidth={5}>
          <FormControl fullWidth>
            <FormControlLabel
              control={
                <Switch
                  checked={permission.enabled(GroupPermission.archive_download)}
                  onChange={onAllowArchiveDownloadChange}
                />
              }
              label={t("group.serverSideBatchDownload")}
            />
            <NoMarginHelperText>{t("group.serverSideBatchDownloadDes")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>
        {values?.id != AnonymousGroupID && (
          <>
            <SettingForm lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel
                  control={
                    <Switch checked={(values?.settings?.source_batch ?? 0) > 0} onChange={onAllowDirectLinkChange} />
                  }
                  label={t("group.getDirectLink")}
                />
                <NoMarginHelperText>{t("group.getDirectLinkDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <Collapse in={(values?.settings?.source_batch ?? 0) > 0} unmountOnExit>
              <Stack spacing={3}>
                <SettingForm lgWidth={5} title={t("group.bathSourceLinkLimit")}>
                  <FormControl fullWidth>
                    <DenseFilledTextField
                      slotProps={{
                        htmlInput: {
                          type: "number",
                          min: 0,
                        },
                      }}
                      value={values?.settings?.source_batch ?? 0}
                      onChange={onSourceBatchChange}
                    />
                    <NoMarginHelperText>{t("group.bathSourceLinkLimitDes")}</NoMarginHelperText>
                  </FormControl>
                </SettingForm>
                <SettingForm lgWidth={5}>
                  <FormControl fullWidth>
                    <FormControlLabel
                      control={
                        <Switch
                          checked={values?.settings?.redirected_source ?? false}
                          onChange={onRedirectedSourceChange}
                        />
                      }
                      label={t("group.redirectedSource")}
                    />
                    <NoMarginHelperText>{t("group.redirectedSourceDes")}</NoMarginHelperText>
                  </FormControl>
                </SettingForm>
              </Stack>
              <Collapse in={values?.settings?.redirected_source} unmountOnExit>
                <Stack spacing={3}>
                  <SettingForm lgWidth={5}>
                    <FormControl fullWidth sx={{ mt: 3 }}>
                      <FormControlLabel
                        control={
                          <Switch
                            checked={!permission.enabled(GroupPermission.unique_direct_link)}
                            onChange={onReuseDirectLinkChange}
                          />
                        }
                        label={t("group.reuseDirectLink")}
                      />
                      <NoMarginHelperText>{t("group.reuseDirectLinkDes")}</NoMarginHelperText>
                    </FormControl>
                  </SettingForm>
                  <SettingForm lgWidth={5} pro>
                    <FormControl fullWidth>
                      <FormControlLabel
                        control={<Switch checked={false} />}
                        label={
                          <Box sx={{ display: "flex", alignItems: "center" }}>
                            {t("group.folderDirectLink")}
                            <ProChip size="small" label="Pro" />
                          </Box>
                        }
                      />
                      <NoMarginHelperText>{t("group.folderDirectLinkDes")}</NoMarginHelperText>
                    </FormControl>
                  </SettingForm>
                </Stack>
              </Collapse>
            </Collapse>
          </>
        )}
        <SettingForm lgWidth={5} title={t("group.downloadSpeedLimit")}>
          <FormControl fullWidth>
            <SizeInput
              suffix={"/s"}
              variant={"outlined"}
              value={values?.speed_limit ?? 0}
              onChange={onDownloadSpeedLimitChange}
            />
            <NoMarginHelperText>{t("group.downloadSpeedLimitDes")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>
      </SettingSectionContent>
    </SettingSection>
  );
};

export default UploadDownloadSection;
