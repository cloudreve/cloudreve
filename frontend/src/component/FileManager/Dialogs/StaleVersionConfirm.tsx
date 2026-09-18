import { Trans, useTranslation } from "react-i18next";
import { Button, DialogContent, Stack } from "@mui/material";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { useCallback } from "react";
import DraggableDialog, { StyledDialogContentText } from "../../Dialogs/DraggableDialog.tsx";
import { askSaveAs, staleVersionDialogPromisePool } from "../../../redux/thunks/dialog.ts";
import { closeStaleVersionDialog } from "../../../redux/globalStateSlice.ts";
import CrUri from "../../../util/uri.ts";

const StaleVersionConfirm = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();

  const open = useAppSelector((state) => state.globalState.staleVersionDialogOpen);
  const uri = useAppSelector((state) => state.globalState.staleVersionUri);
  const promiseId = useAppSelector((state) => state.globalState.staleVersionPromiseId);

  const onClose = useCallback(() => {
    dispatch(closeStaleVersionDialog());
    if (promiseId) {
      staleVersionDialogPromisePool[promiseId]?.reject("cancel");
    }
  }, [dispatch, promiseId]);

  const onAccept = useCallback(
    (e?: React.FormEvent<HTMLFormElement>) => {
      if (e) {
        e.preventDefault();
      }
      if (promiseId) {
        staleVersionDialogPromisePool[promiseId]?.resolve({ overwrite: true });
        dispatch(closeStaleVersionDialog());
      }
    },
    [promiseId, name],
  );

  const onSaveAs = useCallback(async () => {
    if (!uri) {
      return;
    }
    try {
      const fileName = new CrUri(uri).elements().pop();
      if (fileName && promiseId) {
        const saveAsDst = await dispatch(askSaveAs(fileName));
        const dst = new CrUri(saveAsDst.uri).join(saveAsDst.name);
        staleVersionDialogPromisePool[promiseId]?.resolve({
          overwrite: false,
          saveAs: dst.toString(),
        });
        dispatch(closeStaleVersionDialog());
      }
    } catch (e) {
      return;
    }
  }, [dispatch, promiseId, uri]);

  return (
    <DraggableDialog
      title={t("application:modals.versionConflict")}
      showActions
      okText={t("application:modals.overwrite")}
      showCancel
      onAccept={onAccept}
      secondaryAction={
        <Button variant={"contained"} onClick={onSaveAs} color="primary">
          {t("modals.saveAs")}
        </Button>
      }
      dialogProps={{
        open: open ?? false,
        onClose: onClose,
        fullWidth: true,
        maxWidth: "sm",
      }}
    >
      <DialogContent sx={{ pb: 0 }}>
        <Stack spacing={2}>
          <StyledDialogContentText>
            {t("modals.conflictDes1")}
            <ul>
              <Trans i18nKey="modals.conflictDes2" ns={"application"} components={[<li key={0} />, <li key={1} />]} />
            </ul>
          </StyledDialogContentText>
        </Stack>
      </DialogContent>
    </DraggableDialog>
  );
};
export default StaleVersionConfirm;
