import { DialogContent, FormControl, InputLabel, MenuItem, Select, Stack, useMediaQuery, useTheme } from "@mui/material";
import { useSnackbar } from "notistack";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { sendCreateRemoteDownload } from "../../../api/api.ts";
import { defaultPath } from "../../../hooks/useNavigation.tsx";
import { closeRemoteDownloadDialog } from "../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { getFileLinkedUri } from "../../../util";
import CrUri, { Filesystem } from "../../../util/uri.ts";
import { FileDisplayForm } from "../../Common/Form/FileDisplayForm.tsx";
import { OutlineIconTextField } from "../../Common/Form/OutlineIconTextField.tsx";
import { PathSelectorForm } from "../../Common/Form/PathSelectorForm.tsx";
import TargetNodeSelect from "../../Common/Form/TargetNodeSelect.tsx";
import { ViewTaskAction } from "../../Common/Snackbar/snackbar.tsx";
import DraggableDialog from "../../Dialogs/DraggableDialog.tsx";
import Edit from "../../Icons/Edit.tsx";
import Link from "../../Icons/Link.tsx";
import LockClosedKey from "../../Icons/LockClosedKey.tsx";
import PersonOutlined from "../../Icons/PersonOutlined.tsx";
import { FileManagerIndex } from "../FileManager.tsx";

const providerNames: Record<string, string> = {
  aria2: "Aria2",
  qbittorrent: "qBittorrent",
};

const providerDisplayName = (p: string) => providerNames[p] ?? p;

const CreateRemoteDownload = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const { enqueueSnackbar } = useSnackbar();

  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const [loading, setLoading] = useState(false);
  const [path, setPath] = useState("");
  const [url, setUrl] = useState("");
  const [fileName, setFileName] = useState("");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [headers, setHeaders] = useState("");
  const [provider, setProvider] = useState("");
  const [targetNode, setTargetNode] = useState("");

  const providers = useAppSelector((state) => state.siteConfig.explorer?.config?.remote_download_providers);
  const open = useAppSelector((state) => state.globalState.remoteDownloadDialogOpen);
  const target = useAppSelector((state) => state.globalState.remoteDownloadDialogFile);
  const current = useAppSelector((state) => state.fileManager[FileManagerIndex.main].pure_path);

  useEffect(() => {
    if (open) {
      const initialPath = new CrUri(current ?? defaultPath);
      const fs = initialPath.fs();
      setPath(fs == Filesystem.shared_with_me || fs == Filesystem.trash ? defaultPath : initialPath.toString());
      setUrl("");
      setFileName("");
      setUsername("");
      setPassword("");
      setHeaders("");
      setProvider("");
      setTargetNode("");
    }
  }, [open]);

  const onClose = useCallback(() => {
    dispatch(closeRemoteDownloadDialog());
  }, [dispatch]);

  const onAccept = useCallback(() => {
    if (!target && !url) {
      return;
    }

    setLoading(true);
    dispatch(
      sendCreateRemoteDownload({
        src_file: target ? getFileLinkedUri(target) : undefined,
        dst: path,
        src: url ? url.split("\n") : undefined,
        file_name: fileName || undefined,
        username: username || undefined,
        password: password || undefined,
        headers: headers ? headers.split("\n").filter((h) => h.trim()) : undefined,
        provider: provider || undefined,
        target_node: targetNode || undefined,
      }),
    )
      .then(() => {
        dispatch(closeRemoteDownloadDialog());
        enqueueSnackbar({
          message: t("modals.taskCreated"),
          variant: "success",
          action: ViewTaskAction("/downloads"),
        });
      })
      .finally(() => {
        setLoading(false);
      });
  }, [target, url, path, fileName, username, password, headers, provider, targetNode]);

  return (
    <DraggableDialog
      title={t("application:modals.newRemoteDownloadTitle")}
      showActions
      loading={loading}
      disabled={!target && !url}
      showCancel
      onAccept={onAccept}
      dialogProps={{
        open: open ?? false,
        onClose: onClose,
        fullWidth: true,
        maxWidth: "sm",
        disableRestoreFocus: true,
      }}
    >
      <DialogContent sx={{ pt: 1 }}>
        <Stack spacing={3}>
          <Stack spacing={3} direction={isMobile ? "column" : "row"}>
            {target && <FileDisplayForm file={target} label={t("modals.remoteDownloadURL")} />}
            {!target && (
              <OutlineIconTextField
                icon={<Link />}
                variant="outlined"
                value={url}
                multiline
                onChange={(e) => setUrl(e.target.value)}
                placeholder={t("modals.remoteDownloadURLDescription")}
                label={t("application:modals.remoteDownloadURL")}
                fullWidth
              />
            )}
          </Stack>
          <Stack spacing={3} direction={isMobile ? "column" : "row"}>
            <PathSelectorForm
              onChange={setPath}
              path={path}
              variant={"downloadTo"}
              label={t("modals.remoteDownloadDst")}
            />
          </Stack>
          <Stack spacing={3} direction={isMobile ? "column" : "row"}>
            <OutlineIconTextField
              icon={<Edit />}
              variant="outlined"
              value={fileName}
              onChange={(e) => setFileName(e.target.value)}
              label={t("application:modals.remoteDownloadFileName")}
              placeholder={t("modals.remoteDownloadFileNameDescription")}
              fullWidth
            />
          </Stack>
          <TargetNodeSelect value={targetNode} onChange={setTargetNode} />
          {providers && providers.length > 1 && (
            <Stack spacing={3} direction={isMobile ? "column" : "row"}>
              <FormControl variant="outlined" fullWidth>
                <InputLabel>{t("application:modals.remoteDownloadProvider")}</InputLabel>
                <Select
                  variant="outlined"
                  label={t("application:modals.remoteDownloadProvider")}
                  value={provider}
                  onChange={(e) => setProvider(e.target.value as string)}
                >
                  <MenuItem value="">
                    <em>{t("application:modals.remoteDownloadProviderAuto")}</em>
                  </MenuItem>
                  {providers.map((p) => (
                    <MenuItem key={p} value={p}>
                      {providerDisplayName(p)}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Stack>
          )}
          {!target && (
            <Stack spacing={3} direction={isMobile ? "column" : "row"}>
              <OutlineIconTextField
                icon={<PersonOutlined />}
                variant="outlined"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                label={t("application:modals.remoteDownloadHttpUser")}
                fullWidth
              />
              <OutlineIconTextField
                icon={<LockClosedKey />}
                variant="outlined"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                label={t("application:modals.remoteDownloadHttpPassword")}
                fullWidth
              />
            </Stack>
          )}
          {!target && (
            <Stack spacing={3} direction={isMobile ? "column" : "row"}>
              <OutlineIconTextField
                icon={<Link />}
                variant="outlined"
                value={headers}
                multiline
                minRows={2}
                onChange={(e) => setHeaders(e.target.value)}
                placeholder={t("modals.remoteDownloadHeadersDescription")}
                label={t("application:modals.remoteDownloadHeaders")}
                fullWidth
              />
            </Stack>
          )}
        </Stack>
      </DialogContent>
    </DraggableDialog>
  );
};
export default CreateRemoteDownload;
