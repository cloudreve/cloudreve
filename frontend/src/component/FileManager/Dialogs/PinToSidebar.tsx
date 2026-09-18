import { useTranslation } from "react-i18next";
import { DialogContent } from "@mui/material";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { ChangeEvent, useCallback, useEffect, useState } from "react";
import DraggableDialog from "../../Dialogs/DraggableDialog.tsx";
import { closePinFileDialog } from "../../../redux/globalStateSlice.ts";
import { pinToSidebar } from "../../../redux/thunks/settings.ts";
import { FilledTextField } from "../../Common/StyledComponents.tsx";

const PinToSidebar = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();

  const [name, setName] = useState("");
  const [loading, setLoading] = useState(false);

  const open = useAppSelector((state) => state.globalState.pinFileDialogOpen);
  const uri = useAppSelector((state) => state.globalState.pinFileUri);

  const onClose = useCallback(() => {
    if (!loading) {
      dispatch(closePinFileDialog());
    }
  }, [dispatch, loading]);

  const onAccept = useCallback(
    async (e?: React.FormEvent<HTMLFormElement>) => {
      if (e) {
        e.preventDefault();
      }

      if (!uri) {
        return;
      }

      setLoading(true);
      try {
        await dispatch(pinToSidebar(uri, name));
      } catch (e) {
      } finally {
        setLoading(false);
        dispatch(closePinFileDialog());
      }
    },
    [name, dispatch, uri, setLoading],
  );

  useEffect(() => {
    if (uri && open) {
      setName("");
    }
  }, [uri]);

  const onNameChange = useCallback(
    (e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
      setName(e.target.value);
    },
    [dispatch, setName],
  );

  return (
    <DraggableDialog
      title={t("application:fileManager.pin")}
      showActions
      loading={loading}
      showCancel
      onAccept={onAccept}
      dialogProps={{
        open: open ?? false,
        onClose: onClose,
        fullWidth: true,
        maxWidth: "xs",
      }}
    >
      <DialogContent>
        <FilledTextField
          sx={{ mt: 2 }}
          variant="filled"
          autoFocus
          helperText={t("application:fileManager.optional")}
          margin="dense"
          label={t("application:fileManager.pinAlias")}
          type="text"
          value={name}
          onChange={onNameChange}
          fullWidth
        />
      </DialogContent>
    </DraggableDialog>
  );
};
export default PinToSidebar;
