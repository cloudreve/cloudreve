import { Icon } from "@iconify/react";
import { Button, ButtonProps } from "@mui/material";
import { useTranslation } from "react-i18next";
import { ApiPrefix } from "../../../../api/request.ts";
import { useAppSelector } from "../../../../redux/hooks.ts";
import { useQuery } from "../../../../util";

export default function QQLoginButton(props: ButtonProps) {
  const { t } = useTranslation();
  const query = useQuery();
  const { qq_connect_enabled } = useAppSelector((state) => state.siteConfig.login.config);

  if (!qq_connect_enabled) {
    return null;
  }

  const startLogin = () => {
    const redirect = query.get("redirect");
    const target = new URL(ApiPrefix + "/session/qq/login", window.location.origin);
    if (redirect) {
      target.searchParams.set("redirect", redirect);
    }
    window.location.assign(target.toString());
  };

  return (
    <Button fullWidth variant="outlined" startIcon={<Icon icon="ri:qq-fill" />} onClick={startLogin} {...props}>
      {t("login.signInWith", { name: "QQ" })}
    </Button>
  );
}
