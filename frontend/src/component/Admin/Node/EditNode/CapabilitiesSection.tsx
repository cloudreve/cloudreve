import {
  Alert,
  CircularProgress,
  Collapse,
  FormControl,
  FormControlLabel,
  IconButton,
  Link,
  ListItemText,
  SelectChangeEvent,
  Switch,
  Typography,
  useTheme,
} from "@mui/material";
import { useSnackbar } from "notistack";
import { lazy, Suspense, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { testNodeDownloader } from "../../../../api/api";
import { DownloaderProvider, Node, NodeType } from "../../../../api/dashboard";
import { NodeCapability } from "../../../../api/workflow";
import { useAppDispatch } from "../../../../redux/hooks";
import Boolset from "../../../../util/boolset";
import { DefaultCloseAction } from "../../../Common/Snackbar/snackbar";
import { DenseFilledTextField, DenseSelect, SecondaryButton } from "../../../Common/StyledComponents";
import { SquareMenuItem } from "../../../FileManager/ContextMenu/ContextMenu";
import QuestionCircle from "../../../Icons/QuestionCircle";
import SettingForm from "../../../Pages/Setting/SettingForm";
import { Code } from "../../../Common/Code.tsx";
import { EndpointInput } from "../../Common/EndpointInput";
import { NoMarginHelperText, SettingSection, SettingSectionContent } from "../../Settings/Settings";
import { NodeSettingContext } from "./NodeSettingWrapper";
import StoreFilesHintDialog from "./StoreFilesHintDialog";
const MonacoEditor = lazy(() => import("../../../Viewers/CodeViewer/MonacoEditor"));

const CapabilitiesSection = () => {
  const { t } = useTranslation("dashboard");
  const { values, setNode } = useContext(NodeSettingContext);
  const theme = useTheme();
  const dispatch = useAppDispatch();
  const { enqueueSnackbar } = useSnackbar();

  const [editedConfigAria2, setEditedConfigAria2] = useState("");
  const [editedConfigQbittorrent, setEditedConfigQbittorrent] = useState("");
  const [testDownloaderLoading, setTestDownloaderLoading] = useState(false);
  const [storeFilesHintDialogOpen, setStoreFilesHintDialogOpen] = useState(false);

  const capabilities = useMemo(() => {
    return new Boolset(values.capabilities ?? "");
  }, [values.capabilities]);

  const hasRemoteDownload = useMemo(() => {
    return capabilities.enabled(NodeCapability.remote_download);
  }, [capabilities]);

  useEffect(() => {
    setEditedConfigAria2(
      values.settings?.aria2?.options ? JSON.stringify(values.settings?.aria2?.options, null, 2) : "",
    );
  }, [values.settings?.aria2?.options]);

  useEffect(() => {
    setEditedConfigQbittorrent(
      values.settings?.qbittorrent?.options ? JSON.stringify(values.settings?.qbittorrent?.options, null, 2) : "",
    );
  }, [values.settings?.qbittorrent?.options]);

  const onCapabilityChange = useCallback(
    (capability: number) => (e: React.ChangeEvent<HTMLInputElement>) => {
      setNode((p: Node) => ({
        ...p,
        capabilities: new Boolset(p.capabilities).set(capability, e.target.checked).toString(),
      }));
    },
    [setNode],
  );

  const onProviderChange = useCallback(
    (e: SelectChangeEvent<unknown>) => {
      const provider = e.target.value as DownloaderProvider;
      setNode((p: Node) => ({
        ...p,
        settings: {
          ...p.settings,
          provider,
        },
      }));
    },
    [setNode],
  );

  const onAria2ServerChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setNode((p: Node) => ({
        ...p,
        settings: {
          ...p.settings,
          aria2: {
            ...p.settings?.aria2,
            server: e.target.value,
          },
        },
      }));
    },
    [setNode],
  );

  const onAria2TokenChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setNode((p: Node) => ({
        ...p,
        settings: {
          ...p.settings,
          aria2: {
            ...p.settings?.aria2,
            token: e.target.value,
          },
        },
      }));
    },
    [setNode],
  );

  const onAria2TempPathChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setNode((p: Node) => ({
        ...p,
        settings: {
          ...p.settings,
          aria2: {
            ...p.settings?.aria2,
            temp_path: e.target.value ? e.target.value : undefined,
          },
        },
      }));
    },
    [setNode],
  );

  const onQBittorrentServerChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setNode((p: Node) => ({
        ...p,
        settings: {
          ...p.settings,
          qbittorrent: {
            ...p.settings?.qbittorrent,
            server: e.target.value,
          },
        },
      }));
    },
    [setNode],
  );

  const onQBittorrentUserChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setNode((p: Node) => ({
        ...p,
        settings: {
          ...p.settings,
          qbittorrent: {
            ...p.settings?.qbittorrent,
            user: e.target.value ? e.target.value : undefined,
          },
        },
      }));
    },
    [setNode],
  );

  const onQBittorrentPasswordChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setNode((p: Node) => ({
        ...p,
        settings: {
          ...p.settings,
          qbittorrent: {
            ...p.settings?.qbittorrent,
            password: e.target.value ? e.target.value : undefined,
          },
        },
      }));
    },
    [setNode],
  );

  const onQBittorrentTempPathChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setNode((p: Node) => ({
        ...p,
        settings: {
          ...p.settings,
          qbittorrent: {
            ...p.settings?.qbittorrent,
            temp_path: e.target.value ? e.target.value : undefined,
          },
        },
      }));
    },
    [setNode],
  );

  const onIntervalChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const interval = parseInt(e.target.value);
      setNode((p: Node) => ({
        ...p,
        settings: {
          ...p.settings,
          interval: isNaN(interval) ? undefined : interval,
        },
      }));
    },
    [setNode],
  );

  const onWaitForSeedingChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setNode((p: Node) => ({
        ...p,
        settings: {
          ...p.settings,
          wait_for_seeding: e.target.checked ? true : undefined,
        },
      }));
    },
    [setNode],
  );

  const onURLValidationDisabledChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setNode((p: Node) => ({
        ...p,
        settings: {
          ...p.settings,
          url_validation: {
            ...p.settings?.url_validation,
            disabled: e.target.checked ? true : undefined,
          },
        },
      }));
    },
    [setNode],
  );

  const onURLValidationAllowedHostsChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const list = e.target.value
        .split("\n")
        .map((s) => s.trim())
        .filter((s) => s !== "");
      setNode((p: Node) => ({
        ...p,
        settings: {
          ...p.settings,
          url_validation: {
            ...p.settings?.url_validation,
            allowed_hosts: list.length ? list : undefined,
          },
        },
      }));
    },
    [setNode],
  );

  const onURLValidationAllowedCIDRsChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const list = e.target.value
        .split("\n")
        .map((s) => s.trim())
        .filter((s) => s !== "");
      setNode((p: Node) => ({
        ...p,
        settings: {
          ...p.settings,
          url_validation: {
            ...p.settings?.url_validation,
            allowed_cidrs: list.length ? list : undefined,
          },
        },
      }));
    },
    [setNode],
  );

  const urlValidation = values.settings?.url_validation;
  const allowedHostsValue = useMemo(
    () => (urlValidation?.allowed_hosts ?? []).join("\n"),
    [urlValidation?.allowed_hosts],
  );
  const allowedCIDRsValue = useMemo(
    () => (urlValidation?.allowed_cidrs ?? []).join("\n"),
    [urlValidation?.allowed_cidrs],
  );

  const onEditedConfigAria2Blur = useCallback(
    (value: string) => {
      var res: Record<string, any> | undefined = undefined;
      if (value) {
        try {
          res = JSON.parse(value);
        } catch (e) {
          console.error(e);
        }
      }
      setNode((p: Node) => ({ ...p, settings: { ...p.settings, aria2: { ...p.settings?.aria2, options: res } } }));
    },
    [editedConfigAria2, setNode],
  );

  const onEditedConfigQbittorrentBlur = useCallback(
    (value: string) => {
      var res: Record<string, any> | undefined = undefined;
      if (value) {
        try {
          res = JSON.parse(value);
        } catch (e) {
          console.error(e);
        }
      }
      setNode((p: Node) => ({
        ...p,
        settings: { ...p.settings, qbittorrent: { ...p.settings?.qbittorrent, options: res } },
      }));
    },
    [editedConfigQbittorrent, setNode],
  );

  const onTestDownloaderClick = useCallback(() => {
    setTestDownloaderLoading(true);
    dispatch(testNodeDownloader({ node: values }))
      .then((res) => {
        enqueueSnackbar({
          variant: "success",
          message: t("node.downloaderTestPass", { version: res }),
          action: DefaultCloseAction,
        });
      })
      .finally(() => {
        setTestDownloaderLoading(false);
      });
  }, [values]);

  const onStoreFilesClick = useCallback(() => {
    setStoreFilesHintDialogOpen(true);
  }, []);

  return (
    <>
      <StoreFilesHintDialog open={storeFilesHintDialogOpen} onClose={() => setStoreFilesHintDialogOpen(false)} />
      <SettingSection>
        <Typography variant="h6" gutterBottom>
          {t("node.features")}
        </Typography>
        <SettingSectionContent>
          <SettingForm lgWidth={5}>
            <FormControl fullWidth>
              <FormControlLabel
                control={
                  <Switch
                    checked={capabilities.enabled(NodeCapability.create_archive)}
                    onChange={onCapabilityChange(NodeCapability.create_archive)}
                  />
                }
                label={t("application:fileManager.createArchive")}
              />
              <NoMarginHelperText>{t("node.createArchiveDes")}</NoMarginHelperText>
            </FormControl>
          </SettingForm>
          <SettingForm lgWidth={5}>
            <FormControl fullWidth>
              <FormControlLabel
                control={
                  <Switch
                    checked={capabilities.enabled(NodeCapability.extract_archive)}
                    onChange={onCapabilityChange(NodeCapability.extract_archive)}
                  />
                }
                label={t("application:fileManager.extractArchive")}
              />
              <NoMarginHelperText>{t("node.extractArchiveDes")}</NoMarginHelperText>
            </FormControl>
          </SettingForm>
          <SettingForm lgWidth={5}>
            <FormControl fullWidth>
              <FormControlLabel
                control={
                  <Switch
                    checked={capabilities.enabled(NodeCapability.remote_download)}
                    onChange={onCapabilityChange(NodeCapability.remote_download)}
                  />
                }
                label={t("application:navbar.remoteDownload")}
              />
              <NoMarginHelperText>{t("node.remoteDownloadDes")}</NoMarginHelperText>
            </FormControl>
          </SettingForm>
          {values.type === NodeType.slave && (
            <SettingForm lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel
                  control={
                    <Switch
                      onChange={onStoreFilesClick}
                      disabled={(values.edges?.storage_policy?.length ?? 0) > 0}
                      checked={(values.edges?.storage_policy?.length ?? 0) > 0}
                    />
                  }
                  label={t("node.storeFiles")}
                />
                <NoMarginHelperText>{t("node.storeFilesDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
          )}
        </SettingSectionContent>
      </SettingSection>

      <Collapse in={hasRemoteDownload} unmountOnExit>
        <SettingSection>
          <Typography
            variant="h6"
            gutterBottom
            sx={{
              display: "flex",
              alignItems: "center",
              gap: 0.5,
            }}
          >
            {t("node.remoteDownload")}
            <IconButton
              onClick={() => {
                window.open("https://docs.cloudreve.org/usage/remote-download", "_blank");
              }}
            >
              <QuestionCircle />
            </IconButton>
          </Typography>
          <SettingSectionContent>
            <SettingForm title={t("node.downloader")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseSelect value={values.settings?.provider || DownloaderProvider.aria2} onChange={onProviderChange}>
                  <SquareMenuItem value={DownloaderProvider.aria2}>
                    <ListItemText primary="Aria2" slotProps={{ primary: { variant: "body2" } }} />
                  </SquareMenuItem>
                  <SquareMenuItem value={DownloaderProvider.qbittorrent}>
                    <ListItemText primary="qBittorrent" slotProps={{ primary: { variant: "body2" } }} />
                  </SquareMenuItem>
                </DenseSelect>
                <NoMarginHelperText>
                  {values.settings?.provider === DownloaderProvider.qbittorrent
                    ? t("node.qbittorrentDes")
                    : t("node.aria2Des")}
                </NoMarginHelperText>
              </FormControl>
            </SettingForm>

            {values.settings?.provider === DownloaderProvider.aria2 && (
              <>
                <SettingForm title={t("node.rpcServer")} lgWidth={5}>
                  <FormControl fullWidth>
                    <EndpointInput
                      fullWidth
                      required
                      value={values.settings?.aria2?.server || ""}
                      onChange={onAria2ServerChange}
                      variant={"outlined"}
                    />
                    <NoMarginHelperText>
                      <Trans i18nKey="node.rpcServerHelpDes" ns="dashboard" components={[<Code />]} />
                    </NoMarginHelperText>
                  </FormControl>
                </SettingForm>
                <SettingForm title={t("node.rpcToken")} lgWidth={5}>
                  <FormControl fullWidth>
                    <DenseFilledTextField value={values.settings?.aria2?.token || ""} onChange={onAria2TokenChange} />
                    <NoMarginHelperText>
                      <Trans i18nKey="node.rpcTokenDes" ns="dashboard" components={[<Code />]} />
                    </NoMarginHelperText>
                  </FormControl>
                </SettingForm>
                <SettingForm title={t("group.aria2Options")} lgWidth={5}>
                  <FormControl fullWidth>
                    <Suspense fallback={<CircularProgress />}>
                      <MonacoEditor
                        theme={theme.palette.mode === "dark" ? "vs-dark" : "vs"}
                        language="json"
                        value={editedConfigAria2}
                        onChange={(value) => setEditedConfigAria2(value || "")}
                        onBlur={onEditedConfigAria2Blur}
                        height="200px"
                        minHeight="200px"
                        options={{
                          wordWrap: "on",
                          minimap: { enabled: false },
                          scrollBeyondLastLine: false,
                        }}
                      />
                    </Suspense>
                    <NoMarginHelperText>
                      <Trans
                        i18nKey="node.downloaderOptionDes"
                        ns="dashboard"
                        components={[
                          <Link href="https://aria2.github.io/manual/en/html/aria2c.html#id2" target="_blank" />,
                        ]}
                      />
                    </NoMarginHelperText>
                  </FormControl>
                </SettingForm>
                <SettingForm title={t("node.tempPath")} lgWidth={5}>
                  <FormControl fullWidth>
                    <DenseFilledTextField
                      value={values.settings?.aria2?.temp_path || ""}
                      onChange={onAria2TempPathChange}
                    />
                    <NoMarginHelperText>{t("node.tempPathDes")}</NoMarginHelperText>
                  </FormControl>
                </SettingForm>
              </>
            )}

            {values.settings?.provider === DownloaderProvider.qbittorrent && (
              <>
                <SettingForm title={t("node.webUIEndpoint")} lgWidth={5}>
                  <FormControl fullWidth>
                    <EndpointInput
                      fullWidth
                      required
                      value={values.settings?.qbittorrent?.server || ""}
                      onChange={onQBittorrentServerChange}
                      variant={"outlined"}
                    />
                    <NoMarginHelperText>
                      <Trans i18nKey="node.webUIEndpointDes" ns="dashboard" components={[<Code />]} />
                    </NoMarginHelperText>
                  </FormControl>
                </SettingForm>
                <SettingForm title={t("policy.accessCredential")} lgWidth={5}>
                  <FormControl fullWidth>
                    <DenseFilledTextField
                      placeholder={t("node.webUIUsername")}
                      value={values.settings?.qbittorrent?.user || ""}
                      onChange={onQBittorrentUserChange}
                    />
                    <DenseFilledTextField
                      placeholder={t("node.webUIPassword")}
                      type="password"
                      sx={{ mt: 1 }}
                      value={values.settings?.qbittorrent?.password || ""}
                      onChange={onQBittorrentPasswordChange}
                    />
                    <NoMarginHelperText>{t("node.webUICredDes")}</NoMarginHelperText>
                  </FormControl>
                </SettingForm>
                <SettingForm title={t("group.aria2Options")} lgWidth={5}>
                  <FormControl fullWidth>
                    <Suspense fallback={<CircularProgress />}>
                      <MonacoEditor
                        theme={theme.palette.mode === "dark" ? "vs-dark" : "vs"}
                        language="json"
                        value={editedConfigQbittorrent}
                        onChange={(value) => setEditedConfigQbittorrent(value || "")}
                        onBlur={onEditedConfigQbittorrentBlur}
                        height="200px"
                        minHeight="200px"
                        options={{
                          wordWrap: "on",
                          minimap: { enabled: false },
                          scrollBeyondLastLine: false,
                        }}
                      />
                    </Suspense>
                    <NoMarginHelperText>
                      <Trans
                        i18nKey="node.downloaderOptionDes"
                        ns="dashboard"
                        components={[
                          <Link
                            href="https://github.com/qbittorrent/qBittorrent/wiki/WebUI-API-(qBittorrent-4.1)#add-new-torrent"
                            target="_blank"
                          />,
                        ]}
                      />
                    </NoMarginHelperText>
                  </FormControl>
                </SettingForm>
                <SettingForm title={t("node.tempPath")} lgWidth={5}>
                  <FormControl fullWidth>
                    <DenseFilledTextField
                      value={values.settings?.qbittorrent?.temp_path || ""}
                      onChange={onQBittorrentTempPathChange}
                    />
                    <NoMarginHelperText>{t("node.tempPathDes")}</NoMarginHelperText>
                  </FormControl>
                </SettingForm>
              </>
            )}

            <SettingForm title={t("node.refreshInterval")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  type="number"
                  slotProps={{ htmlInput: { min: 1 } }}
                  required
                  value={values.settings?.interval || ""}
                  onChange={onIntervalChange}
                />
                <NoMarginHelperText>{t("node.refreshIntervalDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>

            <SettingForm lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel
                  control={
                    <Switch checked={values.settings?.wait_for_seeding || false} onChange={onWaitForSeedingChange} />
                  }
                  label={t("node.waitForSeeding")}
                />
                <NoMarginHelperText>{t("node.waitForSeedingDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>

            <SettingForm title={t("node.urlValidation")} lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel
                  control={
                    <Switch checked={urlValidation?.disabled || false} onChange={onURLValidationDisabledChange} />
                  }
                  label={t("node.urlValidationDisable")}
                />
                <NoMarginHelperText>{t("node.urlValidationDisableDes")}</NoMarginHelperText>
                {urlValidation?.disabled && (
                  <Alert severity="warning" sx={{ mt: 1 }}>
                    {t("node.urlValidationDisabledWarning")}
                  </Alert>
                )}
              </FormControl>
            </SettingForm>
            <Collapse in={!urlValidation?.disabled} unmountOnExit>
              <SettingSectionContent>
                <SettingForm title={t("node.allowedHosts")} lgWidth={5}>
                  <FormControl fullWidth>
                    <DenseFilledTextField
                      multiline
                      rows={3}
                      value={allowedHostsValue}
                      onChange={onURLValidationAllowedHostsChange}
                      placeholder={"nas.lan\n192.168.1.50"}
                    />
                    <NoMarginHelperText>{t("node.allowedHostsDes")}</NoMarginHelperText>
                  </FormControl>
                </SettingForm>
                <SettingForm title={t("node.allowedCIDRs")} lgWidth={5}>
                  <FormControl fullWidth>
                    <DenseFilledTextField
                      multiline
                      rows={3}
                      value={allowedCIDRsValue}
                      onChange={onURLValidationAllowedCIDRsChange}
                      placeholder={"192.168.0.0/16\nfd00::/8"}
                    />
                    <NoMarginHelperText>{t("node.allowedCIDRsDes")}</NoMarginHelperText>
                  </FormControl>
                </SettingForm>
              </SettingSectionContent>
            </Collapse>

            <SettingForm lgWidth={5}>
              <SecondaryButton onClick={onTestDownloaderClick} variant="contained" loading={testDownloaderLoading}>
                {t("node.testDownloader")}
              </SecondaryButton>
            </SettingForm>
          </SettingSectionContent>
        </SettingSection>
      </Collapse>
    </>
  );
};

export default CapabilitiesSection;
