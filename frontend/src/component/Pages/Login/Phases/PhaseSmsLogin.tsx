import { Button, FormControl } from "@mui/material";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { sendSmsCode } from "../../../../api/api.ts";
import { useAppDispatch, useAppSelector } from "../../../../redux/hooks.ts";
import { Captcha, CaptchaParams } from "../../../Common/Captcha/Captcha.tsx";
import { OutlineIconTextField } from "../../../Common/Form/OutlineIconTextField.tsx";
import Password from "../../../Icons/Password.tsx";
import PhoneLaptopOutlined from "../../../Icons/PhoneLaptopOutlined.tsx";
import { Control } from "../Signin/SignIn.tsx";

export interface SmsLoginState {
  phone: string;
  code: string;
}

interface PhaseSmsLoginProps {
  setSmsState: (state: SmsLoginState) => void;
  control?: Control;
  captchaGen: number;
  setCaptchaState: (state: CaptchaParams) => void;
  captchaState: React.MutableRefObject<CaptchaParams | undefined>;
  scene?: "login" | "bind" | "reset";
}

const RESEND_SECONDS = 60;

const PhaseSmsLogin = ({
  setSmsState,
  control,
  captchaGen,
  setCaptchaState,
  captchaState,
  scene = "login",
}: PhaseSmsLoginProps) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const { login_captcha } = useAppSelector((state) => state.siteConfig.login.config);
  const [phone, setPhone] = useState("");
  const [code, setCode] = useState("");
  const [sending, setSending] = useState(false);
  const [countdown, setCountdown] = useState(0);

  useEffect(() => {
    if (countdown <= 0) {
      return;
    }
    const timer = setTimeout(() => setCountdown((c) => c - 1), 1000);
    return () => clearTimeout(timer);
  }, [countdown]);

  const onPhoneChange = (v: string) => {
    setPhone(v);
    setSmsState({ phone: v, code });
  };
  const onCodeChange = (v: string) => {
    setCode(v);
    setSmsState({ phone, code: v });
  };

  const onSend = () => {
    setSending(true);
    dispatch(sendSmsCode({ phone, scene, ...captchaState.current }))
      .then(() => setCountdown(RESEND_SECONDS))
      .finally(() => setSending(false));
  };

  return (
    <>
      <FormControl variant="standard" margin="normal" required fullWidth>
        <OutlineIconTextField
          label={t("login.phoneNumber")}
          variant="outlined"
          inputProps={{ id: "phone", type: "tel", name: "phone", required: "true" }}
          onChange={(e) => onPhoneChange(e.target.value)}
          icon={<PhoneLaptopOutlined />}
          autoComplete="tel"
          value={phone}
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
          inputProps={{ id: "sms-code", type: "text", name: "sms-code", required: "true" }}
          onChange={(e) => onCodeChange(e.target.value)}
          icon={<Password />}
          value={code}
        />
      </FormControl>
      <Button
        variant="text"
        size="small"
        disabled={sending || countdown > 0 || !phone || (login_captcha && !captchaState.current)}
        onClick={onSend}
        sx={{ mt: 1 }}
      >
        {countdown > 0 ? t("login.resendSmsCode", { seconds: countdown }) : t("login.sendSmsCode")}
      </Button>
      {control?.submit}
      {control?.back}
    </>
  );
};

export default PhaseSmsLogin;
