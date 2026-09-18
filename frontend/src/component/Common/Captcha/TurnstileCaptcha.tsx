import { Turnstile } from "@marsidev/react-turnstile";
import { Box, useTheme } from "@mui/material";
import i18next from "i18next";
import { useEffect, useRef } from "react";
import { useAppSelector } from "../../../redux/hooks.ts";
import { CaptchaParams } from "./Captcha.tsx";

export interface TurnstileProps {
  onStateChange: (state: CaptchaParams) => void;
  generation: number;
}

const TurnstileCaptcha = ({ onStateChange, generation, ...rest }: TurnstileProps) => {
  const theme = useTheme();

  const captchaRef = useRef();
  const turnstileKey = useAppSelector((state) => state.siteConfig.basic.config.turnstile_site_id);

  const refreshCaptcha = async () => {
    captchaRef.current?.reset();
  };

  useEffect(() => {
    refreshCaptcha();
  }, [generation]);

  const onCompleted = (t: string) => {
    onStateChange({ ticket: t });
  };

  return (
    <Box sx={{ textAlign: "center" }}>
      {turnstileKey && (
        <Turnstile
          ref={captchaRef}
          siteKey={turnstileKey}
          options={{
            size: "flexible",
            theme: theme.palette.mode,
            language: i18next.language,
          }}
          onSuccess={onCompleted}
          {...rest}
        />
      )}
    </Box>
  );
};

export default TurnstileCaptcha;
