import { Box, DialogContent, Divider } from "@mui/material";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { closeSaveAsDialog } from "../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { saveAsDialogPromisePool } from "../../../redux/thunks/dialog.ts";
import { FilledTextField } from "../../Common/StyledComponents.tsx";
import DraggableDialog from "../../Dialogs/DraggableDialog.tsx";
import FolderPicker, { useFolderSelector } from "../FolderPicker.tsx";

const SaveAs = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();

  const [selectedFile, selectedPath] = useFolderSelector();
  const open = useAppSelector((state) => state.globalState.saveAsDialogOpen);
  const initialName = useAppSelector((state) => state.globalState.saveAsInitialName);
  const promiseId = useAppSelector((state) => state.globalState.saveAsPromiseId);
  const [name, setName] = useState("");

  useEffect(() => {
    if (open) {
      setName(initialName ?? "");
    }
  }, [open]);

  const onClose = useCallback(() => {
    dispatch(closeSaveAsDialog());
    if (promiseId) {
      saveAsDialogPromisePool[promiseId]?.reject("cancel");
    }
  }, [dispatch, promiseId]);

  const onAccept = useCallback(
    (e?: React.FormEvent<HTMLFormElement>) => {
      if (e) {
        e.preventDefault();
      }

      const dst = selectedFile && selectedFile.path ? selectedFile.path : selectedPath;
      dispatch(closeSaveAsDialog());
      if (promiseId && dst) {
        saveAsDialogPromisePool[promiseId]?.resolve({
          uri: dst,
          name: name,
        });
      }
    },
    [promiseId, selectedFile, name, selectedPath],
  );

  return (
    <DraggableDialog
      title={t("application:modals.saveAs")}
      showActions
      secondaryFullWidth
      onAccept={onAccept}
      secondaryAction={
        <Box sx={{ display: "flex", alignItems: "flex-end" }}>
          <FilledTextField
            variant="filled"
            autoFocus
            margin="dense"
            onChange={(e) => setName(e.target.value)}
            label={t("modals.fileName")}
            type="text"
            value={name}
            fullWidth
            required
          />
        </Box>
      }
      denseAction
      dialogProps={{
        open: open ?? false,
        onClose: onClose,
        fullWidth: true,
        maxWidth: "lg",
        disableRestoreFocus: true,
        PaperProps: {
          sx: {
            height: "100%",
          },
        },
      }}
    >
      <DialogContent sx={{ display: "flex" }}>
        <FolderPicker disableSharedWithMe={true} disableTrash={true} />
      </DialogContent>
      <Divider />
    </DraggableDialog>
  );
};
export default SaveAs;
