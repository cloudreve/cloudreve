import { DialogContent, Grid2, InputAdornment, TextField, useMediaQuery, useTheme } from "@mui/material";
import { useSnackbar } from "notistack";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { sendExtractArchive } from "../../../api/api.ts";
import { closeExtractArchiveDialog } from "../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { fileExtension, getFileLinkedUri } from "../../../util";
import EncodingSelector, { defaultEncodingValue } from "../../Common/Form/EncodingSelector.tsx";
import { FileDisplayForm } from "../../Common/Form/FileDisplayForm.tsx";
import { PathSelectorForm } from "../../Common/Form/PathSelectorForm.tsx";
import { ViewTaskAction } from "../../Common/Snackbar/snackbar.tsx";
import DraggableDialog from "../../Dialogs/DraggableDialog.tsx";
import Password from "../../Icons/Password.tsx";
import { FileManagerIndex } from "../FileManager.tsx";

const ExtractArchive = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const { enqueueSnackbar } = useSnackbar();

  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const [loading, setLoading] = useState(false);
  const [path, setPath] = useState("");
  const [encoding, setEncoding] = useState(defaultEncodingValue);
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);

  const open = useAppSelector((state) => state.globalState.extractArchiveDialogOpen);
  const target = useAppSelector((state) => state.globalState.extractArchiveDialogFile);
  const current = useAppSelector((state) => state.fileManager[FileManagerIndex.main].pure_path);
  const mask = useAppSelector((state) => state.globalState.extractArchiveDialogMask);
  const predefinedEncoding = useAppSelector((state) => state.globalState.extractArchiveDialogEncoding);

  useEffect(() => {
    setEncoding(predefinedEncoding ?? defaultEncodingValue);
  }, [predefinedEncoding]);

  const showEncodingOption = useMemo(() => {
    const ext = fileExtension(target?.name ?? "");
    return ext === "zip";
  }, [target?.name]);

  const showPasswordOption = useMemo(() => {
    const ext = fileExtension(target?.name ?? "");
    return ext === "zip" || ext === "7z";
  }, [target?.name]);

  useEffect(() => {
    if (open) {
      setPath(current ?? "");
    }
  }, [open]);

  const onClose = useCallback(() => {
    dispatch(closeExtractArchiveDialog());
  }, [dispatch]);

  const onAccept = useCallback(() => {
    if (!target) {
      return;
    }

    setLoading(true);
    dispatch(
      sendExtractArchive({
        src: [getFileLinkedUri(target)],
        dst: path,
        encoding: showEncodingOption && encoding != defaultEncodingValue ? encoding : undefined,
        password: showPasswordOption && password ? password : undefined,
        file_mask: mask ?? undefined,
      }),
    )
      .then(() => {
        dispatch(closeExtractArchiveDialog());
        enqueueSnackbar({
          message: t("modals.taskCreated"),
          variant: "success",
          action: ViewTaskAction(),
        });
      })
      .finally(() => {
        setLoading(false);
      });
  }, [target, encoding, path, showPasswordOption, showEncodingOption, password, mask]);

  return (
    <DraggableDialog
      title={t("application:fileManager.extractArchive")}
      showActions
      loading={loading}
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
        <Grid2 container spacing={3}>
          {target && (
            <Grid2
              size={{
                xs: 12,
                md: showEncodingOption ? 6 : 12,
              }}
            >
              <FileDisplayForm file={target} label={t("modals.archiveFile")} />
            </Grid2>
          )}
          {showEncodingOption && (
            <Grid2
              size={{
                xs: 12,
                md: 6,
              }}
            >
              <EncodingSelector
                value={encoding}
                onChange={setEncoding}
                variant="outlined"
                fullWidth
                showIcon={!isMobile}
              />
            </Grid2>
          )}
          <Grid2
            size={{
              xs: 12,
            }}
          >
            <PathSelectorForm onChange={setPath} path={path} variant={"extractTo"} label={t("modals.decompressTo")} />
          </Grid2>
          {showPasswordOption && (
            <Grid2
              size={{
                xs: 12,
              }}
            >
              <TextField
                slotProps={{
                  input: {
                    startAdornment: !isMobile && (
                      <InputAdornment position="start">
                        <Password />
                      </InputAdornment>
                    ),
                  },
                }}
                fullWidth
                placeholder={t("application:modals.passwordDescription")}
                label={t("modals.password")}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </Grid2>
          )}
        </Grid2>
      </DialogContent>
    </DraggableDialog>
  );
};
export default ExtractArchive;
