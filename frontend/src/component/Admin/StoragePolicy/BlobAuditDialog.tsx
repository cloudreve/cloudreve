import {
  Button,
  Checkbox,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  FormControlLabel,
} from "@mui/material";
import { useSnackbar } from "notistack";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { sendBlobAuditTask } from "../../../api/api";
import { StoragePolicy } from "../../../api/dashboard";
import { useAppDispatch } from "../../../redux/hooks";
import { DefaultCloseAction } from "../../Common/Snackbar/snackbar";

export interface BlobAuditDialogProps {
  open: boolean;
  onClose: () => void;
  policy?: StoragePolicy;
}

const BlobAuditDialog = ({ open, onClose, policy }: BlobAuditDialogProps) => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const { enqueueSnackbar } = useSnackbar();
  const [deleteOrphans, setDeleteOrphans] = useState(false);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (open) {
      setDeleteOrphans(false);
    }
  }, [open]);

  const onSubmit = () => {
    if (!policy?.id) {
      return;
    }
    setLoading(true);
    dispatch(sendBlobAuditTask({ policy_id: policy.id, delete: deleteOrphans }))
      .then(() => {
        enqueueSnackbar(t("policy.blobAuditSubmitted"), { variant: "success", action: DefaultCloseAction });
        onClose();
      })
      .finally(() => {
        setLoading(false);
      });
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="xs" fullWidth>
      <DialogTitle>{t("policy.blobAudit", { name: policy?.name ?? "" })}</DialogTitle>
      <DialogContent>
        <DialogContentText sx={{ mb: 2 }}>{t("policy.blobAuditDes")}</DialogContentText>
        <FormControlLabel
          control={
            <Checkbox
              size="small"
              checked={deleteOrphans}
              onChange={(e) => setDeleteOrphans(e.target.checked)}
            />
          }
          label={t("policy.blobAuditDelete")}
        />
        {deleteOrphans && (
          <DialogContentText color="error" variant="body2">
            {t("policy.blobAuditDeleteWarning")}
          </DialogContentText>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t("common:cancel")}</Button>
        <Button variant="contained" color={deleteOrphans ? "error" : "primary"} onClick={onSubmit} disabled={loading}>
          {t("policy.blobAuditStart")}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default BlobAuditDialog;
