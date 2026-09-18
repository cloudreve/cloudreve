import { DialogContent, Stack } from "@mui/material";
import { ChangeEvent, useCallback, useContext, useEffect, useRef, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { FileType } from "../../../api/explorer.ts";
import { closeRenameFileModal, setRenameFileModalError } from "../../../redux/fileManagerSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { renameDialogPromisePool } from "../../../redux/thunks/dialog.ts";
import { validateFileName } from "../../../redux/thunks/file.ts";
import DraggableDialog, { StyledDialogContentText } from "../../Dialogs/DraggableDialog.tsx";

import { FilledTextField } from "../../Common/StyledComponents.tsx";
import { FmIndexContext } from "../FmIndexContext.tsx";

const Rename = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();

  const [name, setName] = useState("");
  const formRef = useRef<HTMLFormElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const fmIndex = useContext(FmIndexContext);

  const open = useAppSelector((state) => state.fileManager[0].renameFileModalOpen);
  const targets = useAppSelector((state) => state.fileManager[0].renameFileModalSelected);
  const promiseId = useAppSelector((state) => state.fileManager[0].renameFileModalPromiseId);
  const loading = useAppSelector((state) => state.fileManager[0].renameFileModalLoading);
  const error = useAppSelector((state) => state.fileManager[0].renameFileModalError);

  const onClose = useCallback(() => {
    dispatch(
      closeRenameFileModal({
        index: 0,
        value: undefined,
      }),
    );
    if (promiseId) {
      renameDialogPromisePool[promiseId]?.reject("cancel");
    }
  }, [dispatch, targets, promiseId]);

  const onAccept = useCallback(
    (e?: React.FormEvent<HTMLFormElement>) => {
      if (e) {
        e.preventDefault();
      }
      if (promiseId) {
        dispatch(validateFileName(0, renameDialogPromisePool[promiseId]?.resolve, name));
      }
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
    if (targets && open) {
      setName(targets.name);
    }
  }, [targets, open]);

  useEffect(() => {
    if (targets && open && inputRef.current) {
      const lastDot = targets.type == FileType.folder ? 0 : targets.name.lastIndexOf(".");
      inputRef.current.setSelectionRange(0, lastDot > 0 ? lastDot : targets.name.length);
    }
  }, [inputRef.current, open]);

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
      title={t("application:fileManager.rename")}
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
          <StyledDialogContentText>
            <Trans
              i18nKey="modals.renameDescription"
              ns={"application"}
              values={{
                name: targets?.name,
              }}
              components={[<strong key={0} />]}
            />
          </StyledDialogContentText>
          <form ref={formRef} onSubmit={onAccept}>
            <FilledTextField
              inputRef={inputRef}
              variant="filled"
              autoFocus
              error={!!error}
              helperText={error}
              margin="dense"
              label={t("application:modals.newName")}
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
export default Rename;
