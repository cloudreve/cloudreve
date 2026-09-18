import DraggableDialog from "../../../Dialogs/DraggableDialog.tsx";
import { useTranslation } from "react-i18next";
import { useSnackbar } from "notistack";
import { useAppDispatch } from "../../../../redux/hooks.ts";
import React, { useEffect, useState } from "react";
import { Box, DialogContent, FormControl, Stack, styled, Typography } from "@mui/material";
import { CSSTransition, SwitchTransition } from "react-transition-group";
import AutoHeight from "../../../Common/AutoHeight.tsx";
import FacebookCircularProgress from "../../../Common/CircularProgress.tsx";
import { sendUpdateUserSetting } from "../../../../api/api.ts";
import { MuiOtpInput } from "mui-one-time-password-input";

export interface Disable2FADialogProps {
  open?: boolean;
  onClose: () => void;
  on2FADisabled: () => void;
}

const MuiOtpInputStyled = styled(MuiOtpInput)`
  display: flex;
  gap: 8px;
  max-width: 650px;
  margin-inline: auto;
`;

const Disable2FADialog = ({ open, onClose, on2FADisabled }: Disable2FADialogProps) => {
  const { t } = useTranslation();
  const { enqueueSnackbar } = useSnackbar();
  const dispatch = useAppDispatch();

  const [loading, setLoading] = useState(false);
  const [code, setCode] = useState("");

  useEffect(() => {
    if (open) {
      setLoading(false);
      setCode("");
    }
  }, [open]);

  useEffect(() => {
    if (code.length === 6) {
      setLoading(true);
      dispatch(
        sendUpdateUserSetting({
          two_fa_enabled: false,
          two_fa_code: code,
        }),
      )
        .then(() => {
          enqueueSnackbar({
            message: t("setting.settingSaved"),
            variant: "success",
          });
          on2FADisabled();
          onClose();
        })
        .catch(() => {
          setLoading(false);
          setCode("");
        });
    }
  }, [code]);

  return (
    <DraggableDialog
      title={t("application:setting.disable2FA")}
      showCancel
      hideOk
      showActions
      dialogProps={{
        open: !!open,
        onClose: onClose,
        fullWidth: true,
        maxWidth: "xs",
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
                      pt: 3,
                      height: "100%",
                      display: "flex",
                      justifyContent: "center",
                      alignItems: "center",
                    }}
                  >
                    <FacebookCircularProgress />
                  </Box>
                )}
                {!loading && (
                  <Stack spacing={1}>
                    <Typography variant={"body2"}>{t("setting.inputCurrent2FACode")}</Typography>
                    <FormControl variant="standard" margin="normal" required>
                      <MuiOtpInputStyled
                        TextFieldsProps={{ disabled: loading }}
                        autoFocus
                        length={6}
                        value={code}
                        onChange={setCode}
                      />
                    </FormControl>
                  </Stack>
                )}
              </Box>
            </CSSTransition>
          </SwitchTransition>
        </AutoHeight>
      </DialogContent>
    </DraggableDialog>
  );
};

export default Disable2FADialog;
