import { Box } from "@mui/material";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useAppSelector } from "../../../redux/hooks.ts";
import { SecondaryButton } from "../StyledComponents.tsx";
import { CaptchaParams } from "./Captcha.tsx";

const TCAPTCHA_SCRIPT = "https://turing.captcha.qcloud.com/TCaptcha.js";
const TCAPTCHA_SCRIPT_ID = "tcaptcha-script";

export interface TCaptchaProps {
  onStateChange: (state: CaptchaParams) => void;
  generation: number;
  fullWidth?: boolean;
  [x: string]: unknown;
}

declare global {
  interface Window {
    TencentCaptcha?: new (
      appId: string,
      callback: (res: { ret: number; ticket: string; randstr: string }) => void,
      options?: Record<string, unknown>,
    ) => { show: () => void; destroy?: () => void };
  }
}

// TCaptcha renders a verify button opening Tencent's captcha popup. On
// success the widget yields { ticket, randstr }, which the backend verifies
// via DescribeCaptchaResult.
const TCaptcha = ({ onStateChange, generation, fullWidth, ...rest }: TCaptchaProps) => {
  const { t } = useTranslation("common");
  const appId = useAppSelector((state) => state.siteConfig.basic.config.tcaptcha_app_id);
  const [ready, setReady] = useState(false);
  const [verified, setVerified] = useState(false);
  const captchaRef = useRef<{ show: () => void; destroy?: () => void } | null>(null);
  const onStateChangeRef = useRef(onStateChange);

  useEffect(() => {
    onStateChangeRef.current = onStateChange;
  }, [onStateChange]);

  useEffect(() => {
    if (!appId) {
      return;
    }

    let script = document.getElementById(TCAPTCHA_SCRIPT_ID) as HTMLScriptElement | null;
    const onLoad = () => setReady(true);
    if (!script) {
      script = document.createElement("script");
      script.id = TCAPTCHA_SCRIPT_ID;
      script.src = TCAPTCHA_SCRIPT;
      script.async = true;
      script.onload = onLoad;
      document.head.appendChild(script);
    } else if (window.TencentCaptcha) {
      setReady(true);
    } else {
      script.onload = onLoad;
    }
  }, [appId]);

  // Regeneration invalidates a previous solve.
  useEffect(() => {
    if (generation > 0) {
      setVerified(false);
      captchaRef.current = null;
    }
  }, [generation]);

  const openCaptcha = () => {
    if (!window.TencentCaptcha || !appId) {
      return;
    }

    captchaRef.current?.destroy?.();
    const instance = new window.TencentCaptcha(
      appId,
      (res) => {
        if (res.ret === 0) {
          setVerified(true);
          onStateChangeRef.current({ ticket: res.ticket, randstr: res.randstr });
        }
      },
      { needFeedBack: false },
    );
    captchaRef.current = instance;
    instance.show();
  };

  if (!appId) {
    return null;
  }

  return (
    <Box sx={{ textAlign: "center", ...(fullWidth && { width: "100%" }) }} {...rest}>
      <SecondaryButton
        variant="outlined"
        fullWidth={fullWidth}
        onClick={openCaptcha}
        disabled={!ready}
        color={verified ? "success" : "primary"}
      >
        {verified ? t("captcha.verified") : t("captcha.verify")}
      </SecondaryButton>
    </Box>
  );
};

export default TCaptcha;
