import React, { useCallback, useEffect } from "react";
import { Box, DialogContent, styled, Typography } from "@mui/material";
import { useTranslation } from "react-i18next";
import { useAppDispatch, useAppSelector } from "../../redux/hooks.ts";
import DraggableDialog from "../Dialogs/DraggableDialog.tsx";
import { setUploadFromClipboardDialog } from "../../redux/globalStateSlice.ts";
import Clipboard from "../Icons/Clipboard.tsx";

export interface PasteUploadDialogProps {
  onFilePasted: (files: File[]) => void;
}

export interface PasteTargetProps extends PasteUploadDialogProps {
  onClose: () => void;
}

const PasteTargetContainer = styled(Box)(({ theme }) => ({
  borderRadius: theme.shape.borderRadius,
  backgroundColor: theme.palette.mode == "light" ? "rgba(0, 0, 0, 0.06)" : "rgba(255, 255, 255, 0.09)",
  display: "flex",
  justifyContent: "center",
  alignItems: "center",
  flexDirection: "column",
  padding: theme.spacing(2),
  color: theme.palette.text.secondary,
}));

const PasteTarget = ({ onFilePasted, onClose }: PasteTargetProps) => {
  const { t } = useTranslation();
  const pasteHandler = (e: ClipboardEvent) => {
    e.preventDefault();
    if (!e.clipboardData?.files.length) {
      return;
    }

    onFilePasted(Array.from(e.clipboardData.files));
    onClose();
  };

  useEffect(() => {
    // listen to onpaste event for window
    window.addEventListener("paste", pasteHandler);
    return () => {
      window.removeEventListener("paste", pasteHandler);
    };
  }, []);

  const disableDefault = useCallback((e: React.MouseEvent | React.KeyboardEvent | React.ClipboardEvent) => {
    e.preventDefault();
    return false;
  }, []);
  return (
    <PasteTargetContainer sx={{ py: 5, mt: 0.5 }}>
      <Clipboard sx={{ fontSize: 60 }} />
      <Typography sx={{ pt: 1 }}>{t("uploader.pasteFilesHere")}</Typography>
    </PasteTargetContainer>
  );
};

export default function PasteUploadDialog({ onFilePasted }: PasteUploadDialogProps) {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const open = useAppSelector((state) => state.globalState.uploadFromClipboardDialogOpen);

  const onClose = useCallback(() => {
    dispatch(setUploadFromClipboardDialog(false));
  }, []);

  return (
    <DraggableDialog
      title={t("uploader.uploadFromClipboard")}
      dialogProps={{
        fullWidth: true,
        maxWidth: "xs",
        open: !!open,
        onClose,
      }}
    >
      <DialogContent>
        <PasteTarget onClose={onClose} onFilePasted={onFilePasted} />
      </DialogContent>
    </DraggableDialog>
  );
}
