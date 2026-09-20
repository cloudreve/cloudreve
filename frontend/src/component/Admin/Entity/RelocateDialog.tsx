import { Button, Dialog, DialogActions, DialogContent, DialogContentText, DialogTitle } from "@mui/material";
import { useSnackbar } from "notistack";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { relocateEntities } from "../../../api/api";
import { useAppDispatch } from "../../../redux/hooks";
import { DefaultCloseAction } from "../../Common/Snackbar/snackbar";
import SinglePolicySelectionInput from "../Common/SinglePolicySelectionInput";

export interface RelocateDialogProps {
  open: boolean;
  onClose: () => void;
  entityIDs?: number[];
  srcPolicyID?: number;
  srcPolicyName?: string;
  srcUserID?: number;
  srcUserName?: string;
}

const RelocateDialog = ({ open, onClose, entityIDs, srcPolicyID, srcPolicyName, srcUserID, srcUserName }: RelocateDialogProps) => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const { enqueueSnackbar } = useSnackbar();
  const [dstPolicyID, setDstPolicyID] = useState<number | undefined>(undefined);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (open) {
      setDstPolicyID(undefined);
    }
  }, [open]);

  const onSubmit = () => {
    if (!dstPolicyID) {
      return;
    }
    setLoading(true);
    dispatch(
      relocateEntities({
        entity_ids: entityIDs,
        src_policy_id: srcPolicyID,
        src_user_id: srcUserID,
        dst_policy_id: dstPolicyID,
      }),
    )
      .then(() => {
        enqueueSnackbar(t("entity.relocateSubmitted"), { variant: "success", action: DefaultCloseAction });
        onClose();
      })
      .finally(() => {
        setLoading(false);
      });
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="xs" fullWidth>
      <DialogTitle>{t("entity.relocateTitle")}</DialogTitle>
      <DialogContent>
        <DialogContentText sx={{ mb: 2 }}>
          {srcPolicyID
            ? t("entity.relocatePolicyDes", { name: srcPolicyName ?? "" })
            : srcUserID
              ? t("entity.relocateUserDes", { name: srcUserName ?? "" })
              : t("entity.relocateEntitiesDes", { num: entityIDs?.length ?? 0 })}
        </DialogContentText>
        <SinglePolicySelectionInput value={dstPolicyID} onChange={setDstPolicyID} />
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t("common:cancel")}</Button>
        <Button variant="contained" onClick={onSubmit} disabled={loading || !dstPolicyID}>
          {t("entity.relocateStart")}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default RelocateDialog;
