import { useTranslation } from "react-i18next";
import { DialogContent, List, ListItemButton, ListItemText } from "@mui/material";
import { useAppDispatch, useAppSelector } from "../../redux/hooks.ts";
import React, { useCallback } from "react";
import DraggableDialog from "./DraggableDialog.tsx";
import { selectOptionDialogPromisePool } from "../../redux/thunks/dialog.ts";
import { closeSelectOptionDialog } from "../../redux/globalStateSlice.ts";

const SelectOption = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();

  const open = useAppSelector((state) => state.globalState.selectOptionDialogOpen);
  const title = useAppSelector((state) => state.globalState.selectOptionTitle);
  const promiseId = useAppSelector((state) => state.globalState.selectOptionPromiseId);
  const options = useAppSelector((state) => state.globalState.selectOptionDialogOptions);

  const onClose = useCallback(() => {
    dispatch(closeSelectOptionDialog());
    if (promiseId) {
      selectOptionDialogPromisePool[promiseId]?.reject("cancel");
    }
  }, [dispatch, promiseId]);

  const onAccept = useCallback(
    (v: any) => {
      dispatch(closeSelectOptionDialog());
      if (promiseId) {
        selectOptionDialogPromisePool[promiseId]?.resolve(v);
      }
    },
    [promiseId],
  );

  return (
    <DraggableDialog
      title={t(title ?? "")}
      dialogProps={{
        open: open ?? false,
        onClose: onClose,
        maxWidth: "sm",
      }}
    >
      <DialogContent>
        <List component="nav">
          {options?.map((o) => (
            <ListItemButton key={o.value} onClick={() => onAccept(o.value)}>
              <ListItemText
                primary={o.name}
                secondary={o.description}
                slotProps={{
                  primary: {
                    variant: "body2",
                    fontWeight: "bold",
                  },

                  secondary: { variant: "body2" },
                }}
              />
            </ListItemButton>
          ))}
        </List>
      </DialogContent>
    </DraggableDialog>
  );
};
export default SelectOption;
