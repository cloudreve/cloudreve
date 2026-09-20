import DraggableDialog from "../../../Dialogs/DraggableDialog.tsx";
import { useTranslation } from "react-i18next";
import { useSnackbar } from "notistack";
import { useAppDispatch } from "../../../../redux/hooks.ts";
import { useEffect, useState } from "react";
import { Box, DialogContent, FormControl, Grid2, IconButton, Stack, styled, Tooltip, Typography } from "@mui/material";
import { CSSTransition, SwitchTransition } from "react-transition-group";
import AutoHeight from "../../../Common/AutoHeight.tsx";
import FacebookCircularProgress from "../../../Common/CircularProgress.tsx";
import { regenerate2FABackupCodes } from "../../../../api/api.ts";
import { copyToClipboard } from "../../../../util";
import { MuiOtpInput } from "mui-one-time-password-input";
import CopyOutlined from "../../../Icons/CopyOutlined.tsx";

export interface Backup2FACodesDialogProps {
  open?: boolean;
  onClose: () => void;
  onCodesRegenerated: (count: number) => void;
}

const MuiOtpInputStyled = styled(MuiOtpInput)`
  display: flex;
  gap: 8px;
  max-width: 650px;
  margin-inline: auto;
`;

const Backup2FACodesDialog = ({ open, onClose, onCodesRegenerated }: Backup2FACodesDialogProps) => {
  const { t } = useTranslation();
  const { enqueueSnackbar } = useSnackbar();
  const dispatch = useAppDispatch();

  const [loading, setLoading] = useState(false);
  const [code, setCode] = useState("");
  const [codes, setCodes] = useState<string[] | null>(null);

  useEffect(() => {
    if (open) {
      setLoading(false);
      setCode("");
      setCodes(null);
    }
  }, [open]);

  useEffect(() => {
    if (code.length === 6 && !codes) {
      setLoading(true);
      dispatch(regenerate2FABackupCodes(code))
        .then((res) => {
          setCodes(res);
          onCodesRegenerated(res.length);
        })
        .catch(() => {
          setCode("");
        })
        .finally(() => {
          setLoading(false);
        });
    }
  }, [code]);

  return (
    <DraggableDialog
      title={t("application:setting.backup2FACodes")}
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
              key={`${loading}-${codes != null}`}
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
                {!loading && !codes && (
                  <Stack spacing={1}>
                    <Typography variant={"body2"}>{t("setting.backupCodesDes")}</Typography>
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
                {!loading && codes && (
                  <Stack spacing={1}>
                    <Typography variant={"body2"} color="warning.main">
                      {t("setting.backupCodesWarning")}
                    </Typography>
                    <Grid2 container spacing={0.5} sx={{ fontFamily: "monospace" }}>
                      {codes.map((c) => (
                        <Grid2 key={c} size={6}>
                          {c}
                        </Grid2>
                      ))}
                    </Grid2>
                    <Box sx={{ display: "flex", justifyContent: "flex-end" }}>
                      <Tooltip title={t("setting.copyCodes")}>
                        <IconButton
                          size="small"
                          onClick={() => {
                            copyToClipboard(codes.join("\n"));
                            enqueueSnackbar({ message: t("setting.copied"), variant: "success" });
                          }}
                        >
                          <CopyOutlined fontSize="small" />
                        </IconButton>
                      </Tooltip>
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

export default Backup2FACodesDialog;
