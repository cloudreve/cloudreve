import { Box, InputAdornment, Skeleton, TextField } from "@mui/material";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { getCaptcha } from "../../../api/api.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { CaptchaParams } from "./Captcha.tsx";

export interface DefaultCaptchaProps {
  onStateChange: (state: CaptchaParams) => void;
  generation: number;
  noLabel?: boolean;
}

const DefaultCaptcha = ({ onStateChange, generation, noLabel, ...rest }: DefaultCaptchaProps) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();

  const [captcha, setCaptcha] = useState("");
  const [sessionId, setSessionId] = useState("");
  const [captchaData, setCaptchaData] = useState<string>();

  const isFirstMount = useRef(true);
  const inputRef = useRef<HTMLInputElement>(null);
  const [shouldFocus, setShouldFocus] = useState(false);

  const refreshCaptcha = async (resetInput = false) => {
    setCaptchaData(undefined);
    const captchaResponse = await dispatch(getCaptcha());
    setCaptchaData(captchaResponse.image);
    setSessionId(captchaResponse.ticket);

    if (resetInput) {
      setCaptcha("");
      setShouldFocus(true);
    }
  };

  useEffect(() => {
    refreshCaptcha(!isFirstMount.current);
    isFirstMount.current = false;
  }, [generation]);

  useEffect(() => {
    if (shouldFocus) {
      inputRef.current?.focus();
      setShouldFocus(false);
    }
  }, [shouldFocus]);

  useEffect(() => {
    onStateChange({ captcha, ticket: sessionId });
  }, [captcha, sessionId]);

  return (
    <TextField
      inputRef={inputRef}
      sx={{
        "& .MuiOutlinedInput-root": {
          pr: 0.5,
        },
      }}
      variant={"outlined"}
      label={noLabel ? undefined : t("login.captcha")}
      onChange={(e) => setCaptcha(e.target.value)}
      value={captcha}
      autoComplete={"true"}
      {...rest}
      slotProps={{
        input: {
          endAdornment: (
            <InputAdornment position={"end"}>
              <Box
                sx={{
                  cursor: "pointer",
                  height: 48,
                }}
                title={t("login.clickToRefresh")}
              >
                {!captchaData && (
                  <Skeleton
                    animation={"wave"}
                    sx={{
                      borderRadius: (theme) => `${theme.shape.borderRadius}px`,
                    }}
                    variant="rounded"
                    width={192}
                    height={48}
                  />
                )}
                {captchaData && (
                  <Box
                    component={"img"}
                    sx={{
                      borderRadius: (theme) => `${theme.shape.borderRadius}px`,
                      height: 48,
                    }}
                    src={captchaData}
                    alt="captcha"
                    onClick={() => refreshCaptcha(true)}
                  />
                )}
              </Box>
            </InputAdornment>
          ),
        },

        htmlInput: {
          name: "captcha",
          id: "captcha",
        },
      }}
    />
  );
};

export default DefaultCaptcha;
