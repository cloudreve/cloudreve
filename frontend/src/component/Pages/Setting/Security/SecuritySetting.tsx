import { Icon } from "@iconify/react";
import { LoadingButton } from "@mui/lab";
import {
  Avatar,
  Box,
  Collapse,
  List,
  ListItem,
  ListItemAvatar,
  ListItemSecondaryAction,
  Stack,
  styled,
  Typography,
  useMediaQuery,
  useTheme,
} from "@mui/material";
import { useSnackbar } from "notistack";
import React, { useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import {
  sendRevokeOAuthGrant,
  sendUnbindSso,
  sendUpdateUserSetting,
  sendVaultDisable,
  sendVaultLock,
  sendVaultSetup,
} from "../../../../api/api.ts";
import { LinkedAccount, OAuthGrant, Passkey, UserSettings } from "../../../../api/user.ts";
import { useAppDispatch, useAppSelector } from "../../../../redux/hooks.ts";
import { confirmOperation } from "../../../../redux/thunks/dialog.ts";
import SessionManager from "../../../../session";
import { DefaultCloseAction } from "../../../Common/Snackbar/snackbar.tsx";
import {
  DenseFilledTextField,
  SecondaryButton,
  SquareChip,
  StyledListItemText,
} from "../../../Common/StyledComponents.tsx";
import TimeBadge from "../../../Common/TimeBadge.tsx";
import AppsListOutlined from "../../../Icons/AppsListOutlined.tsx";
import Dismiss from "../../../Icons/Dismiss.tsx";
import Edit from "../../../Icons/Edit.tsx";
import LockClosed from "../../../Icons/LockClosed.tsx";
import LockClosedKey from "../../../Icons/LockClosedKey.tsx";
import Open from "../../../Icons/Open.tsx";
import ShieldLock from "../../../Icons/ShieldLock.tsx";
import { ProfileSettingProps } from "../ProfileSetting.tsx";
import SettingForm from "../SettingForm.tsx";
import Backup2FACodesDialog from "./Backup2FACodesDialog.tsx";
import Disable2FADialog from "./Disable2FADialog.tsx";
import Enable2FADialog from "./Enable2FADialog.tsx";
import PasskeyList from "./PasskeyList.tsx";
import dayjs from "dayjs";

export interface SecuritySettingProps {
  setting: UserSettings;
  setSetting: (setting: UserSettings) => void;
}

const StyledOAuthGrantListItem = styled(ListItem)(({ theme }) => ({
  borderRadius: theme.shape.borderRadius,
  border: `1px solid ${theme.palette.mode === "light" ? "rgba(0, 0, 0, 0.23)" : "rgba(255, 255, 255, 0.23)"}`,
  marginTop: theme.spacing(1),
  paddingBottom: theme.spacing(0.5),
  paddingTop: theme.spacing(0.5),
}));

const isUrl = (str: string) =>
  str.startsWith("http://") || str.startsWith("https://") || str.startsWith("data:") || str.startsWith("/");

const OAuthGrantItem = ({ grant, onRevoked }: { grant: OAuthGrant; onRevoked: (clientId: string) => void }) => {
  const { t } = useTranslation();
  const theme = useTheme();
  const dispatch = useAppDispatch();
  const [loading, setLoading] = useState(false);

  const onRevoke = () => {
    dispatch(confirmOperation(t("setting.revokeOAuthGrantConfirm"))).then(() => {
      setLoading(true);
      dispatch(sendRevokeOAuthGrant(grant.client_id))
        .then(() => {
          onRevoked(grant.client_id);
        })
        .finally(() => {
          setLoading(false);
        });
    });
  };

  const renderIcon = () => {
    if (!grant.client_logo) {
      return (
        <Avatar sx={{ bgcolor: theme.palette.primary.main }}>
          <AppsListOutlined />
        </Avatar>
      );
    }

    if (isUrl(grant.client_logo)) {
      return (
        <Box
          component="img"
          src={grant.client_logo}
          sx={{
            width: 40,
            display: "block",
            height: 40,
            maxWidth: 40,
            maxHeight: 40,
            objectFit: "contain",
            borderRadius: 1,
            bgcolor: theme.palette.background.paper,
          }}
        />
      );
    }

    return (
      <Avatar sx={{ bgcolor: theme.palette.primary.main }}>
        <Icon icon={grant.client_logo} style={{ fontSize: 24 }} />
      </Avatar>
    );
  };

  return (
    <StyledOAuthGrantListItem sx={{ pr: "150px" }}>
      <ListItemAvatar>{renderIcon()}</ListItemAvatar>
      <StyledListItemText
        primary={t(grant.client_name)}
        secondaryTypographyProps={{
          variant: "caption",
          sx: {
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
          },
        }}
        secondary={
          <Box
            component="span"
            sx={{
              color:
                grant.last_used_at && dayjs().diff(dayjs(grant.last_used_at), "day") < 7
                  ? theme.palette.success.main
                  : theme.palette.text.secondary,
            }}
          >
            {grant.last_used_at ? (
              <Trans
                i18nKey={"setting.oauthGrantLastUsed"}
                ns={"application"}
                components={[<TimeBadge datetime={grant.last_used_at} variant={"inherit"} />]}
              />
            ) : (
              t("setting.neverUsed")
            )}
          </Box>
        }
      />
      <ListItemSecondaryAction>
        <LoadingButton
          loading={loading}
          variant={"outlined"}
          onClick={onRevoke}
          startIcon={<Dismiss />}
          color={"error"}
        >
          <span>{t("setting.revokeOAuthGrant")}</span>
        </LoadingButton>
      </ListItemSecondaryAction>
    </StyledOAuthGrantListItem>
  );
};

const linkedAccountProviderName = (t: (key: string) => string, provider: string): string => {
  switch (provider) {
    case "qq":
      return t("setting.providerQQ");
    default:
      return provider;
  }
};

const LinkedAccountItem = ({
  account,
  onUnlinked,
}: {
  account: LinkedAccount;
  onUnlinked: (provider: string) => void;
}) => {
  const { t } = useTranslation();
  const theme = useTheme();
  const dispatch = useAppDispatch();
  const [loading, setLoading] = useState(false);

  const onUnbind = () => {
    dispatch(confirmOperation(t("setting.unlinkAccountConfirm"))).then(() => {
      setLoading(true);
      dispatch(sendUnbindSso(account.provider))
        .then(() => {
          onUnlinked(account.provider);
        })
        .finally(() => {
          setLoading(false);
        });
    });
  };

  return (
    <StyledOAuthGrantListItem sx={{ pr: "150px" }}>
      <ListItemAvatar>
        <Avatar sx={{ bgcolor: theme.palette.primary.main }}>
          {account.provider === "qq" ? <Icon icon="ri:qq-fill" style={{ fontSize: 24 }} /> : <AppsListOutlined />}
        </Avatar>
      </ListItemAvatar>
      <StyledListItemText
        primary={linkedAccountProviderName(t, account.provider)}
        secondaryTypographyProps={{
          variant: "caption",
        }}
        secondary={
          <Trans
            i18nKey={"setting.linkedAt"}
            ns={"application"}
            components={[<TimeBadge key="0" datetime={account.created_at} variant={"inherit"} />]}
          />
        }
      />
      <ListItemSecondaryAction>
        <LoadingButton
          loading={loading}
          variant={"outlined"}
          onClick={onUnbind}
          startIcon={<Dismiss />}
          color={"error"}
        >
          <span>{t("setting.unlinkAccount")}</span>
        </LoadingButton>
      </ListItemSecondaryAction>
    </StyledOAuthGrantListItem>
  );
};

const VaultSetting = ({ setting, setSetting }: ProfileSettingProps) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const { enqueueSnackbar } = useSnackbar();
  const [password, setPassword] = useState("");
  const [repeat, setRepeat] = useState("");
  const [showDisable, setShowDisable] = useState(false);
  const [disablePassword, setDisablePassword] = useState("");
  const [loading, setLoading] = useState(false);

  const notify = (variant: "success" | "warning", message: string) =>
    enqueueSnackbar({ variant, message, action: DefaultCloseAction });

  const onSetup = () => {
    if (password !== repeat) {
      notify("warning", t("login.passwordNotMatch"));
      return;
    }
    setLoading(true);
    dispatch(sendVaultSetup(password))
      .then(() => {
        setSetting({ ...setting, vault_enabled: true });
        setPassword("");
        setRepeat("");
        notify("success", t("vault.enabled"));
      })
      .finally(() => setLoading(false));
  };

  const onLock = () => {
    setLoading(true);
    dispatch(sendVaultLock())
      .then(() => {
        setSetting({ ...setting, vault_unlocked: false });
        notify("success", t("vault.locked"));
      })
      .finally(() => setLoading(false));
  };

  const onDisable = () => {
    setLoading(true);
    dispatch(sendVaultDisable(disablePassword))
      .then(() => {
        setSetting({ ...setting, vault_enabled: false, vault_unlocked: false });
        setShowDisable(false);
        setDisablePassword("");
        notify("success", t("vault.disabled"));
      })
      .finally(() => setLoading(false));
  };

  return (
    <SettingForm
      title={
        <Box>
          {t("vault.title")}
          {setting.vault_enabled && (
            <SquareChip
              sx={{
                ml: 1,
                height: "initial",
                fontSize: (theme) => theme.typography.caption.fontSize,
              }}
              color={setting.vault_unlocked ? "success" : "default"}
              size={"small"}
              variant={"outlined"}
              label={t(`vault.${setting.vault_unlocked ? "unlocked" : "lockedState"}`)}
            />
          )}
        </Box>
      }
      lgWidth={5}
    >
      <Typography variant={"body2"} color={"text.secondary"}>
        {t("vault.description")}
      </Typography>
      {!setting.vault_enabled && (
        <Stack spacing={2} sx={{ mt: 2 }}>
          <DenseFilledTextField
            required
            label={t("vault.setPassword")}
            fullWidth
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            inputProps={{ type: "password", minLength: 6, maxLength: 128 }}
          />
          <DenseFilledTextField
            required
            label={t("login.repeatPassword")}
            fullWidth
            value={repeat}
            onChange={(e) => setRepeat(e.target.value)}
            inputProps={{ type: "password", minLength: 6, maxLength: 128 }}
          />
          <Box>
            <LoadingButton
              variant={"contained"}
              onClick={onSetup}
              loading={loading}
              disabled={password.length < 6}
              startIcon={<ShieldLock />}
            >
              <span>{t("vault.enable")}</span>
            </LoadingButton>
          </Box>
        </Stack>
      )}
      {setting.vault_enabled && (
        <Box sx={{ mt: 1 }}>
          <Stack direction={"row"} spacing={1}>
            {setting.vault_unlocked && (
              <SecondaryButton variant={"contained"} onClick={onLock} disabled={loading} startIcon={<LockClosed />}>
                {t("vault.lockNow")}
              </SecondaryButton>
            )}
            <SecondaryButton
              variant={"contained"}
              color={"error"}
              onClick={() => setShowDisable(!showDisable)}
              startIcon={<Dismiss />}
            >
              {t("vault.disable")}
            </SecondaryButton>
          </Stack>
          <Collapse in={showDisable} unmountOnExit>
            <Stack spacing={2} sx={{ mt: 2 }}>
              <Typography variant={"body2"} color={"text.secondary"}>
                {t("vault.disableDescription")}
              </Typography>
              <DenseFilledTextField
                required
                label={t("vault.confirmPassword")}
                fullWidth
                value={disablePassword}
                onChange={(e) => setDisablePassword(e.target.value)}
                inputProps={{ type: "password" }}
              />
              <Box>
                <LoadingButton
                  variant={"contained"}
                  color={"error"}
                  onClick={onDisable}
                  loading={loading}
                  disabled={disablePassword == ""}
                >
                  <span>{t("vault.confirmDisable")}</span>
                </LoadingButton>
              </Box>
            </Stack>
          </Collapse>
        </Box>
      )}
    </SettingForm>
  );
};

