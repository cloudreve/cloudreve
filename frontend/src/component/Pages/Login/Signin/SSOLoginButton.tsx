import { Button, ButtonProps } from "@mui/material";
import { useTranslation } from "react-i18next";
import { ApiPrefix } from "../../../../api/request.ts";
import { useAppSelector } from "../../../../redux/hooks.ts";
import { useQuery } from "../../../../util";
import Enter from "../../../Icons/Enter.tsx";

export default function SSOLoginButton(props: ButtonProps) {
  const { t } = useTranslation();
  const query = useQuery();
  const { sso_enabled, sso_display_name } = useAppSelector((state) => state.siteConfig.login.config);

  if (!sso_enabled) {
    return null;
  }

  const startLogin = () => {
    const redirect = query.get("redirect");
    const target = new URL(ApiPrefix + "/session/sso", window.location.origin);
    if (redirect) {
      target.searchParams.set("redirect", redirect);
    }
    window.location.assign(target.toString());
  };

  return (
    <Button fullWidth variant="outlined" startIcon={<Enter />} onClick={startLogin} {...props}>
      {t("login.signInWith", { name: sso_display_name || "SSO" })}
    </Button>
  );
}
