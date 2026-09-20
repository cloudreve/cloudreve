import { Box, FormControl, FormControlLabel, Link, ListItemText, Stack, Switch, Typography } from "@mui/material";
import { useContext, useMemo } from "react";
import { Trans, useTranslation } from "react-i18next";
import { Link as RouterLink } from "react-router-dom";
import { isTrueVal } from "../../../../session/utils.ts";
import SizeInput from "../../../Common/SizeInput.tsx";
import { DenseFilledTextField, DenseSelect } from "../../../Common/StyledComponents.tsx";
import { SquareMenuItem } from "../../../FileManager/ContextMenu/ContextMenu.tsx";
import SettingForm from "../../../Pages/Setting/SettingForm.tsx";
import { Code } from "../../../Common/Code.tsx";
import GroupSelectionInput from "../../Common/GroupSelectionInput.tsx";
import SharesInput from "../../Common/SharesInput.tsx";
import { NoMarginHelperText, SettingSection, SettingSectionContent } from "../Settings.tsx";
import { SettingContext } from "../SettingWrapper.tsx";
import QQConnectSettings from "./QQConnectSettings.tsx";
import SmsGatewaySettings from "./SmsGatewaySettings.tsx";
import SSOSettings from "./SSOSettings.tsx";
import WeChatConnectSettings from "./WeChatConnectSettings.tsx";

