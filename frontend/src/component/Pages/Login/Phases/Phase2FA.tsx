import { useTranslation } from "react-i18next";
import { Control } from "../Signin/SignIn.tsx";
import { Button, FormControl, styled, TextField, Typography } from "@mui/material";
import { MuiOtpInput } from "mui-one-time-password-input";
import { useState } from "react";

interface Phase2FAProps {
  control?: Control;
  otp: string;
  onOtpChange: (otp: string) => void;
  loading: boolean;
}

const MuiOtpInputStyled = styled(MuiOtpInput)`
  display: flex;
  gap: 8px;
  max-width: 650px;
  margin-inline: auto;
`;

const Phase2FA = ({ control, otp, onOtpChange, loading }: Phase2FAProps) => {
  const { t } = useTranslation();
  const [useBackup, setUseBackup] = useState(false);

  const toggleMode = () => {
    setUseBackup((v) => !v);
    onOtpChange("");
  };

  return (
    <>
      <Typography color={"text.secondary"}>
        {useBackup ? t("login.inputBackupCode") : t("login.input2FACode")}
      </Typography>
      <FormControl variant="standard" margin="normal" required fullWidth>
        {useBackup ? (
          <TextField
            variant="standard"
            autoFocus
            fullWidth
            disabled={loading}
            placeholder="xxxx-xxxx"
            value={otp}
            onChange={(e) => onOtpChange(e.target.value)}
            slotProps={{ htmlInput: { style: { textAlign: "center", letterSpacing: 2 } } }}
          />
        ) : (
          <MuiOtpInputStyled
            TextFieldsProps={{ disabled: loading }}
            autoFocus
            length={6}
            value={otp}
            onChange={onOtpChange}
          />
        )}
      </FormControl>
      <Button size="small" onClick={toggleMode} disabled={loading} sx={{ mb: 1 }}>
        {useBackup ? t("login.use2FACode") : t("login.useBackupCode")}
      </Button>

      {control?.submit}
      {control?.back}
    </>
  );
};

export default Phase2FA;