const SecuritySetting = ({ setting, setSetting }: ProfileSettingProps) => {
  const { t } = useTranslation();
  const theme = useTheme();
  const dispatch = useAppDispatch();
  const isMobile = useMediaQuery(theme.breakpoints.down("md"));
  const { enqueueSnackbar } = useSnackbar();
  const navigate = useNavigate();

  const authEnabled = useAppSelector((s) => s.siteConfig.login.config.authn);
  const qqConnectEnabled = useAppSelector((s) => s.siteConfig.login.config.qq_connect_enabled);

  const resetPwdFormRef = React.createRef<HTMLFormElement>();
  const [showResetPassword, setShowResetPassword] = useState(false);
  const [resetLoading, setResetLoading] = useState(false);
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [repeatPassword, setRepeatPassword] = useState("");
  const [enable2FAOpen, setEnable2FAOpen] = useState(false);
  const [disable2FAOpen, setDisable2FAOpen] = useState(false);
  const [backup2FAOpen, setBackup2FAOpen] = useState(false);

  const submitResetPassword = () => {
    if (!resetPwdFormRef.current) {
      return;
    }
    if (!resetPwdFormRef.current.checkValidity()) {
      resetPwdFormRef.current.reportValidity();
      return;
    }

    if (newPassword !== repeatPassword) {
      enqueueSnackbar({
        variant: "warning",
        message: t("login.passwordNotMatch"),
        action: DefaultCloseAction,
      });
      return;
    }

    setResetLoading(true);
    dispatch(
      sendUpdateUserSetting({
        current_password: currentPassword,
        new_password: newPassword,
      }),
    )
      .then(() => {
        SessionManager.signOutCurrent();
        navigate("/session");
        setShowResetPassword(false);
        enqueueSnackbar({
          variant: "success",
          message: t("login.passwordReset"),
          action: DefaultCloseAction,
        });
      })
      .finally(() => {
        setResetLoading(false);
      });
  };

  const on2FAChange = (enabled: boolean) => () => {
    setSetting({
      ...setting,
      two_fa_enabled: enabled,
    });
  };

  const onPasskeyAdded = (passkey: Passkey) => {
    setSetting({
      ...setting,
      passkeys: setting.passkeys ? [...setting.passkeys, passkey] : [passkey],
    });
  };

  const onPasskeyDeleted = (passkeyID: string) => {
    setSetting({
      ...setting,
      passkeys: setting.passkeys?.filter((p) => p.id !== passkeyID),
    });
  };

  const onOAuthGrantRevoked = (clientId: string) => {
    setSetting({
      ...setting,
      oauth_grants: setting.oauth_grants?.filter((x) => x.client_id != clientId),
    });
  };

  const onAccountUnlinked = (provider: string) => {
    setSetting({
      ...setting,
      linked_accounts: setting.linked_accounts?.filter((x) => x.provider != provider),
    });
  };

  return (
    <Stack spacing={3}>
      <SettingForm title={t("login.password")} lgWidth={5}>
        <Collapse in={!showResetPassword}>
          <SecondaryButton
            sx={{ mt: 1 }}
            onClick={() => setShowResetPassword(true)}
            variant={"contained"}
            startIcon={<Edit />}
          >
            {t("login.resetPassword")}
          </SecondaryButton>
        </Collapse>
        <Collapse in={showResetPassword} sx={{ mt: 2 }} unmountOnExit>
          <form ref={resetPwdFormRef}>
            <Stack spacing={2}>
              <DenseFilledTextField
                required
                label={t("setting.originalPassword")}
                fullWidth
                value={currentPassword}
                onChange={(e) => setCurrentPassword(e.target.value)}
                inputProps={{
                  type: "password",
                  minLength: 4,
                  maxLength: 64,
                }}
              />
              <DenseFilledTextField
                required
                label={t("login.newPassword")}
                fullWidth
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                inputProps={{
                  type: "password",
                  minLength: 6,
                  maxLength: 128,
                }}
              />
              <DenseFilledTextField
                required
                label={t("login.repeatPassword")}
                fullWidth
                value={repeatPassword}
                onChange={(e) => setRepeatPassword(e.target.value)}
                inputProps={{
                  type: "password",
                  minLength: 6,
                  maxLength: 128,
                }}
              />
              <Box>
                <LoadingButton variant={"contained"} onClick={submitResetPassword} loading={resetLoading}>
                  <span>{t("fileManager.save")}</span>
                </LoadingButton>
              </Box>
            </Stack>
          </form>
        </Collapse>
      </SettingForm>
      <SettingForm
        title={
          <Box>
            {t("setting.2fa")}
            {setting.two_fa_enabled && (
              <SquareChip
                sx={{
                  ml: 1,
                  height: "initial",
                  fontSize: (theme) => theme.typography.caption.fontSize,
                }}
                color={"success"}
                size={"small"}
                variant={"outlined"}
                label={t("setting.enabled")}
              />
            )}
          </Box>
        }
        lgWidth={5}
      >
        <SecondaryButton
          sx={{ mt: 1 }}
          onClick={() => (setting.two_fa_enabled ? setDisable2FAOpen(true) : setEnable2FAOpen(true))}
          variant={"contained"}
          startIcon={<Open />}
        >
          {t(`setting.${setting.two_fa_enabled ? "disable" : "enable"}2FA`)}
        </SecondaryButton>
        {setting.two_fa_enabled && (
          <SecondaryButton
            sx={{ mt: 1, ml: 1 }}
            variant={"outlined"}
            startIcon={<LockClosedKey />}
            onClick={() => setBackup2FAOpen(true)}
          >
            {t("setting.backup2FACodes")}
            {typeof setting.two_factor_backup_count === "number" && ` (${setting.two_factor_backup_count})`}
          </SecondaryButton>
        )}
      </SettingForm>
      {authEnabled && (
        <SettingForm title={t("setting.hardwareAuthenticator")}>
          <PasskeyList setting={setting} onPasskeyAdded={onPasskeyAdded} onPasskeyDeleted={onPasskeyDeleted} />
        </SettingForm>
      )}
      {setting.oauth_grants && setting.oauth_grants.length > 0 && (
        <SettingForm title={t("setting.externalApps")}>
          <List disablePadding>
            {setting.oauth_grants.map((grant) => (
              <OAuthGrantItem key={grant.client_id} grant={grant} onRevoked={onOAuthGrantRevoked} />
            ))}
          </List>
        </SettingForm>
      )}
      {(qqConnectEnabled || (setting.linked_accounts && setting.linked_accounts.length > 0)) && (
        <SettingForm title={t("setting.linkedAccounts")} lgWidth={5}>
          <List disablePadding>
            {setting.linked_accounts?.map((account) => (
              <LinkedAccountItem key={account.provider} account={account} onUnlinked={onAccountUnlinked} />
            ))}
          </List>
          {qqConnectEnabled && !setting.linked_accounts?.some((a) => a.provider === "qq") && (
            <SecondaryButton
              sx={{ mt: 1 }}
              variant={"contained"}
              startIcon={<Icon icon="ri:qq-fill" />}
              onClick={() => {
                window.location.href = "/api/v4/session/qq/login?link=1";
              }}
            >
              {t("setting.linkQQAccount")}
            </SecondaryButton>
          )}
        </SettingForm>
      )}
      <VaultSetting setting={setting} setSetting={setSetting} />
      <Enable2FADialog open={enable2FAOpen} onClose={() => setEnable2FAOpen(false)} on2FAEnabled={on2FAChange(true)} />
      <Disable2FADialog
        open={disable2FAOpen}
        onClose={() => setDisable2FAOpen(false)}
        on2FADisabled={on2FAChange(false)}
      />
      <Backup2FACodesDialog
        open={backup2FAOpen}
        onClose={() => setBackup2FAOpen(false)}
        onCodesRegenerated={(count) => setSetting({ ...setting, two_factor_backup_count: count })}
      />
    </Stack>
  );
};

export default SecuritySetting;
