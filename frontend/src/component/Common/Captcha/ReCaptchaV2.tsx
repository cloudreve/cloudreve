import { useEffect, useRef } from "react";
import { useAppSelector } from "../../../redux/hooks.ts";
import { CaptchaParams } from "./Captcha.tsx";
import ReCAPTCHA from "react-google-recaptcha";
import { Box, useTheme } from "@mui/material";

export interface ReCaptchaV2Props {
  onStateChange: (state: CaptchaParams) => void;
  generation: number;
}

declare global {
  interface Window {
    subTitle: string;
    recaptchaOptions: {
      useRecaptchaNet: boolean;
    };
  }
}

window.recaptchaOptions = {
  useRecaptchaNet: true,
};

const ReCaptchaV2 = ({ onStateChange, generation, ...rest }: ReCaptchaV2Props) => {
  const theme = useTheme();

  const captchaRef = useRef();
  const reCaptchaKey = useAppSelector((state) => state.siteConfig.basic.config.captcha_ReCaptchaKey);

  const refreshCaptcha = async () => {
    captchaRef.current?.reset();
  };

  useEffect(() => {
    refreshCaptcha();
  }, [generation]);

  const onCompleted = () => {
    const recaptchaValue = captchaRef.current?.getValue();
    if (recaptchaValue) {
      onStateChange({ captcha: recaptchaValue });
    }
  };

  return (
    <Box sx={{ textAlign: "center" }}>
      <ReCAPTCHA
        style={{ display: "inline-block" }}
        ref={captchaRef}
        sitekey={reCaptchaKey}
        onChange={onCompleted}
        theme={theme.palette.mode}
        {...rest}
      />
    </Box>
  );
};

export default ReCaptchaV2;
