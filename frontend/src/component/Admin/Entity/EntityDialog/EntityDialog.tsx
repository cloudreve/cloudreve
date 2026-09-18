import { Box, DialogContent } from "@mui/material";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { CSSTransition, SwitchTransition } from "react-transition-group";
import { getEntityDetail } from "../../../../api/api.ts";
import { Entity } from "../../../../api/dashboard.ts";
import { useAppDispatch } from "../../../../redux/hooks.ts";
import AutoHeight from "../../../Common/AutoHeight.tsx";
import FacebookCircularProgress from "../../../Common/CircularProgress.tsx";
import DraggableDialog from "../../../Dialogs/DraggableDialog.tsx";
import EntityForm from "./EntityForm.tsx";

export interface EntityDialogProps {
  open: boolean;
  onClose: () => void;
  entityID?: number;
}

const EntityDialog = ({ open, onClose, entityID }: EntityDialogProps) => {
  const dispatch = useAppDispatch();
  const { t } = useTranslation("dashboard");
  const [values, setValues] = useState<Entity>({ edges: {}, id: 0 });
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!entityID || !open) {
      return;
    }
    setLoading(true);
    dispatch(getEntityDetail(entityID))
      .then((res) => {
        setValues(res);
      })
      .catch(() => {
        onClose();
      })
      .finally(() => {
        setLoading(false);
      });
  }, [open]);

  return (
    <DraggableDialog
      title={t("entity.entityDialogTitle")}
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
                {!loading && <EntityForm values={values} />}
              </Box>
            </CSSTransition>
          </SwitchTransition>
        </AutoHeight>
      </DialogContent>
    </DraggableDialog>
  );
};

export default EntityDialog;
