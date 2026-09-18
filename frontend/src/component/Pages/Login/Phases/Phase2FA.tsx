import { useTranslation } from "react-i18next";
import { useAppSelector } from "../../../../redux/hooks.ts";
import { Control } from "../Signin/SignIn.tsx";
import { FormControl, styled, Typography } from "@mui/material";
import { MuiOtpInput } from "mui-one-time-password-input";

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
  const regEnabled = useAppSelector((state) => state.siteConfig.login.config.register_enabled);
  return (
    <>
      <Typography color={"text.secondary"}>{t("login.input2FACode")}</Typography>
      <FormControl variant="standard" margin="normal" required fullWidth>
        <MuiOtpInputStyled
          TextFieldsProps={{ disabled: loading }}
          autoFocus
          length={6}
          value={otp}
          onChange={onOtpChange}
        />
      </FormControl>

      {control?.submit}
      {control?.back}
    </>
  );
};

export default Phase2FA;