const UserSession = () => {
  const { t } = useTranslation("dashboard");
  const { formRef, setSettings, values } = useContext(SettingContext);

  const defaultSymbolics = useMemo(() => {
    let result: number[] = [];
    try {
      result = JSON.parse(values?.default_symbolics ?? "[]");
    } catch (e) {
      console.error(e);
    }
    return result;
  }, [values?.default_symbolics]);

  return (
    <Box component={"form"} ref={formRef} onSubmit={(e) => e.preventDefault()}>
      <Stack spacing={5}>
        <SettingSection>
          <Typography variant="h6" gutterBottom>
            {t("settings.accountManagement")}
          </Typography>
          <SettingSectionContent>
            <SettingForm lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel
                  control={
                    <Switch
                      checked={isTrueVal(values.register_enabled)}
                      onChange={(e) =>
                        setSettings({
                          register_enabled: e.target.checked ? "1" : "0",
                        })
                      }
                    />
                  }
                  label={t("settings.allowNewRegistrations")}
                />
                <NoMarginHelperText>{t("settings.allowNewRegistrationsDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            {isTrueVal(values.register_enabled) && (
              <SettingForm lgWidth={5}>
                <FormControl fullWidth>
                  <FormControlLabel
                    control={
                      <Switch
                        checked={isTrueVal(values.invitation_code_required)}
                        onChange={(e) =>
                          setSettings({
                            invitation_code_required: e.target.checked ? "1" : "0",
                          })
                        }
                      />
                    }
                    label={t("settings.invitationCodeRequired")}
                  />
                  <NoMarginHelperText>{t("settings.invitationCodeRequiredDes")}</NoMarginHelperText>
                </FormControl>
              </SettingForm>
            )}
            <SettingForm lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel
                  control={
                    <Switch
                      checked={isTrueVal(values.email_active)}
                      onChange={(e) =>
                        setSettings({
                          email_active: e.target.checked ? "1" : "0",
                        })
                      }
                    />
                  }
                  label={t("settings.emailActivation")}
                />
                <NoMarginHelperText>
                  <Trans
                    i18nKey="settings.emailActivationDes"
                    ns={"dashboard"}
                    components={[<Link href={"/admin/settings?tab=email"} />]}
                  />
                </NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel
                  control={
                    <Switch
                      checked={isTrueVal(values.authn_enabled)}
                      onChange={(e) =>
                        setSettings({
                          authn_enabled: e.target.checked ? "1" : "0",
                        })
                      }
                    />
                  }
                  label={t("settings.webauthn")}
                />
                <NoMarginHelperText>{t("settings.webauthnDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel
                  control={
                    <Switch
                      checked={isTrueVal(values.expose_user_email)}
                      onChange={(e) =>
                        setSettings({
                          expose_user_email: e.target.checked ? "1" : "0",
                        })
                      }
                    />
                  }
                  label={t("settings.exposeUserEmail")}
                />
                <NoMarginHelperText>{t("settings.exposeUserEmailDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("settings.defaultGroup")} lgWidth={5}>
              <FormControl>
                <GroupSelectionInput
                  value={values.default_group}
                  onChange={(g) =>
                    setSettings({
                      default_group: g,
                    })
                  }
                />
                <NoMarginHelperText>{t("settings.defaultGroupDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("settings.defaultSymbolics")} lgWidth={5}>
              <FormControl>
                <SharesInput
                  value={defaultSymbolics}
                  onChange={(ids) =>
                    setSettings({
                      default_symbolics: JSON.stringify(ids),
                    })
                  }
                />
                <NoMarginHelperText>
                  <Trans
                    i18nKey="settings.defaultSymbolicsDes"
                    ns={"dashboard"}
                    components={[<Link component={RouterLink} to={"/admin/share"} />]}
                  />
                </NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel
                  control={
                    <Switch
                      checked={isTrueVal(values.share_default_private)}
                      onChange={(e) =>
                        setSettings({
                          share_default_private: e.target.checked ? "1" : "0",
                        })
                      }
                    />
                  }
                  label={t("settings.shareDefaultPrivate")}
                />
                <NoMarginHelperText>{t("settings.shareDefaultPrivateDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("settings.defaultShareLinksInProfile")} lgWidth={5}>
              <FormControl>
                <DenseSelect
                  value={values.default_share_links_in_profile ?? ""}
                  onChange={(e) =>
                    setSettings({
                      default_share_links_in_profile: e.target.value as string,
                    })
                  }
                >
                  {["profileSharePublicOnly", "profileShareAll", "profileShareHide"].map((v, i) => (
                    <SquareMenuItem key={v} value={["", "all_share", "hide_share"][i]}>
                      <ListItemText
                        slotProps={{
                          primary: { variant: "body2" },
                        }}
                      >
                        {t(`settings.${v}`)}
                      </ListItemText>
                    </SquareMenuItem>
                  ))}
                </DenseSelect>
                <NoMarginHelperText>{t("settings.defaultShareLinksInProfileDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("vas.filterEmailProvider")} lgWidth={5}>
              <FormControl>
                <DenseSelect
                  value={values.email_filter_mode ?? "0"}
                  onChange={(e) =>
                    setSettings({
                      email_filter_mode: e.target.value as string,
                    })
                  }
                >
                  {["filterEmailProviderDisabled", "filterEmailProviderWhitelist", "filterEmailProviderBlacklist"].map(
                    (v, i) => (
                      <SquareMenuItem key={v} value={i.toString()}>
                        <ListItemText
                          slotProps={{
                            primary: { variant: "body2" },
                          }}
                        >
                          {t(`vas.${v}`)}
                        </ListItemText>
                      </SquareMenuItem>
                    ),
                  )}
                </DenseSelect>
                <NoMarginHelperText>{t("vas.filterEmailProviderDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            {(values.email_filter_mode ?? "0") !== "0" && (
              <SettingForm title={t("vas.filterEmailProviderRule")} lgWidth={5}>
                <FormControl fullWidth>
                  <DenseFilledTextField
                    value={values.email_filter_list}
                    onChange={(e) =>
                      setSettings({
                        email_filter_list: e.target.value,
                      })
                    }
                    multiline
                    minRows={3}
                    placeholder={"example.com\nmail.example.org"}
                  />
                  <NoMarginHelperText>{t("vas.filterEmailProviderRuleDes")}</NoMarginHelperText>
                </FormControl>
              </SettingForm>
            )}
            <SettingForm lgWidth={5}>
              <FormControl fullWidth>
                <FormControlLabel
                  control={
                    <Switch
                      checked={isTrueVal(values.email_disable_subaddress)}
                      onChange={(e) =>
                        setSettings({
                          email_disable_subaddress: e.target.checked ? "1" : "0",
                        })
                      }
                    />
                  }
                  label={
                    <>
                      {t("vas.disableSubAddressEmail")}
                    </>
                  }
                />
                <NoMarginHelperText>
                  <Trans i18nKey="vas.disableSubAddressEmailDes" ns={"dashboard"} components={[<Code />]} />
                </NoMarginHelperText>
              </FormControl>
            </SettingForm>
            {isTrueVal(values.email_disable_subaddress) && (
              <SettingForm title={t("vas.subAddressChars")} lgWidth={5}>
                <FormControl fullWidth>
                  <DenseFilledTextField
                    fullWidth
                    placeholder="+"
                    value={values.email_subaddress_chars ?? ""}
                    onChange={(e) => setSettings({ email_subaddress_chars: e.target.value })}
                  />
                  <NoMarginHelperText>
                    <Trans i18nKey="vas.subAddressCharsDes" ns={"dashboard"} components={[<Code />]} />
                  </NoMarginHelperText>
                </FormControl>
              </SettingForm>
            )}
          </SettingSectionContent>
        </SettingSection>
        <SettingSection>
          <Typography variant="h6" gutterBottom sx={{ display: "flex", alignItems: "center" }}>
            {t("settings.thirdPartySignIn")}
          </Typography>
          <SettingSectionContent>
            <SettingForm lgWidth={5}>
              <SSOSettings />
            </SettingForm>
            <SettingForm lgWidth={5}>
              <QQConnectSettings />
            </SettingForm>
            <SettingForm lgWidth={5}>
              <WeChatConnectSettings />
            </SettingForm>
            <SettingForm lgWidth={5}>
              <SmsGatewaySettings />
            </SettingForm>
          </SettingSectionContent>
        </SettingSection>
        <SettingSection>
          <Typography variant="h6" gutterBottom>
            {t("settings.avatar")}
          </Typography>
          <SettingSectionContent>
            <SettingForm title={t("settings.avatarFilePath")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  value={values.avatar_path}
                  onChange={(e) =>
                    setSettings({
                      avatar_path: e.target.value,
                    })
                  }
                  required
                />
                <NoMarginHelperText>{t("settings.avatarFilePathDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("settings.avatarSize")} lgWidth={5}>
              <FormControl>
                <SizeInput
                  variant={"outlined"}
                  required
                  label={t("application:navbar.minimum")}
                  value={parseInt(values.avatar_size) ?? 0}
                  onChange={(e) =>
                    setSettings({
                      avatar_size: e.toString(),
                    })
                  }
                />
                <NoMarginHelperText>{t("settings.avatarSizeDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("settings.avatarImageSize")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  value={values.avatar_size_l}
                  onChange={(e) =>
                    setSettings({
                      avatar_size_l: e.target.value,
                    })
                  }
                  type={"number"}
                  inputProps={{ step: 1, min: 1 }}
                  required
                />
                <NoMarginHelperText>{t("settings.avatarImageSizeDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
            <SettingForm title={t("settings.gravatarServer")} lgWidth={5}>
              <FormControl fullWidth>
                <DenseFilledTextField
                  value={values.gravatar_server}
                  onChange={(e) =>
                    setSettings({
                      gravatar_server: e.target.value,
                    })
                  }
                  required
                />
                <NoMarginHelperText>{t("settings.gravatarServerDes")}</NoMarginHelperText>
              </FormControl>
            </SettingForm>
          </SettingSectionContent>
        </SettingSection>
      </Stack>
    </Box>
  );
};

export default UserSession;
