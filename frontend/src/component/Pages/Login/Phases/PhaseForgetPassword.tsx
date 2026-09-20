import { useTranslation } from "react-i18next";
import { Control } from "../Signin/SignIn.tsx";
import { useAppDispatch, useAppSelector } from "../../../../redux/hooks.ts";
import { Box, Button, FormControl, Link } from "@mui/material";
import { LoadingButton } from "@mui/lab";
import { useEffect, useState } from "react";
import { enqueueSnackbar } from "notistack";
import { sendSmsCode, sendSmsReset } from "../../../../api/api.ts";
import { Captcha, CaptchaParams } from "../../../Common/Captcha/Captcha.tsx";
import { OutlineIconTextField } from "../../../Common/Form/OutlineIconTextField.tsx";
import LockClosedOutlined from "../../../Icons/LockClosedOutlined.tsx";
import Password from "../../../Icons/Password.tsx";
import PhoneLaptopOutlined from "../../../Icons/PhoneLaptopOutlined.tsx";
import { DefaultCloseAction } from "../../../Common/Snackbar/snackbar.tsx";

interface PhaseForgetPasswordProps {
  email: string;
  control?: Control;
  captchaGen: number;
  setCaptchaState: (state: CaptchaParams) => void;
  captchaState?: React.MutableRefObject<CaptchaParams | undefined>;
  onSmsResetDone?: () => void;
}

const RESEND_SECONDS = 60;

const PhaseForgetPassword = ({
  captchaGen,
  setCaptchaState,
  captchaState,
  control,
  onSmsResetDone,
}: PhaseForgetPasswordProps) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const { forget_captcha, sms_enabled, login_captcha } = useAppSelector((state) => state.siteConfig.login.config);
  const [smsMode, setSmsMode] = useState(false);
  const [phone, setPhone] = useState("");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [sending, setSending] = useState(false);
  const [resetting, setResetting] = useState(false);
  const [countdown, setCountdown] = useState(0);

  useEffect(() => {
    if (countdown <= 0) {
      return;
    }
    const timer = setTimeout(() => setCountdown((c) => c - 1), 1000);
    return () => clearTimeout(timer);
  }, [countdown]);

  const onSendCode = () => {
    setSending(true);
    dispatch(sendSmsCode({ phone, scene: "reset", ...captchaState?.current }))
      .then(() => setCountdown(RESEND_SECONDS))
      .finally(() => setSending(false));
  };

  const onSmsReset = () => {
    setResetting(true);
    dispatch(sendSmsReset({ phone, code, password }))
      .then(() => {
        enqueueSnackbar({
          message: t("login.passwordReset"),
          variant: "success",
          action: DefaultCloseAction,
        });
        onSmsResetDone?.();
      })
      .finally(() => setResetting(false));
  };

  if (smsMode) {
    return (
      <>
        <FormControl variant="standard" margin="normal" required fullWidth>
          <OutlineIconTextField
            label={t("login.phoneNumber")}
            variant="outlined"
            inputProps={{ type: "tel", name: "phone", required: "true" }}
            onChange={(e) => setPhone(e.target.value)}
            icon={<PhoneLaptopOutlined />}
            value={phone}
            autoComplete="tel"
            autoFocus
          />
        </FormControl>
        {login_captcha && (
          <FormControl variant="standard" margin="normal" required fullWidth>
            <Captcha generation={captchaGen} required={true} fullWidth={true} onStateChange={setCaptchaState} />
          </FormControl>
        )}
        <FormControl variant="standard" margin="normal" required fullWidth>
          <OutlineIconTextField
            label={t("login.smsCode")}
            variant="outlined"
            inputProps={{ type: "text", required: "true" }}
            onChange={(e) => setCode(e.target.value)}
            icon={<Password />}
            value={code}
          />
        </FormControl>
        <Button
          variant="text"
          size="small"
          disabled={sending || countdown > 0 || !phone || (login_captcha && !captchaState?.current)}
          onClick={onSendCode}
          sx={{ mt: 1 }}
        >
          {countdown > 0 ? t("login.resendSmsCode", { seconds: countdown }) : t("login.sendSmsCode")}
        </Button>
        <FormControl variant="standard" margin="normal" required fullWidth>
          <OutlineIconTextField
            label={t("login.newPassword")}
            variant="outlined"
            inputProps={{ type: "password", required: "true" }}
            onChange={(e) => setPassword(e.target.value)}
            icon={<LockClosedOutlined />}
            value={password}
            autoComplete="new-password"
          />
        </FormControl>
        <LoadingButton
          sx={{ mt: 2 }}
          fullWidth
          variant="contained"
          color="primary"
          loading={resetting}
          disabled={!phone || code.length !== 6 || password.length < 6}
          onClick={onSmsReset}
        >
          <span>{t("login.resetPassword")}</span>
        </LoadingButton>
        {control?.back}
      </>
    );
  }

  return (
    <>
      {forget_captcha && (
        <FormControl variant="standard" margin="normal" required fullWidth>
          <Captcha generation={captchaGen} required={true} fullWidth={true} onStateChange={setCaptchaState} />
        </FormControl>
      )}
      {control?.submit}
      {sms_enabled && (
        <Box sx={{ mt: 1, textAlign: "center", typography: "body2" }}>
          <Link component="button" type="button" underline="hover" onClick={() => setSmsMode(true)}>
            {t("login.resetViaSms")}
          </Link>
        </Box>
      )}
      {control?.back}
    </>
  );
};

export default PhaseForgetPassword;
