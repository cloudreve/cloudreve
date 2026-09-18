import { Box, Divider, FormControl, Link, Stack } from "@mui/material";
import { useEffect } from "react";
import { Trans, useTranslation } from "react-i18next";
import { Link as RouterLink } from "react-router-dom";
import { LoginResponse } from "../../../../api/user.ts";
import { useAppSelector } from "../../../../redux/hooks.ts";
import { useQuery } from "../../../../util";
import { OutlineIconTextField } from "../../../Common/Form/OutlineIconTextField.tsx";
import MailOutlined from "../../../Icons/MailOutlined.tsx";
import PasskeyLoginButton from "../Signin/PasskeyLoginButton.tsx";
import { Control } from "../Signin/SignIn.tsx";

export const LegalLinks = () => {
  const { t } = useTranslation();
  const tos = useAppSelector((state) => state.siteConfig.login.config.tos_url);
  const privacyPolicy = useAppSelector((state) => state.siteConfig.login.config.privacy_policy_url);
  return (
    <>
      {(tos || privacyPolicy) && (
        <Box
          sx={{
            mt: 2,
            color: "text.secondary",
            typography: "caption",
            textAlign: "center",
          }}
        >
          {tos && (
            <Link target={"_blank"} underline="hover" color={"inherit"} href={tos}>
              {t("login.termOfUse")}
            </Link>
          )}
          {tos && privacyPolicy && " | "}
          {privacyPolicy && (
            <Link target={"_blank"} underline="hover" color={"inherit"} href={privacyPolicy}>
              {t("login.privacyPolicy")}
            </Link>
          )}
        </Box>
      )}
    </>
  );
};

interface PhaseCollectEmailProps {
  email: string;
  setEmail: (email: string) => void;
  control?: Control;
  onOAuthPasskeyLogin?: (response: LoginResponse) => void;
}

const PhaseCollectEmail = ({ email, setEmail, control, onOAuthPasskeyLogin }: PhaseCollectEmailProps) => {
  const { t } = useTranslation();
  const query = useQuery();
  const { register_enabled, authn } = useAppSelector((state) => state.siteConfig.login.config);
  const tos = useAppSelector((state) => state.siteConfig.login.config.tos_url);
  const privacyPolicy = useAppSelector((state) => state.siteConfig.login.config.privacy_policy_url);

  const showFooter = tos || privacyPolicy || authn;

  useEffect(() => {
    if (!!query.get("email")) {
      setEmail(query.get("email") ?? "");
    }
  }, []);

  return (
    <>
      <FormControl variant="standard" margin="normal" required fullWidth>
        <OutlineIconTextField
          label={t("login.email")}
          variant={"outlined"}
          inputProps={{
            id: "email",
            type: "email",
            name: "email",
            required: "true",
          }}
          onChange={(e) => setEmail(e.target.value)}
          icon={<MailOutlined />}
          autoComplete={"username webauthn"}
          value={email}
          autoFocus
        />
      </FormControl>
      {control?.submit}
      {control?.back}
      {register_enabled && (
        <Box sx={{ mt: 2, typography: "body2", textAlign: "center" }}>
          <Trans
            ns={"application"}
            i18nKey={"login.noAccountSignupNow"}
            components={[<Link underline="hover" component={RouterLink} to="/session/signup" />]}
          />
        </Box>
      )}
      {showFooter && (
        <>
          <Divider sx={{ my: 2 }} />
          <Stack spacing={1}>{authn && <PasskeyLoginButton autoComplete onLoginSuccess={onOAuthPasskeyLogin} />}</Stack>
          <LegalLinks />
        </>
      )}
    </>
  );
};

export default PhaseCollectEmail;
