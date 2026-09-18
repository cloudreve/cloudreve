import { Box, IconButton, List, Typography, useMediaQuery, useTheme } from "@mui/material";
import dayjs from "dayjs";
import { useSnackbar } from "notistack";
import { useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import {
  sendDeletePasskey,
  sendFinishPasskeyRegistration,
  sendPreparePasskeyRegistration,
} from "../../../../api/api.ts";
import { Passkey, UserSettings } from "../../../../api/user.ts";
import { useAppDispatch } from "../../../../redux/hooks.ts";
import { confirmOperation } from "../../../../redux/thunks/dialog.ts";
import { bufferEncode, urlBase64BufferDecode } from "../../../../util";
import { DefaultCloseAction } from "../../../Common/Snackbar/snackbar.tsx";
import { NoWrapTypography, SecondaryLoadingButton } from "../../../Common/StyledComponents.tsx";
import TimeBadge from "../../../Common/TimeBadge.tsx";
import Dismiss from "../../../Icons/Dismiss.tsx";
import Fingerprint from "../../../Icons/Fingerprint.tsx";
import ShieldAdd from "../../../Icons/ShieldAdd.tsx";
import SettingListItem from "../SettingListItem.tsx";

export interface PasskeyProps {
  setting: UserSettings;
  onPasskeyAdded: (passkey: Passkey) => void;
  onPasskeyDeleted: (passkeyID: string) => void;
}

const PasskeyList = (props: PasskeyProps) => {
  const { t } = useTranslation();
  const { enqueueSnackbar } = useSnackbar();
  const dispatch = useAppDispatch();
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("md"));

  const [loading, setLoading] = useState(false);
  const [deleteLoading, setDeleteLoading] = useState(false);
  const addPasskey = async () => {
    if (!navigator.credentials || !window.PublicKeyCredential) {
      enqueueSnackbar({
        message: t("setting.browserNotSupported"),
        variant: "warning",
        action: DefaultCloseAction,
      });

      return;
    }

    setLoading(true);
    try {
      const opts = await dispatch(sendPreparePasskeyRegistration());
      const credential = await navigator.credentials.create({
        publicKey: {
          rp: opts.publicKey.rp,
          user: {
            id: urlBase64BufferDecode(opts.publicKey.user.id),
            name: opts.publicKey.user.name,
            displayName: opts.publicKey.user.displayName,
          },

          // url decode base64
          challenge: urlBase64BufferDecode(opts.publicKey.challenge),
          pubKeyCredParams: opts.publicKey.pubKeyCredParams,
          timeout: opts.publicKey.timeout,
          excludeCredentials: opts.publicKey.excludeCredentials?.map((c) => ({
            id: urlBase64BufferDecode(c.id),
            type: c.type,
          })),
          authenticatorSelection: opts.publicKey.authenticatorSelection,
        },
      });
      if (credential) {
        const c = credential as PublicKeyCredential;
        const newPasskey = await dispatch(
          sendFinishPasskeyRegistration({
            response: JSON.stringify({
              id: credential.id,
              type: credential.type,
              rawId: bufferEncode(c.rawId),
              response: {
                // @ts-ignore
                attestationObject: bufferEncode(c.response.attestationObject),
                clientDataJSON: bufferEncode(c.response.clientDataJSON),
              },
            }),
            name: t("setting.passkeyName"),
            ua: navigator.userAgent,
          }),
        );

        props.onPasskeyAdded(newPasskey);
      }
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  const deletePasskey = (passkeyID: string) => {
    dispatch(confirmOperation(t("setting.removedAuthenticatorConfirm"))).then(() => {
      setDeleteLoading(true);
      dispatch(sendDeletePasskey(passkeyID))
        .then(() => {
          props.onPasskeyDeleted(passkeyID);
        })
        .finally(() => {
          setDeleteLoading(false);
        });
    });
  };
  return (
    <Box>
      {!props.setting.passkeys?.length && (
        <Box
          onClick={addPasskey}
          sx={{
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
            borderRadius: (theme) => `${theme.shape.borderRadius}px`,
            border: (theme) =>
              `1px solid ${theme.palette.mode === "light" ? "rgba(0, 0, 0, 0.23)" : "rgba(255, 255, 255, 0.23)"}`,
            textAlign: "center",
            p: 1,
            py: 2,
          }}
        >
          <Fingerprint
            sx={{
              fontSize: 40,
              color: (theme) => theme.palette.text.secondary,
            }}
          />
          <Typography color={"textSecondary"} variant={"caption"}>
            {t("setting.noAuthenticator")}
          </Typography>
        </Box>
      )}
      <List disablePadding>
        {props.setting.passkeys?.map((passkey) => (
          <SettingListItem
            iconColor={theme.palette.action.active}
            icon={Fingerprint}
            secondaryAction={
              <IconButton onClick={() => deletePasskey(passkey.id)} disabled={deleteLoading}>
                <Dismiss fontSize={"small"} />
              </IconButton>
            }
            settingTitle={<NoWrapTypography variant={"inherit"}>{passkey.name}</NoWrapTypography>}
            settingDescription={
              <Box>
                <Trans
                  i18nKey={"setting.accountCreatedAt"}
                  ns={"application"}
                  components={[<TimeBadge datetime={passkey.created_at} variant={"inherit"} />]}
                />
                <Box
                  component={isMobile ? "div" : "span"}
                  sx={{
                    ml: isMobile ? 0 : 1,
                    color: (theme) =>
                      passkey.used_at && dayjs().diff(dayjs(passkey.used_at), "day") < 7
                        ? theme.palette.success.main
                        : theme.palette.text.secondary,
                  }}
                >
                  {passkey.used_at ? (
                    <Trans
                      i18nKey={"setting.usedAt"}
                      ns={"application"}
                      components={[<TimeBadge datetime={passkey.used_at} variant={"inherit"} />]}
                    />
                  ) : (
                    t("setting.neverUsed")
                  )}
                </Box>
              </Box>
            }
          />
        ))}
      </List>
      <SecondaryLoadingButton
        sx={{ mt: 1 }}
        variant={"contained"}
        loading={loading}
        onClick={addPasskey}
        startIcon={<ShieldAdd />}
      >
        <span>{t("setting.addNewAuthenticator")}</span>
      </SecondaryLoadingButton>
    </Box>
  );
};

export default PasskeyList;
