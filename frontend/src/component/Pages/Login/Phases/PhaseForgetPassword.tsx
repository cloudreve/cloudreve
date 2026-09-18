import { useTranslation } from "react-i18next";
import { Control } from "../Signin/SignIn.tsx";
import { useAppSelector } from "../../../../redux/hooks.ts";
import { FormControl } from "@mui/material";
import { Captcha, CaptchaParams } from "../../../Common/Captcha/Captcha.tsx";

interface PhaseForgetPasswordProps {
  email: string;
  control?: Control;
  captchaGen: number;
  setCaptchaState: (state: CaptchaParams) => void;
}

const PhaseForgetPassword = ({ captchaGen, setCaptchaState, control }: PhaseForgetPasswordProps) => {
  const { t } = useTranslation();
  const { forget_captcha } = useAppSelector((state) => state.siteConfig.login.config);

  return (
    <>
      {forget_captcha && (
        <FormControl variant="standard" margin="normal" required fullWidth>
          <Captcha generation={captchaGen} required={true} fullWidth={true} onStateChange={setCaptchaState} />
        </FormControl>
      )}
      {control?.submit}
      {control?.back}
    </>
  );
};

export default PhaseForgetPassword;
