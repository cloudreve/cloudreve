import { useTranslation } from "react-i18next";
import { DialogContent, Stack } from "@mui/material";
import { useAppDispatch, useAppSelector } from "../../redux/hooks.ts";
import { useCallback } from "react";
import DraggableDialog, { StyledDialogContentText } from "./DraggableDialog.tsx";
import { generalDialogPromisePool } from "../../redux/thunks/dialog.ts";
import { closeConfirmDialog } from "../../redux/globalStateSlice.ts";

const Confirmation = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();

  const open = useAppSelector((state) => state.globalState.confirmDialogOpen);
  const message = useAppSelector((state) => state.globalState.confirmDialogMessage);
  const promiseId = useAppSelector((state) => state.globalState.confirmPromiseId);

  const onClose = useCallback(() => {
    dispatch(closeConfirmDialog());
    if (promiseId) {
      generalDialogPromisePool[promiseId]?.reject("cancel");
    }
  }, [dispatch, promiseId]);

  const onAccept = useCallback(() => {
    dispatch(closeConfirmDialog());
    if (promiseId) {
      generalDialogPromisePool[promiseId]?.resolve();
    }
  }, [promiseId]);

  return (
    <DraggableDialog
      title={t("common:areYouSure")}
      showActions
      showCancel
      onAccept={onAccept}
      dialogProps={{
        open: open ?? false,
        onClose: onClose,
      }}
    >
      <DialogContent>
        <Stack spacing={2}>
          <StyledDialogContentText>{message}</StyledDialogContentText>
        </Stack>
      </DialogContent>
    </DraggableDialog>
  );
};
export default Confirmation;
