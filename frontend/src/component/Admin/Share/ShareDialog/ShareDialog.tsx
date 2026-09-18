import { Box, DialogContent } from "@mui/material";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { CSSTransition, SwitchTransition } from "react-transition-group";
import { getShareDetail } from "../../../../api/api.ts";
import { Share } from "../../../../api/dashboard.ts";
import { useAppDispatch } from "../../../../redux/hooks.ts";
import AutoHeight from "../../../Common/AutoHeight.tsx";
import FacebookCircularProgress from "../../../Common/CircularProgress.tsx";
import DraggableDialog from "../../../Dialogs/DraggableDialog.tsx";
import ShareForm from "./ShareForm.tsx";

export interface ShareDialogProps {
  open: boolean;
  onClose: () => void;
  shareID?: number;
}

const ShareDialog = ({ open, onClose, shareID }: ShareDialogProps) => {
  const dispatch = useAppDispatch();
  const { t } = useTranslation("dashboard");
  const [values, setValues] = useState<Share>({ edges: {}, id: 0 });
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!shareID || !open) {
      return;
    }
    setLoading(true);
    dispatch(getShareDetail(shareID))
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
      title={t("share.shareDialogTitle")}
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
                {!loading && <ShareForm values={values} />}
              </Box>
            </CSSTransition>
          </SwitchTransition>
        </AutoHeight>
      </DialogContent>
    </DraggableDialog>
  );
};

export default ShareDialog;
