import { LoadingButton } from "@mui/lab";
import { Box, Divider, FormControl, Link, Typography } from "@mui/material";
import i18next from "i18next";
import { useSnackbar } from "notistack";
import { useEffect, useRef, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { Link as RouterLink, useNavigate } from "react-router-dom";
import { CSSTransition, SwitchTransition } from "react-transition-group";
import { sendSinUp } from "../../../api/api.ts";
import { AppError, Code } from "../../../api/request.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import PageTitle from "../../../router/PageTitle.tsx";
import { useQuery } from "../../../util";
import { Captcha, CaptchaParams } from "../../Common/Captcha/Captcha.tsx";
import { OutlineIconTextField } from "../../Common/Form/OutlineIconTextField.tsx";
import { DefaultCloseAction } from "../../Common/Snackbar/snackbar.tsx";
import EmailClock from "../../Icons/EmailClock.tsx";
import MailOutlined from "../../Icons/MailOutlined.tsx";
import Password from "../../Icons/Password.tsx";
import { LegalLinks } from "./Phases/PhaseCollectEmail.tsx";

export enum SignUpPhase {
  Main,
  EmailActivation,
}

interface signUpPhaseProps {
  title: string;
}

const signUpPhaseSettings: Record<SignUpPhase, signUpPhaseProps> = {
  [SignUpPhase.Main]: {
    title: "login.createNewAccount",
  },
  [SignUpPhase.EmailActivation]: {
    title: "login.lastStep",
  },
};

const SignUp = () => {
  const { t } = useTranslation();
  const { reg_captcha } = useAppSelector((state) => state.siteConfig.login.config);
  const tos = useAppSelector((state) => state.siteConfig.login.config.tos_url);
  const privacyPolicy = useAppSelector((state) => state.siteConfig.login.config.privacy_policy_url);
  const navigate = useNavigate();
  const { enqueueSnackbar } = useSnackbar();
  const dispatch = useAppDispatch();
  const query = useQuery();

  const [phase, setPhase] = useState<SignUpPhase>(SignUpPhase.Main);
  const formRef = useRef<HTMLFormElement>(null);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [passwordRepeat, setPasswordRepeat] = useState("");
  const [captchaGen, setCaptchaGen] = useState(0);
  const captchaState = useRef<CaptchaParams>();
  const [loading, setLoading] = useState(false);

  const showFooter = tos || privacyPolicy;

  useEffect(() => {
    if (!!query.get("email")) {
      setEmail(query.get("email") ?? "");
    }
  }, []);

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
      sendSinUp({
        email,
        password,
        language: i18next.language,
        ...captchaState.current,
      }),
    )
      .then(() => {
        navigate("/session?phase=email&email=" + encodeURIComponent(email));
        enqueueSnackbar({
          message: t("login.signUpSuccess"),
          variant: "success",
          action: DefaultCloseAction,
        });
      })
      .catch((e) => {
        if (e instanceof AppError && e.code === Code.Continue) {
          setPhase(SignUpPhase.EmailActivation);
        } else {
          setCaptchaGen((g) => g + 1);
        }
      })
      .finally(() => {
        setLoading(false);
      });
  };

  return (
    <Box>
      <PageTitle title={t("login.signUp")} />
      <Box sx={{ overflow: "hidden" }}>
        <Typography variant={"h6"}>{t(signUpPhaseSettings[phase].title)}</Typography>
        <Box component={"form"} sx={{ mt: 1 }} ref={formRef}>
          <SwitchTransition>
            <CSSTransition
              addEndListener={(node, done) => node.addEventListener("transitionend", done, false)}
              classNames="side"
              key={phase}
            >
              <Box>
                {phase === SignUpPhase.EmailActivation && (
                  <Box
                    sx={{
                      display: "flex",
                      flexDirection: "column",
                      alignItems: "center",
                      pt: 3,
                      pb: 2,
                    }}
                  >
                    <EmailClock
                      sx={{
                        fontSize: 60,
                        color: (t) => t.palette.action.disabled,
                      }}
                    />
                    <Typography
                      sx={{
                        mt: 3,
                      }}
                      variant={"body2"}
                      color={"textSecondary"}
                    >
                      {t("application:login.activateDescription")}
                    </Typography>
                  </Box>
                )}
                {phase === SignUpPhase.Main && (
                  <Box>
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
                        value={email}
                        autoFocus={true}
                      />
                    </FormControl>
                    <FormControl variant="standard" margin="normal" required fullWidth>
                      <OutlineIconTextField
                        variant={"outlined"}
                        label={t("login.password")}
                        inputProps={{
                          name: "password",
                          type: "password",
                          id: "password",
                          required: "true",
                          minLength: 6,
                          maxLength: 128,
                        }}
                        onChange={(e) => setPassword(e.target.value)}
                        icon={<Password />}
                        value={password}
                        autoComplete={"false"}
                      />
                    </FormControl>
                    <FormControl variant="standard" margin="normal" required fullWidth>
                      <OutlineIconTextField
                        variant={"outlined"}
                        label={t("login.repeatPassword")}
                        inputProps={{
                          name: "repeatPassword",
                          type: "password",
                          id: "repeatPassword",
                          required: "true",
                          minLength: 6,
                          maxLength: 128,
                        }}
                        onChange={(e) => setPasswordRepeat(e.target.value)}
                        icon={<Password />}
                        value={passwordRepeat}
                        autoComplete={"false"}
                      />
                    </FormControl>
                    {reg_captcha && (
                      <FormControl variant="standard" margin="normal" required fullWidth>
                        <Captcha
                          generation={captchaGen}
                          required={true}
                          fullWidth={true}
                          onStateChange={(s) => (captchaState.current = s)}
                        />
                      </FormControl>
                    )}
                    <LoadingButton
                      sx={{ mt: 2 }}
                      onClick={submit}
                      fullWidth
                      variant="contained"
                      color="primary"
                      loading={loading}
                    >
                      <span>{t("login.signUp")}</span>
                    </LoadingButton>
                    <Box sx={{ mt: 2, typography: "body2", textAlign: "center" }}>
                      <Trans
                        ns={"application"}
                        i18nKey={"login.haveAccountSignInNow"}
                        components={[<Link underline="hover" component={RouterLink} to="/session" />]}
                      />
                    </Box>
                    {showFooter && (
                      <>
                        <Divider sx={{ my: 2 }} />
                        <LegalLinks />
                      </>
                    )}
                  </Box>
                )}
              </Box>
            </CSSTransition>
          </SwitchTransition>
        </Box>
      </Box>
    </Box>
  );
};

export default SignUp;
