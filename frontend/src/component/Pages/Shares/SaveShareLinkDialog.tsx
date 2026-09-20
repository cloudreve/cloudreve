import { DialogContent, Stack } from "@mui/material";
import { useSnackbar } from "notistack";
import { ChangeEvent, useCallback, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { getShareInfo } from "../../../api/api.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { parseShareLink, saveShareToMyFiles } from "../../../redux/thunks/share.ts";
import { DefaultCloseAction } from "../../Common/Snackbar/snackbar.tsx";
import { FilledTextField } from "../../Common/StyledComponents.tsx";
import DraggableDialog from "../../Dialogs/DraggableDialog.tsx";

interface SaveShareLinkDialogProps {
  open: boolean;
  onClose: () => void;
}

const SaveShareLinkDialog = ({ open, onClose }: SaveShareLinkDialogProps) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const { enqueueSnackbar } = useSnackbar();

  const [link, setLink] = useState("");
  const [name, setName] = useState("");
  const [password, setPassword] = useState("");
  const [linkError, setLinkError] = useState("");
  const [loading, setLoading] = useState(false);
  const formRef = useRef<HTMLFormElement>(null);

  const reset = useCallback(() => {
    setLink("");
    setName("");
    setPassword("");
    setLinkError("");
  }, []);

  const handleClose = useCallback(() => {
    reset();
    onClose();
  }, [reset, onClose]);

  const onAccept = useCallback(
    (e?: React.FormEvent<HTMLFormElement>) => {
      e?.preventDefault();
      const parsed = parseShareLink(link);
      if (!parsed) {
        setLinkError(t("application:share.invalidShareLink"));
        return;
      }

      setLoading(true);
      const sharePassword = parsed.password || password || undefined;
      dispatch(getShareInfo(parsed.id, sharePassword))
        .then((share) => {
          if (!share.unlocked && share.password_protected && !sharePassword) {
            setLinkError(t("application:share.sharePasswordRequired"));
            return;
          }
          return dispatch(saveShareToMyFiles(share, sharePassword, name)).then(() => {
            reset();
            onClose();
          });
        })
        .catch(() => {
          enqueueSnackbar({
            message: t("application:share.shareNotExist"),
            variant: "error",
            action: DefaultCloseAction,
          });
        })
        .finally(() => {
          setLoading(false);
        });
    },
    [dispatch, link, password, name, t, enqueueSnackbar, reset, onClose],
  );

  const onOkClicked = useCallback(() => {
    if (formRef.current?.reportValidity()) {
      onAccept();
    }
  }, [onAccept]);

  const onLinkChange = useCallback(
    (e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
      setLink(e.target.value);
      setLinkError("");
    },
    [],
  );

  return (
    <DraggableDialog
      title={t("application:share.saveShareLink")}
      showActions
      loading={loading}
      showCancel
      onAccept={onOkClicked}
      dialogProps={{
        open,
        onClose: handleClose,
        fullWidth: true,
        maxWidth: "sm",
        disableRestoreFocus: true,
      }}
    >
      <DialogContent>
        <Stack spacing={2}>
          <form ref={formRef} onSubmit={onAccept}>
            <Stack spacing={2}>
              <FilledTextField
                variant="filled"
                autoFocus
                error={!!linkError}
                helperText={linkError || t("application:share.saveShareLinkHint")}
                margin="dense"
                label={t("application:share.shareLink")}
                type="text"
                value={link}
                onChange={onLinkChange}
                fullWidth
                required
              />
              <FilledTextField
                variant="filled"
                margin="dense"
                label={t("application:share.savedAs")}
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                fullWidth
              />
              <FilledTextField
                variant="filled"
                margin="dense"
                label={t("application:share.sharePassword")}
                type="text"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                fullWidth
              />
            </Stack>
          </form>
        </Stack>
      </DialogContent>
    </DraggableDialog>
  );
};

export default SaveShareLinkDialog;
