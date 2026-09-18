import { LoadingButton } from "@mui/lab";
import { Box, FormControl, Link, Typography } from "@mui/material";
import { useSnackbar } from "notistack";
import { useRef, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { Link as RouterLink, useNavigate } from "react-router-dom";
import { sendReset } from "../../../api/api.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import PageTitle from "../../../router/PageTitle.tsx";
import { useQuery } from "../../../util";
import { OutlineIconTextField } from "../../Common/Form/OutlineIconTextField.tsx";
import { DefaultCloseAction } from "../../Common/Snackbar/snackbar.tsx";
import Password from "../../Icons/Password.tsx";

const Reset = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { enqueueSnackbar } = useSnackbar();
  const dispatch = useAppDispatch();
  const query = useQuery();

  const formRef = useRef<HTMLFormElement>(null);
  const [password, setPassword] = useState("");
  const [passwordRepeat, setPasswordRepeat] = useState("");
  const [loading, setLoading] = useState(false);

  const submit = () => {
    if (!formRef.current) {
      return;
    }

    if (!formRef.current.checkValidity()) {
      formRef.current.reportValidity();
      return;
    }

    if (passwordRepeat != password) {
      enqueueSnackbar({
        message: t("login.passwordNotMatch"),
        variant: "warning",
        action: DefaultCloseAction,
      });
      return;
    }

    setLoading(true);
    dispatch(
      sendReset(query.get("id") ?? "0", {
        password,
        secret: query.get("secret") ?? "",
      }),
    )
      .then((u) => {
        navigate("/session?phase=email&email=" + encodeURIComponent(u?.email ?? ""));
        enqueueSnackbar({
          message: t("login.passwordReset"),
          variant: "success",
          action: DefaultCloseAction,
        });
      })
      .finally(() => {
        setLoading(false);
      });
  };

  return (
    <Box>
      <PageTitle title={t("login.repeatPassword")} />
      <Box sx={{ overflow: "hidden" }}>
        <Typography variant={"h6"}>{t("login.repeatPassword")}</Typography>
        <Box component={"form"} sx={{ mt: 1 }} ref={formRef}>
          <Box>
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
                  minLength: 6,
                }}
                onChange={(e) => setPassword(e.target.value)}
                icon={<Password />}
                value={password}
                autoComplete={"false"}
              />
            </FormControl>
            <FormControl variant="standard" margin="normal" required fullWidth>
              <OutlineIconTextField
                autoFocus={true}
                variant={"outlined"}
                label={t("login.repeatPassword")}
                inputProps={{
                  name: "password",
                  type: "password",
                  id: "password",
                  required: "true",
                  minLength: 6,
                }}
                onChange={(e) => setPasswordRepeat(e.target.value)}
                icon={<Password />}
                value={passwordRepeat}
                autoComplete={"false"}
              />
            </FormControl>
            <LoadingButton
              sx={{ mt: 2 }}
              onClick={submit}
              fullWidth
              variant="contained"
              color="primary"
              loading={loading}
            >
              <span>{t("login.resetPassword")}</span>
            </LoadingButton>
            <Box sx={{ mt: 2, typography: "body2", textAlign: "center" }}>
              <Trans
                ns={"application"}
                i18nKey={"login.haveAccountSignInNow"}
                components={[<Link underline="hover" component={RouterLink} to="/session" />]}
              />
            </Box>
          </Box>
        </Box>
      </Box>
    </Box>
  );
};

export default Reset;
