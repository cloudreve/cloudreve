import { useTranslation } from "react-i18next";
import { Alert, Stack, Typography } from "@mui/material";
import { useAppSelector } from "../../../../redux/hooks.ts";
import { Control } from "../Signin/SignIn.tsx";

interface SignupNeededProps {
  email: string;
  control?: Control;
}

const SignupNeeded = ({ email, control }: SignupNeededProps) => {
  const { t } = useTranslation();
  const regEnabled = useAppSelector((state) => state.siteConfig.login.config.register_enabled);
  return (
    <>
      {regEnabled && <Typography color={"text.secondary"}>{t("login.signupHint", { email: email })}</Typography>}
      {!regEnabled && <Alert severity={"warning"}>{t("login.accountNotFoundHint", { email: email })}</Alert>}
      <Stack spacing={1} sx={{ mt: 1 }}>
        {regEnabled && control?.submit}
      </Stack>
      {control?.back}
    </>
  );
};

export default SignupNeeded;
