import { DialogContent, Stack, useMediaQuery, useTheme } from "@mui/material";
import { useSnackbar } from "notistack";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { sendCreateArchive } from "../../../api/api.ts";
import { closeCreateArchiveDialog } from "../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { getFileLinkedUri } from "../../../util";
import CrUri from "../../../util/uri.ts";
import { OutlineIconTextField } from "../../Common/Form/OutlineIconTextField.tsx";
import { PathSelectorForm } from "../../Common/Form/PathSelectorForm.tsx";
import { ViewTaskAction } from "../../Common/Snackbar/snackbar.tsx";
import DraggableDialog from "../../Dialogs/DraggableDialog.tsx";
import Archive from "../../Icons/Archive.tsx";
import { FileManagerIndex } from "../FileManager.tsx";

const CreateArchive = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const { enqueueSnackbar } = useSnackbar();

  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const [loading, setLoading] = useState(false);
  const [fileName, setFileName] = useState("archive.zip");
  const [path, setPath] = useState("");

  const open = useAppSelector((state) => state.globalState.createArchiveDialogOpen);
  const targets = useAppSelector((state) => state.globalState.createArchiveDialogFiles);
  const current = useAppSelector((state) => state.fileManager[FileManagerIndex.main].pure_path);

  useEffect(() => {
    if (open) {
      setPath(current ?? "");
    }
  }, [open]);

  const onClose = useCallback(() => {
    dispatch(closeCreateArchiveDialog());
  }, [dispatch]);

  const onAccept = useCallback(() => {
    if (!targets) {
      return;
    }

    setLoading(true);
    const dst = new CrUri(path);
    dispatch(
      sendCreateArchive({
        src: targets?.map((t) => getFileLinkedUri(t)),
        dst: dst.join(fileName).toString(),
      }),
    )
      .then(() => {
        dispatch(closeCreateArchiveDialog());
        enqueueSnackbar({
          message: t("modals.taskCreated"),
          variant: "success",
          action: ViewTaskAction(),
        });
      })
      .finally(() => {
        setLoading(false);
      });
  }, [targets, fileName, path]);

  return (
    <DraggableDialog
      title={t("application:fileManager.createArchive")}
      showActions
      loading={loading}
      showCancel
      disabled={!fileName}
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
          <OutlineIconTextField
            icon={<Archive />}
            variant="outlined"
            value={fileName}
            onChange={(e) => setFileName(e.target.value)}
            label={t("application:modals.zipFileName")}
            fullWidth
          />
          <Stack spacing={3} direction={isMobile ? "column" : "row"}>
            <PathSelectorForm onChange={setPath} path={path} label={t("modals.saveToTitle")} />
          </Stack>
        </Stack>
      </DialogContent>
    </DraggableDialog>
  );
};
export default CreateArchive;
