import { DialogContent, Stack, useMediaQuery, useTheme } from "@mui/material";
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
import { ViewTaskAction } from "../../Common/Snackbar/snackbar.tsx";
import DraggableDialog from "../../Dialogs/DraggableDialog.tsx";
import Link from "../../Icons/Link.tsx";
import { FileManagerIndex } from "../FileManager.tsx";

const CreateRemoteDownload = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const { enqueueSnackbar } = useSnackbar();

  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const [loading, setLoading] = useState(false);
  const [path, setPath] = useState("");
  const [url, setUrl] = useState("");

  const open = useAppSelector((state) => state.globalState.remoteDownloadDialogOpen);
  const target = useAppSelector((state) => state.globalState.remoteDownloadDialogFile);
  const current = useAppSelector((state) => state.fileManager[FileManagerIndex.main].pure_path);

  useEffect(() => {
    if (open) {
      const initialPath = new CrUri(current ?? defaultPath);
      const fs = initialPath.fs();
      setPath(fs == Filesystem.shared_with_me || fs == Filesystem.trash ? defaultPath : initialPath.toString());
      setUrl("");
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
  }, [target, url, path]);

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
        </Stack>
      </DialogContent>
    </DraggableDialog>
  );
};
export default CreateRemoteDownload;
