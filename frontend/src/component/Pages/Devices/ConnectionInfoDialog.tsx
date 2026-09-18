import { useTranslation } from "react-i18next";
import { DialogContent, DialogProps, FilledInput, IconButton, InputAdornment, InputLabel, Stack } from "@mui/material";
import { useAppDispatch } from "../../../redux/hooks.ts";
import DraggableDialog from "../../Dialogs/DraggableDialog.tsx";
import { DavAccount } from "../../../api/setting.ts";
import FormControl from "@mui/material/FormControl";
import CopyOutlined from "../../Icons/CopyOutlined.tsx";
import { copyToClipboard } from "../../../util";
import SessionManager from "../../../session";

export interface ConnectionInfoDialogProps extends DialogProps {
  account?: DavAccount;
}

const InfoTextField = ({ label, value }: { label: string; value: string }) => {
  return (
    <FormControl variant="filled">
      <InputLabel>{label}</InputLabel>
      <FilledInput
        sx={{
          "&.MuiFilledInput-root": {
            paddingRight: "20px",
          },
        }}
        value={value}
        fullWidth
        endAdornment={
          <InputAdornment position="end">
            <IconButton edge="end" onClick={() => copyToClipboard(value)}>
              <CopyOutlined />
            </IconButton>
          </InputAdornment>
        }
        inputProps={{
          readOnly: true,
        }}
      />
    </FormControl>
  );
};

const ConnectionInfoDialog = ({ onClose, account, ...rest }: ConnectionInfoDialogProps) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();

  return (
    <DraggableDialog
      title={t("application:setting.connectionInfo")}
      showActions
      onAccept={() => onClose && onClose({}, "escapeKeyDown")}
      dialogProps={{
        onClose: onClose,
        fullWidth: true,
        maxWidth: "xs",
        disableRestoreFocus: true,
        ...rest,
      }}
    >
      <DialogContent sx={{ pt: 1 }}>
        <Stack spacing={2} direction={"column"}>
          <InfoTextField label={t("application:setting.webdavServer")} value={window.location.origin + "/dav"} />
          <InfoTextField
            label={t("application:setting.userName")}
            value={SessionManager.currentLoginOrNull()?.user.email ?? ""}
          />
          <InfoTextField label={t("application:login.password")} value={account?.password ?? ""} />
        </Stack>
      </DialogContent>
    </DraggableDialog>
  );
};
export default ConnectionInfoDialog;
