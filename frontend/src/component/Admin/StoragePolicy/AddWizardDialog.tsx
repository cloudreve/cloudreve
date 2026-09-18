import { Box, DialogContent } from "@mui/material";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { CSSTransition, SwitchTransition } from "react-transition-group";
import { upsertStoragePolicy } from "../../../api/api";
import { StoragePolicy } from "../../../api/dashboard";
import { PolicyType } from "../../../api/explorer";
import { useAppDispatch } from "../../../redux/hooks";
import AutoHeight from "../../Common/AutoHeight";
import FacebookCircularProgress from "../../Common/CircularProgress";
import DraggableDialog from "../../Dialogs/DraggableDialog";
import { PolicyPropsMap } from "./StoragePolicySetting";

export interface AddWizardDialogProps {
  open: boolean;
  onClose: () => void;
  type: PolicyType;
}

export interface AddWizardProps {
  onSubmit: (data: StoragePolicy) => void;
}

const AddWizardDialog = ({ open, onClose, type }: AddWizardDialogProps) => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();
  const Wizard = PolicyPropsMap[type].wizard;

  const onSubmit = useCallback(
    (data: StoragePolicy) => {
      setLoading(true);
      dispatch(upsertStoragePolicy({ policy: data }))
        .then((res) => {
          onClose();
          navigate(`/admin/policy/${res.id}`);
          setLoading(false);
        })
        .finally(() => {
          setLoading(false);
        });
    },
    [dispatch],
  );

  return (
    <DraggableDialog
      loading={loading}
      title={t("policy.addXStoragePolicy", {
        type: t(PolicyPropsMap[type].name),
      })}
      dialogProps={{ open, onClose, fullWidth: true, maxWidth: PolicyPropsMap[type].wizardSize ?? "lg" }}
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
                {!loading && Wizard && <Wizard onSubmit={onSubmit} />}
                {loading && (
                  <Box
                    sx={{
                      height: "300px",
                      display: "flex",
                      alignItems: "center",
                      justifyContent: "center",
                    }}
                  >
                    <FacebookCircularProgress />
                  </Box>
                )}
              </Box>
            </CSSTransition>
          </SwitchTransition>
        </AutoHeight>
      </DialogContent>
    </DraggableDialog>
  );
};

export default AddWizardDialog;
