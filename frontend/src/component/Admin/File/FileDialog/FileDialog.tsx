import { Box, Button, Collapse, DialogActions, DialogContent } from "@mui/material";
import * as React from "react";
import { createContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { CSSTransition, SwitchTransition } from "react-transition-group";
import { getFileDetail, upsertFile } from "../../../../api/api.ts";
import { File, UpsertFileService } from "../../../../api/dashboard.ts";
import { useAppDispatch } from "../../../../redux/hooks.ts";
import AutoHeight from "../../../Common/AutoHeight.tsx";
import FacebookCircularProgress from "../../../Common/CircularProgress.tsx";
import DraggableDialog from "../../../Dialogs/DraggableDialog.tsx";
import FileForm from "./FileForm.tsx";

export interface FileDialogProps {
  open: boolean;
  onClose: () => void;
  fileID?: number;
  onUpdated?: (file: File) => void;
}

export interface FileDialogContextProps {
  values: File;
  setFile: (f: (p: File) => File) => void;
  formRef?: React.RefObject<HTMLFormElement>;
}

const defaultFile: File = {
  id: 0,
  name: "",
  size: 0,
  edges: {},
};

export const FileDialogContext = createContext<FileDialogContextProps>({
  values: { ...defaultFile },
  setFile: () => {},
});

const FileDialog = ({ open, onClose, fileID, onUpdated }: FileDialogProps) => {
  const dispatch = useAppDispatch();
  const { t } = useTranslation("dashboard");
  const [values, setValues] = useState<File>({
    ...defaultFile,
  });
  const [modifiedValues, setModifiedValues] = useState<File>({
    ...defaultFile,
  });
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const formRef = useRef<HTMLFormElement>(null);

  const showSaveButton = useMemo(() => {
    return JSON.stringify(modifiedValues) !== JSON.stringify(values);
  }, [modifiedValues, values]);

  useEffect(() => {
    if (!fileID || !open) {
      return;
    }
    setLoading(true);
    dispatch(getFileDetail(fileID))
      .then((res) => {
        setValues(res);
        setModifiedValues(res);
      })
      .catch(() => {
        onClose();
      })
      .finally(() => {
        setLoading(false);
      });
  }, [open]);

  const revert = () => {
    setModifiedValues(values);
  };

  const submit = () => {
    if (formRef.current) {
      if (!formRef.current.checkValidity()) {
        formRef.current.reportValidity();
        return;
      }
    }

    const args: UpsertFileService = {
      file: { ...modifiedValues },
    };

    setSubmitting(true);
    dispatch(upsertFile(args))
      .then((res) => {
        setValues(res);
        setModifiedValues(res);
        onUpdated?.(res);
      })
      .finally(() => {
        setSubmitting(false);
      });
  };

  return (
    <FileDialogContext.Provider
      value={{
        values: modifiedValues,
        setFile: setModifiedValues,
        formRef,
      }}
    >
      <DraggableDialog
        title={t("file.fileDialogTitle")}
        dialogProps={{
          fullWidth: true,
          maxWidth: "md",
          open: open,
          onClose: onClose,
        }}
      >
        <DialogContent>
          <AutoHeight>
            <SwitchTransition>
              <CSSTransition
                addEndListener={(node, done) => node.addEventListener("transitionend", done, false)}
                classNames="fade"
                key={`${loading}`}
              >
                <Box>
                  {loading && (
                    <Box
                      sx={{
                        py: 15,
                        height: "100%",
                        display: "flex",
                        justifyContent: "center",
                        alignItems: "center",
                      }}
                    >
                      <FacebookCircularProgress />
                    </Box>
                  )}
                  {!loading && <FileForm />}
                </Box>
              </CSSTransition>
            </SwitchTransition>
          </AutoHeight>
        </DialogContent>
        <Collapse in={showSaveButton}>
          <DialogActions>
            <Button disabled={submitting} onClick={revert}>
              {t("settings.revert")}
            </Button>
            <Button loading={submitting} variant="contained" onClick={submit}>
              {t("settings.save")}
            </Button>
          </DialogActions>
        </Collapse>
      </DraggableDialog>
    </FileDialogContext.Provider>
  );
};

export default FileDialog;
