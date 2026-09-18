import { LoadingButton } from "@mui/lab";
import { ButtonProps } from "@mui/material";
import { useSnackbar } from "notistack";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { sendFinishPasskeyLogin, sendPreparePasskeyLogin } from "../../../../api/api.ts";
import { setHeadlessFrameLoading } from "../../../../redux/globalStateSlice.ts";
import { useAppDispatch } from "../../../../redux/hooks.ts";
import { refreshUserSession } from "../../../../redux/thunks/session.ts";
import { bufferEncode, urlBase64BufferDecode, useQuery } from "../../../../util";
import { DefaultCloseAction } from "../../../Common/Snackbar/snackbar.tsx";
import Fingerprint from "../../../Icons/Fingerprint.tsx";
import { LoginResponse } from "../../../../api/user.ts";

export interface PasskeyLoginButtonProps extends ButtonProps {
  autoComplete?: boolean;
  onLoginSuccess?: (response: LoginResponse) => void;
}

export default function PasskeyLoginButton({ autoComplete, onLoginSuccess, ...rest }: PasskeyLoginButtonProps) {
  const { t } = useTranslation();
  const { enqueueSnackbar } = useSnackbar();
  const dispatch = useAppDispatch();
  const query = useQuery();

  const [loading, setLoading] = useState(false);
  const abortRef = useRef(new AbortController());

  useEffect(() => {
    abortRef.current = new AbortController();
    if (
      autoComplete &&
      navigator.credentials &&
      window.PublicKeyCredential &&
      PublicKeyCredential.isConditionalMediationAvailable
    ) {
      PublicKeyCredential.isConditionalMediationAvailable().then((v) => {
        if (!abortRef.current.signal.aborted) startLogin(true)();
      });
    }
    return () => {
      abortRef.current?.abort();
    };
  }, []);

  const startLogin = (conditional: boolean) => async () => {
    if (!navigator.credentials || !window.PublicKeyCredential) {
      enqueueSnackbar({
        message: t("setting.browserNotSupported"),
        variant: "warning",
        action: DefaultCloseAction,
      });

      return;
    }

    if (!conditional) {
      abortRef.current.abort();
    }

    setLoading(!conditional);
    try {
      const opts = await dispatch(sendPreparePasskeyLogin());
      const credential = await navigator.credentials.get({
        publicKey: {
          // url decode base64
          challenge: urlBase64BufferDecode(opts.options.publicKey.challenge),
          timeout: opts.options.publicKey.timeout,
          rpId: opts.options.publicKey.rpId,
        },
        ...(conditional ? { mediation: "conditional", signal: abortRef.current.signal } : {}),
      });
      if (credential) {
        dispatch(setHeadlessFrameLoading(true));
        const c = credential as PublicKeyCredential;
        const response = await dispatch(
          sendFinishPasskeyLogin({
            session_id: opts.session_id,
            response: JSON.stringify({
              id: credential.id,
              type: credential.type,
              rawId: bufferEncode(c.rawId),
              response: {
                // @ts-ignore
                attestationObject: bufferEncode(c.response.attestationObject),
                clientDataJSON: bufferEncode(c.response.clientDataJSON),
                // @ts-ignore
                signature: bufferEncode(c.response.signature),
                // @ts-ignore
                userHandle: bufferEncode(c.response.userHandle),
                // @ts-ignore
                authenticatorData: bufferEncode(c.response.authenticatorData),
              },
            }),
          }),
        );
        if (onLoginSuccess) {
          onLoginSuccess(response);
        } else {
          dispatch(refreshUserSession(response, query.get("redirect")));
        }
      }
    } catch (e) {
      console.log(e);
    } finally {
      !conditional && setLoading(false);
      dispatch(setHeadlessFrameLoading(false));
    }
  };

  return (
    <LoadingButton
      onClick={startLogin(false)}
      loading={loading}
      variant={"outlined"}
      startIcon={<Fingerprint />}
      fullWidth
      {...rest}
    >
      {t("login.useFIDO2")}
    </LoadingButton>
  );
}
