import React, { useCallback, useState } from "react";
import { DialogContent, FilledInput, InputLabel } from "@mui/material";
import FormControl from "@mui/material/FormControl";
import { useTranslation } from "react-i18next";
import SessionManager, { UserSettings } from "../../../session";
import DraggableDialog from "../../Dialogs/DraggableDialog.tsx";

export interface ConcurrentOptionDialogProps {
  open: boolean;
  onClose: () => void;
  onSave: (count: string) => void;
}

export default function ConcurrentOptionDialog({ open, onClose, onSave }: ConcurrentOptionDialogProps) {
  const { t } = useTranslation();
  const [count, setCount] = useState(SessionManager.getWithFallback(UserSettings.ConcurrentLimit));

  const onAccept = useCallback(() => {
    onSave(count);
  }, [count]);

  return (
    <DraggableDialog
      title={t("uploader.setConcurrent")}
      showActions
      showCancel
      onAccept={onAccept}
      dialogProps={{
        fullWidth: true,
        maxWidth: "xs",
        open,
        onClose,
      }}
    >
      <DialogContent>
        <FormControl variant="filled" fullWidth>
          <InputLabel htmlFor="component-helper">{t("uploader.concurrentTaskNumber")}</InputLabel>
          <FilledInput
            type={"number"}
            inputProps={{
              min: 1,
              step: 1,
              max: 20,
            }}
            value={count}
            onChange={(e) => setCount(e.target.value)}
          />
        </FormControl>
      </DialogContent>
    </DraggableDialog>
  );
}
