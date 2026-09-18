import {
  Box,
  CircularProgress,
  Collapse,
  FormControl,
  FormControlLabel,
  Link,
  Stack,
  Switch,
  Typography,
  useTheme,
} from "@mui/material";
import { lazy, Suspense, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { Link as RouterLink } from "react-router-dom";
import { GroupEnt } from "../../../../api/dashboard";
import { GroupPermission } from "../../../../api/user";
import Boolset from "../../../../util/boolset";
import SizeInput from "../../../Common/SizeInput";
import { DenseFilledTextField } from "../../../Common/StyledComponents";
import SettingForm, { ProChip } from "../../../Pages/Setting/SettingForm";
import ProDialog from "../../Common/ProDialog";
import { NoMarginHelperText, SettingSection, SettingSectionContent } from "../../Settings/Settings";
import { AnonymousGroupID } from "../GroupRow";
import { GroupSettingContext } from "./GroupSettingWrapper";
import MultipleNodeSelectionInput from "./MultipleNodeSelectionInput";

const MonacoEditor = lazy(() => import("../../../Viewers/CodeViewer/MonacoEditor"));

const FileManagementSection = () => {
  const { t } = useTranslation("dashboard");
  const { values, setGroup } = useContext(GroupSettingContext);
  const [proOpen, setProOpen] = useState(false);
  const theme = useTheme();

  const [editedConfig, setEditedConfig] = useState("");

  const permission = useMemo(() => {
    return new Boolset(values.permissions ?? "");
  }, [values.permissions]);

  useEffect(() => {
    setEditedConfig(
      values.settings?.remote_download_options ? JSON.stringify(values.settings?.remote_download_options, null, 2) : "",
    );
  }, [values.settings?.remote_download_options]);

  const onAllowWabDAVChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setGroup((p: GroupEnt) => ({
        ...p,
        permissions: new Boolset(p.permissions).set(GroupPermission.webdav, e.target.checked).toString(),
      }));
    },
    [setGroup],
  );

  const onAllowWabDAVProxyChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setGroup((p: GroupEnt) => ({
        ...p,
        permissions: new Boolset(p.permissions).set(GroupPermission.webdav_proxy, e.target.checked).toString(),
      }));
    },
    [setGroup],
  );

  const onAllowCompressTaskChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setGroup((p: GroupEnt) => ({
        ...p,
        permissions: new Boolset(p.permissions).set(GroupPermission.archive_task, e.target.checked).toString(),
      }));
    },
    [setGroup],
  );

  const onCompressSizeChange = useCallback(
    (e: number) => {
      setGroup((p: GroupEnt) => ({ ...p, settings: { ...p.settings, compress_size: e ? e : undefined } }));
    },
    [setGroup],
  );

  const onDecompressSizeChange = useCallback(
    (e: number) => {
      setGroup((p: GroupEnt) => ({ ...p, settings: { ...p.settings, decompress_size: e ? e : undefined } }));
    },
    [setGroup],
  );

  const onAllowRemoteDownloadChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setGroup((p: GroupEnt) => ({
        ...p,
        permissions: new Boolset(p.permissions).set(GroupPermission.remote_download, e.target.checked).toString(),
      }));
    },
    [setGroup],
  );

  const onEditedConfigBlur = useCallback(
    (value: string) => {
      var res: Record<string, any> | undefined = undefined;
      if (value) {
        try {
          res = JSON.parse(value);
        } catch (e) {
          console.error(e);
        }
      }
      setGroup((p: GroupEnt) => ({ ...p, settings: { ...p.settings, remote_download_options: res } }));
    },
    [editedConfig, setGroup],
  );

  const onAria2BatchSizeChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setGroup((p: GroupEnt) => ({
        ...p,
        settings: { ...p.settings, aria2_batch: parseInt(e.target.value) ? parseInt(e.target.value) : undefined },
      }));
    },
    [setGroup],
  );

  const onAllowAdvanceDeleteChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setGroup((p: GroupEnt) => ({
        ...p,
        permissions: new Boolset(p.permissions).set(GroupPermission.advance_delete, e.target.checked).toString(),
      }));
    },
    [setGroup],
  );

  const onMaxWalkedFilesChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setGroup((p: GroupEnt) => ({
        ...p,
        settings: { ...p.settings, max_walked_files: parseInt(e.target.value) ? parseInt(e.target.value) : undefined },
      }));
    },
    [setGroup],
  );

  const onTrashBinDurationChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setGroup((p: GroupEnt) => ({
        ...p,
        settings: { ...p.settings, trash_retention: parseInt(e.target.value) ? parseInt(e.target.value) : undefined },
      }));
    },
    [setGroup],
  );

  const onProClick = useCallback((e: React.MouseEvent<HTMLDivElement>) => {
    e.stopPropagation();
    setProOpen(true);
  }, []);

  return (
    <SettingSection>
      <ProDialog open={proOpen} onClose={() => setProOpen(false)} />
      <Typography variant="h6" gutterBottom>
        {t("group.fileManagement")}
      </Typography>
      <SettingSectionContent>
        {values?.id != AnonymousGroupID && (
          <>
            <SettingForm lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel
                  control={
                    <Switch checked={permission.enabled(GroupPermission.webdav)} onChange={onAllowWabDAVChange} />
                  }
                  label={t("group.allowWabDAV")}
                />
                <NoMarginHelperText>{t("group.allowWabDAVDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <Collapse in={permission.enabled(GroupPermission.webdav)} unmountOnExit>
              <SettingForm lgWidth={5}>
                <FormControl fullWidth>
                  <FormControlLabel
                    control={
                      <Switch
                        checked={permission.enabled(GroupPermission.webdav_proxy)}
                        onChange={onAllowWabDAVProxyChange}
                      />
                    }
                    label={t("group.allowWabDAVProxy")}
                  />
                  <NoMarginHelperText>{t("group.allowWabDAVProxyDes")}</NoMarginHelperText>
                </FormControl>
              </SettingForm>
            </Collapse>
            <SettingForm lgWidth={5}>
              <FormControl fullWidth onClick={onProClick}>
                <FormControlLabel
                  control={<Switch checked={false} />}
                  label={
                    <Box sx={{ display: "flex", alignItems: "center" }}>
                      {t("group.migratePolicy")}
                      <ProChip size="small" label="Pro" />
                    </Box>
                  }
                />
                <NoMarginHelperText>{t("group.migratePolicyDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel
                  control={
                    <Switch
                      checked={permission.enabled(GroupPermission.archive_task)}
                      onChange={onAllowCompressTaskChange}
                    />
                  }
                  label={t("group.compressTask")}
                />
                <NoMarginHelperText>{t("group.compressTaskDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <Collapse in={permission.enabled(GroupPermission.archive_task)} unmountOnExit>
              <Stack spacing={3}>
                <SettingForm title={t("group.compressSize")} lgWidth={5}>
                  <FormControl fullWidth>
                    <SizeInput
                      variant={"outlined"}
                      required
                      value={values.settings?.compress_size ?? 0}
                      onChange={onCompressSizeChange}
                    />
                    <NoMarginHelperText>{t("group.compressSizeDes")}</NoMarginHelperText>
                  </FormControl>
                </SettingForm>
                <SettingForm title={t("group.decompressSize")} lgWidth={5}>
                  <FormControl fullWidth>
                    <SizeInput
                      variant={"outlined"}
                      required
                      value={values.settings?.decompress_size ?? 0}
                      onChange={onDecompressSizeChange}
                    />
                    <NoMarginHelperText>{t("group.decompressSizeDes")}</NoMarginHelperText>
                  </FormControl>
                </SettingForm>
              </Stack>
            </Collapse>
            <SettingForm lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel
                  control={
                    <Switch
                      checked={permission.enabled(GroupPermission.remote_download)}
                      onChange={onAllowRemoteDownloadChange}
                    />
                  }
                  label={t("group.allowRemoteDownload")}
                />
                <NoMarginHelperText>
                  <Trans
                    ns="dashboard"
                    i18nKey="group.allowRemoteDownloadDes"
                    components={[<Link component={RouterLink} to="/admin/node" />]}
                  />
                </NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <Collapse in={permission.enabled(GroupPermission.remote_download)} unmountOnExit>
              <Stack spacing={3}>
                <SettingForm title={t("group.aria2Options")} lgWidth={5}>
                  <FormControl fullWidth>
                    <Suspense fallback={<CircularProgress />}>
                      <MonacoEditor
                        theme={theme.palette.mode === "dark" ? "vs-dark" : "vs"}
                        language="json"
                        value={editedConfig}
                        onChange={(value) => setEditedConfig(value || "")}
                        onBlur={onEditedConfigBlur}
                        height="200px"
                        minHeight="200px"
                        options={{
                          wordWrap: "on",
                          minimap: { enabled: false },
                          scrollBeyondLastLine: false,
                        }}
                      />
                    </Suspense>
                    <NoMarginHelperText>{t("group.aria2OptionsDes")}</NoMarginHelperText>
                  </FormControl>
                </SettingForm>
                <SettingForm title={t("group.aria2BatchSize")} lgWidth={5}>
                  <FormControl fullWidth>
                    <DenseFilledTextField
                      slotProps={{
                        htmlInput: {
                          type: "number",
                          min: 0,
                        },
                      }}
                      value={values.settings?.aria2_batch ?? 0}
                      onChange={onAria2BatchSizeChange}
                    />
                    <NoMarginHelperText>{t("group.aria2BatchSizeDes")}</NoMarginHelperText>
                  </FormControl>
                </SettingForm>
              </Stack>
            </Collapse>
            <SettingForm lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel
                  control={
                    <Switch
                      checked={permission.enabled(GroupPermission.advance_delete)}
                      onChange={onAllowAdvanceDeleteChange}
                    />
                  }
                  label={t("group.advanceDelete")}
                />
                <NoMarginHelperText>{t("group.advanceDeleteDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("group.allowedNodes")} lgWidth={5} pro>
              <FormControl fullWidth>
                <MultipleNodeSelectionInput />
                <NoMarginHelperText>{t("group.allowedNodesDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm lgWidth={5}>
              <FormControl fullWidth onClick={onProClick}>
                <FormControlLabel
                  control={<Switch checked={false} />}
                  label={
                    <Box sx={{ display: "flex", alignItems: "center" }}>
                      {t("group.allowSelectNode")}
                      <ProChip size="small" label="Pro" />
                    </Box>
                  }
                />
                <NoMarginHelperText>{t("group.allowSelectNodeDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
          </>
        )}

        <SettingForm title={t("group.maxWalkedFiles")} lgWidth={5}>
          <FormControl fullWidth>
            <DenseFilledTextField
              required
              slotProps={{
                htmlInput: {
                  type: "number",
                  min: 1,
                },
              }}
              value={values.settings?.max_walked_files ?? 0}
              onChange={onMaxWalkedFilesChange}
            />
            <NoMarginHelperText>{t("group.maxWalkedFilesDes")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>
        {values?.id != AnonymousGroupID && (
          <SettingForm title={t("group.trashBinDuration")} lgWidth={5}>
            <FormControl fullWidth>
              <DenseFilledTextField
                required
                slotProps={{
                  htmlInput: {
                    type: "number",
                    min: 1,
                  },
                }}
                value={values.settings?.trash_retention ?? 0}
                onChange={onTrashBinDurationChange}
              />
              <NoMarginHelperText>{t("group.trashBinDurationDes")}</NoMarginHelperText>
            </FormControl>
          </SettingForm>
        )}
      </SettingSectionContent>
    </SettingSection>
  );
};

export default FileManagementSection;
