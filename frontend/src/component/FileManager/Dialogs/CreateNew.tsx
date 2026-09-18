import { useTranslation } from "react-i18next";
import { DialogContent, Stack } from "@mui/material";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { ChangeEvent, useCallback, useEffect, useRef, useState } from "react";
import { setRenameFileModalError } from "../../../redux/fileManagerSlice.ts";
import DraggableDialog from "../../Dialogs/DraggableDialog.tsx";
import { createNewDialogPromisePool } from "../../../redux/thunks/dialog.ts";
import { FilledTextField } from "../../Common/StyledComponents.tsx";
import { closeCreateNewDialog, CreateNewDialogType } from "../../../redux/globalStateSlice.ts";
import { submitCreateNew } from "../../../redux/thunks/file.ts";
import { FileType } from "../../../api/explorer.ts";

const CreateNew = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();

  const [name, setName] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const formRef = useRef<HTMLFormElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const open = useAppSelector((state) => state.globalState.createNewDialogOpen);
  const promiseId = useAppSelector((state) => state.globalState.createNewPromiseId);
  const type = useAppSelector((state) => state.globalState.createNewDialogType);
  const defaultName = useAppSelector((state) => state.globalState.createNewDialogDefault);
  const fmIndex = useAppSelector((state) => state.globalState.createNewDialogFmIndex) ?? 0;

  useEffect(() => {
    if (open) {
      setName(defaultName ?? "");
    }
  }, [open]);

  const onClose = useCallback(() => {
    dispatch(closeCreateNewDialog());
    if (promiseId) {
      createNewDialogPromisePool[promiseId]?.reject("cancel");
    }
  }, [dispatch, promiseId]);

  const onAccept = useCallback(
    (e?: React.FormEvent<HTMLFormElement>) => {
      if (e) {
        e.preventDefault();
      }

      setLoading(true);
      dispatch(submitCreateNew(fmIndex, name, type == CreateNewDialogType.folder ? FileType.folder : FileType.file))
        .then((f) => {
          if (promiseId) {
            createNewDialogPromisePool[promiseId]?.resolve(f);
          }
          dispatch(closeCreateNewDialog());
        })
        .finally(() => {
          setLoading(false);
        });
    },
    [promiseId, name],
  );

  const onOkClicked = useCallback(() => {
    if (formRef.current) {
      if (formRef.current.reportValidity()) {
        onAccept();
      }
    }
  }, [formRef, onAccept]);

  useEffect(() => {
    if (open) {
      const lastDot = name.lastIndexOf(".");
      setTimeout(
        () => inputRef.current && inputRef.current.setSelectionRange(0, lastDot > 0 ? lastDot : name.length),
        200,
      );
    }
  }, [open, inputRef.current]);

  const onNameChange = useCallback(
    (e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
      setName(e.target.value);
      if (error) {
        dispatch(setRenameFileModalError({ index: 0, value: undefined }));
      }
    },
    [dispatch, setName, error],
  );

  return (
    <DraggableDialog
      title={t(
        type == CreateNewDialogType.folder ? "application:fileManager.newFolder" : "application:fileManager.newFile",
      )}
      showActions
      loading={loading}
      showCancel
      onAccept={onOkClicked}
      dialogProps={{
        open: open ?? false,
        onClose: onClose,
        fullWidth: true,
        maxWidth: "sm",
        disableRestoreFocus: true,
      }}
    >
      <DialogContent>
        <Stack spacing={2}>
          <form ref={formRef} onSubmit={onAccept}>
            <FilledTextField
              inputRef={inputRef}
              variant="filled"
              autoFocus
              error={!!error}
              helperText={error}
              margin="dense"
              label={t(
                type == CreateNewDialogType.folder ? "application:modals.folderName" : "application:modals.fileName",
              )}
              type="text"
              value={name}
              onChange={onNameChange}
              fullWidth
              required
            />
          </form>
        </Stack>
      </DialogContent>
    </DraggableDialog>
  );
};
export default CreateNew;
