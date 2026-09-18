import { useTranslation } from "react-i18next";
import { Button, DialogContent } from "@mui/material";
import { useAppDispatch, useAppSelector } from "../../redux/hooks.ts";
import React, { useCallback, useRef } from "react";
import DraggableDialog from "./DraggableDialog.tsx";
import { closeBatchDownloadLogDialog } from "../../redux/globalStateSlice.ts";
import { FilledTextField } from "../Common/StyledComponents.tsx";
import { cancelSignals } from "../../redux/thunks/download.ts";

const BatchDownloadLog = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const logRef = useRef<HTMLInputElement>(null);

  const open = useAppSelector((state) => state.globalState.batchDownloadLogDialogOpen);
  const downloadId = useAppSelector((state) => state.globalState.batchDownloadLogDialogId);
  const logs = useAppSelector((state) => state.globalState.batchDownloadLogDialogLogs?.[downloadId ?? ""]);

  const onClose = useCallback(() => {
    dispatch(closeBatchDownloadLogDialog());
  }, [dispatch]);

  const onCancel = useCallback(() => {
    cancelSignals[downloadId ?? ""]?.abort();
    dispatch(closeBatchDownloadLogDialog());
  }, [downloadId, dispatch]);

  return (
    <DraggableDialog
      title={t("modals.directoryDownloadTitle")}
      secondaryAction={
        <Button onClick={onCancel} color={"error"}>
          {t("modals.cancelDownload")}
        </Button>
      }
      showActions
      hideOk
      dialogProps={{
        open: open ?? false,
        onClose: onClose,
        fullWidth: true,
      }}
    >
      <DialogContent sx={{ pb: 0 }}>
        <FilledTextField
          inputProps={{
            ref: logRef,
            sx: {
              // @ts-ignore
              fontSize: (theme) => theme.typography.body2.fontSize,
            },
          }}
          sx={{ pt: 0.5 }}
          minRows={10}
          maxRows={10}
          variant="outlined"
          value={logs}
          multiline
          fullWidth
          id="standard-basic"
        />
      </DialogContent>
    </DraggableDialog>
  );
};
export default BatchDownloadLog;
