import { CaptchaType } from "../../../api/site.ts";
import { useAppSelector } from "../../../redux/hooks.ts";
import CapCaptcha from "./CapCaptcha.tsx";
import DefaultCaptcha from "./DefaultCaptcha.tsx";
import ReCaptchaV2 from "./ReCaptchaV2.tsx";
import TCaptcha from "./TCaptcha.tsx";
import TurnstileCaptcha from "./TurnstileCaptcha.tsx";

export interface CaptchaProps {
  onStateChange: (state: CaptchaParams) => void;
  generation: number;
  noLabel?: boolean;
  [x: string]: any;
}

export interface CaptchaParams {
  [x: string]: any;
}

export const Captcha = (props: CaptchaProps) => {
  const captchaType = useAppSelector((state) => state.siteConfig.basic.config.captcha_type);

  switch (captchaType) {
    case CaptchaType.RECAPTCHA:
      return <ReCaptchaV2 {...props} />;
    case CaptchaType.TCAPTCHA:
      return <TCaptcha {...props} />;
    case CaptchaType.TURNSTILE:
      return <TurnstileCaptcha {...props} />;
    case CaptchaType.CAP:
      return <CapCaptcha {...props} />;
    default:
      return <DefaultCaptcha {...props} />;
  }
};
