import { DialogContent, Link, Typography } from "@mui/material";
import { Trans, useTranslation } from "react-i18next";
import { Link as RouterLink } from "react-router-dom";
import DraggableDialog from "../../../Dialogs/DraggableDialog";
export interface StoreFilesHintDialogProps {
  open: boolean;
  onClose: () => void;
}

const StoreFilesHintDialog = ({ open, onClose }: StoreFilesHintDialogProps) => {
  const { t } = useTranslation("dashboard");
  return (
    <DraggableDialog
      dialogProps={{
        open,
        onClose,
        fullWidth: true,
        maxWidth: "sm",
      }}
      title={t("node.storeFiles")}
    >
      <DialogContent>
        <Typography variant="body2">
          <Trans
            i18nKey="node.storeFilesHint"
            ns="dashboard"
            components={[<Link component={RouterLink} to="/admin/policy" />]}
          />
        </Typography>
      </DialogContent>
    </DraggableDialog>
  );
};

export default StoreFilesHintDialog;
