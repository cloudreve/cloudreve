import { Divider, FormControl, Link, Stack, Typography } from "@mui/material";
import { useTranslation } from "react-i18next";
import { LoginResponse, PrepareLoginResponse } from "../../../../api/user.ts";
import { useAppSelector } from "../../../../redux/hooks.ts";
import { Captcha, CaptchaParams } from "../../../Common/Captcha/Captcha.tsx";
import { OutlineIconTextField } from "../../../Common/Form/OutlineIconTextField.tsx";
import Password from "../../../Icons/Password.tsx";
import PasskeyLoginButton from "../Signin/PasskeyLoginButton.tsx";
import { Control } from "../Signin/SignIn.tsx";

interface PhaseCollectPasswordProps {
  pwd: string;
  email: string;
  setPwd: (pwd: string) => void;
  control?: Control;
  loginOptions?: PrepareLoginResponse;
  captchaGen: number;
  setCaptchaState: (state: CaptchaParams) => void;
  onForget?: () => void;
  onOAuthPasskeyLogin?: (response: LoginResponse) => void;
}

const PhaseCollectPassword = ({
  pwd,
  email,
  setPwd,
  control,
  loginOptions,
  captchaGen,
  setCaptchaState,
  onForget,
  onOAuthPasskeyLogin,
}: PhaseCollectPasswordProps) => {
  const { t } = useTranslation();
  const { login_captcha, authn } = useAppSelector((state) => state.siteConfig.login.config);

  const moreOptions = loginOptions?.webauthn_enabled && authn;
  return (
    <>
      {loginOptions?.password_enabled && (
        <>
          <Typography color={"text.secondary"}>{t("login.enterPasswordHint", { email: email })}</Typography>
          <FormControl variant="standard" margin="normal" required fullWidth>
            <OutlineIconTextField
              autoFocus={true}
              variant={"outlined"}
              label={t("login.password")}
              inputProps={{
                name: "password",
                type: "password",
                id: "password",
                required: "true",
                maxLength: 128,
                minLength: 4,
              }}
              onChange={(e) => setPwd(e.target.value)}
              icon={<Password />}
              value={pwd}
              autoComplete={"true"}
            />
            <Link sx={{ mt: 1 }} underline="hover" variant="body2" href={"#"} onClick={onForget}>
              {t("login.forgetPassword")}
            </Link>
          </FormControl>
          {login_captcha && (
            <FormControl variant="standard" margin="normal" required fullWidth>
              <Captcha generation={captchaGen} required={true} fullWidth={true} onStateChange={setCaptchaState} />
            </FormControl>
          )}
          {control?.submit}
          {moreOptions && (
            <Divider sx={{ my: 2 }} role="presentation">
              <Typography variant="body2" color={"text.secondary"}>
                {t("login.or")}
              </Typography>
            </Divider>
          )}
        </>
      )}
      {!loginOptions?.password_enabled && (
        <Typography color={"text.secondary"} sx={{ mb: 2 }}>
          {t("login.paswordlessHint", { email: email })}
        </Typography>
      )}
      <Stack spacing={1}>
        {loginOptions?.webauthn_enabled && authn && <PasskeyLoginButton onLoginSuccess={onOAuthPasskeyLogin} />}
      </Stack>
      {control?.back}
    </>
  );
};

export default PhaseCollectPassword;
