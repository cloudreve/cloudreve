import DraggableDialog from "../../../Dialogs/DraggableDialog.tsx";
import { useTranslation } from "react-i18next";
import { useSnackbar } from "notistack";
import { useAppDispatch, useAppSelector } from "../../../../redux/hooks.ts";
import React, { useEffect, useState } from "react";
import { Box, DialogContent, FormControl, Stack, styled, Typography, useMediaQuery, useTheme } from "@mui/material";
import { CSSTransition, SwitchTransition } from "react-transition-group";
import AutoHeight from "../../../Common/AutoHeight.tsx";
import FacebookCircularProgress from "../../../Common/CircularProgress.tsx";
import { get2FAInitSecret, sendUpdateUserSetting } from "../../../../api/api.ts";
import { QRCodeSVG } from "qrcode.react";
import SessionManager from "../../../../session";
import { MuiOtpInput } from "mui-one-time-password-input";

export interface Enable2FADialogProps {
  open?: boolean;
  onClose: () => void;
  on2FAEnabled: () => void;
}

const MuiOtpInputStyled = styled(MuiOtpInput)`
  display: flex;
  gap: 8px;
  max-width: 650px;
  margin-inline: auto;
`;

const Enable2FADialog = ({ open, onClose, on2FAEnabled }: Enable2FADialogProps) => {
  const { t } = useTranslation();
  const { enqueueSnackbar } = useSnackbar();
  const dispatch = useAppDispatch();
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("md"));
  const siteTitle = useAppSelector((state) => state.siteConfig.basic.config.title);
  const user = SessionManager.currentLoginOrNull();

  const [loading, setLoading] = useState(false);
  const [code, setCode] = useState("");
  const [secret, setSecret] = useState("");

  useEffect(() => {
    if (open) {
      setLoading(true);
      setCode("");
      dispatch(get2FAInitSecret())
        .then((res) => {
          setSecret(res);
        })
        .finally(() => {
          setLoading(false);
        });
    }
  }, [open]);

  useEffect(() => {
    if (code.length === 6) {
      setLoading(true);
      dispatch(
        sendUpdateUserSetting({
          two_fa_enabled: true,
          two_fa_code: code,
        }),
      )
        .then(() => {
          enqueueSnackbar({
            message: t("setting.settingSaved"),
            variant: "success",
          });
          on2FAEnabled();
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
      title={t("application:setting.enable2FA")}
      showCancel
      hideOk
      showActions
      dialogProps={{
        open: !!open,
        onClose: onClose,
        fullWidth: true,
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
                  <Stack spacing={1} direction={isMobile ? "column" : "row"}>
                    <Box
                      sx={{
                        minWidth: 176,
                        textAlign: "center",
                        p: 1,
                        backgroundColor: "#fff",
                      }}
                    >
                      <QRCodeSVG
                        size={isMobile ? 144 : 160}
                        value={`otpauth://totp/${siteTitle}:${user?.user?.nickname}?secret=${secret}`}
                      />
                    </Box>
                    <Box>
                      <Typography variant={"body2"} sx={{ mt: 1 }}>
                        {t("setting.2faDescription")}
                      </Typography>
                      <Typography variant={"body2"} sx={{ mt: 1 }}>
                        {t("setting.inputCurrent2FACode")}
                      </Typography>
                      <FormControl variant="standard" margin="normal" required>
                        <MuiOtpInputStyled
                          TextFieldsProps={{ disabled: loading }}
                          autoFocus
                          length={6}
                          value={code}
                          onChange={setCode}
                        />
                      </FormControl>
                    </Box>
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

export default Enable2FADialog;
