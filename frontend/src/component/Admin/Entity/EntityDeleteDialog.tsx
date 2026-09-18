import { Checkbox, DialogContent, FormGroup, Stack, Tooltip } from "@mui/material";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { batchDeleteEntities } from "../../../api/api.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { SmallFormControlLabel } from "../../Common/StyledComponents.tsx";
import DialogAccordion from "../../Dialogs/DialogAccordion.tsx";
import DraggableDialog, { StyledDialogContentText } from "../../Dialogs/DraggableDialog.tsx";

export interface EntityDeleteDialogProps {
  entityID?: number[];
  open: boolean;
  onClose?: () => void;
  onDelete?: () => void;
}

const EntityDeleteDialog = ({ entityID, open, onDelete, onClose }: EntityDeleteDialogProps) => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();

  const [force, setForce] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const onAccept = useCallback(() => {
    if (entityID) {
      setDeleting?.(true);
      dispatch(batchDeleteEntities({ ids: entityID, force }))
        .then(() => {
          onDelete?.();
          onClose?.();
        })
        .finally(() => {
          setDeleting?.(false);
        });
    }
  }, [entityID, force, setDeleting]);

  return (
    <DraggableDialog
      title={t("common:areYouSure")}
      showActions
      loading={deleting}
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
        <Stack spacing={2}>
          <StyledDialogContentText>{t("entity.confirmBatchDelete", { num: entityID?.length })}</StyledDialogContentText>
          <DialogAccordion defaultExpanded={force} title={t("application:modals.advanceOptions")}>
            <FormGroup>
              <Tooltip title={t("entity.forceDeleteDes")}>
                <SmallFormControlLabel
                  control={<Checkbox size="small" onChange={(e) => setForce(e.target.checked)} checked={force} />}
                  label={t("entity.forceDelete")}
                />
              </Tooltip>
            </FormGroup>
          </DialogAccordion>
        </Stack>
      </DialogContent>
    </DraggableDialog>
  );
};
export default EntityDeleteDialog;
