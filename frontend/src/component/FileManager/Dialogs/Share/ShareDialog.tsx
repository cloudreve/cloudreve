import { Box, Checkbox, Collapse, DialogContent, IconButton, Stack, Tooltip, useTheme } from "@mui/material";
import dayjs from "dayjs";
import { TFunction } from "i18next";
import React, { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { CSSTransition, SwitchTransition } from "react-transition-group";
import { Share as ShareModel } from "../../../../api/explorer.ts";
import { closeShareLinkDialog } from "../../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../../redux/hooks.ts";
import { createOrUpdateShareLink } from "../../../../redux/thunks/share.ts";
import SessionManager from "../../../../session";
import { copyToClipboard, sendLink } from "../../../../util";
import AutoHeight from "../../../Common/AutoHeight.tsx";
import { FilledTextField, SmallFormControlLabel } from "../../../Common/StyledComponents.tsx";
import DraggableDialog from "../../../Dialogs/DraggableDialog.tsx";
import CopyOutlined from "../../../Icons/CopyOutlined.tsx";
import Share from "../../../Icons/Share.tsx";
import { FileManagerIndex } from "../../FileManager.tsx";
import ShareSettingContent, { downloadOptions, expireOptions, ShareSetting } from "./ShareSetting.tsx";

const shareSettingStorageKey = "cloudreve.share_setting";

const initialSetting = (privateByDefault: boolean): ShareSetting => ({
  is_private: privateByDefault || undefined,
  expires_val: expireOptions[2],
  downloads_val: downloadOptions[0],
});

// lastUsedSetting hydrates the dialog from the previously saved share
// settings (#2519). Password fields are never persisted.
const lastUsedSetting = (privateByDefault: boolean): ShareSetting => {
  try {
    const raw = localStorage.getItem(shareSettingStorageKey);
    if (raw) {
      return { ...initialSetting(privateByDefault), ...JSON.parse(raw) };
    }
  } catch {
    // corrupted or unavailable storage falls back to defaults
  }
  return initialSetting(privateByDefault);
};

const rememberSetting = (setting: ShareSetting) => {
  try {
    const { password, use_custom_password, ...rest } = setting;
    localStorage.setItem(shareSettingStorageKey, JSON.stringify(rest));
  } catch {
    // storage unavailable; remembering is best-effort
  }
};

interface ShareLinkPassword {
  shareLink: string;
  password?: string;
}

const shareToSetting = (share: ShareModel, t: TFunction): ShareSetting => {
  const res: ShareSetting = {
    is_private: share.is_private,
    password: share.password,
    use_custom_password: true,
    share_view: share.share_view,
    show_readme: share.show_readme,
    hide_readme: share.hide_readme,
    allow_upload: share.allow_upload,
    allow_edit: share.allow_edit,
    preview_only: share.preview_only,
    upload_only: share.upload_only,
    note: share.note,
    listed_publicly: share.listed_publicly,
    price_points: share.price && share.price > 0 ? share.price : undefined,
    downloads: share.remain_downloads != undefined && share.remain_downloads > 0,

    expires_val: expireOptions[2],
    downloads_val: downloadOptions[0],
  };

  if (res.downloads) {
    res.downloads_val = {
      value: share.remain_downloads ?? 0,
      label: (share.remain_downloads ?? 0).toString(),
    };
  }

  if (share.expires != undefined) {
    const expires = dayjs(share.expires);
    const isExpired = expires.isBefore(dayjs());
    if (!isExpired) {
      res.expires = true;
      const secondsTtl = dayjs(share.expires).diff(dayjs(), "second");
      res.expires_val = {
        value: secondsTtl,
        label: Math.round(secondsTtl / 60) + " " + t("application:modals.minutes"),
      };
    } else {
      res.expires = false;
    }
  }

  return res;
};

const ShareDialog = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const theme = useTheme();

  const sitePrivateDefault = useAppSelector((state) => state.siteConfig.basic.config.share_default_private);
  const userPrivateDefault = SessionManager.currentLoginOrNull()?.user.share_default_private;
  const privateByDefault = userPrivateDefault ?? !!sitePrivateDefault;

  const [loading, setLoading] = useState(false);
  const [setting, setSetting] = useState<ShareSetting>(() => initialSetting(privateByDefault));
  const [shareLink, setShareLink] = useState<string>("");
  const [includePassword, setIncludePassword] = useState(true);
  const shareLinkPassword = useMemo(() => {
    const start = shareLink.lastIndexOf("/s/");
    const shareLinkParts = shareLink.substring(start + 3).split("/");
    const password = shareLinkParts.length == 2 ? shareLinkParts[1] : undefined;
    return {
      shareLink: password ? shareLink.substring(0, shareLink.lastIndexOf("/")) : shareLink,
      password: password,
    } as ShareLinkPassword;
  }, [shareLink]);

  const open = useAppSelector((state) => state.globalState.shareLinkDialogOpen);
  const target = useAppSelector((state) => state.globalState.shareLinkDialogFile);
  const targets = useAppSelector((state) => state.globalState.shareLinkDialogFiles);
  const editTarget = useAppSelector((state) => state.globalState.shareLinkDialogShare);
  const multiCount = targets && targets.length > 1 ? targets.length : 0;

  useEffect(() => {
    if (open) {
      if (editTarget) {
        setSetting(shareToSetting(editTarget, t));
      } else {
        setSetting(lastUsedSetting(privateByDefault));
      }
      setShareLink("");
      setIncludePassword(true);
    }
  }, [open]);

  const onClose = useCallback(() => {
    if (!loading) {
      dispatch(closeShareLinkDialog());
    }
  }, [dispatch, loading]);

  const onAccept = useCallback(
    async (e?: React.MouseEvent<HTMLElement>) => {
      if (e) {
        e.preventDefault();
      }

      if (!target) return;

      if (shareLink) {
        copyToClipboard(shareLink);
        return;
      }

      setLoading(true);
      try {
        const shareLink = await dispatch(
          createOrUpdateShareLink(FileManagerIndex.main, target, setting, editTarget?.id, targets),
        );
        rememberSetting(setting);
        setShareLink(shareLink);
      } catch (e) {
      } finally {
        setLoading(false);
      }
    },
    [dispatch, target, shareLink, editTarget, setLoading, setting, setShareLink],
  );

  const finalShareLink = useMemo(() => {
    if (includePassword) {
      return shareLink;
    }
    return shareLink.substring(0, shareLink.lastIndexOf("/"));
  }, [includePassword, shareLink]);

  const finalShareLinkPassword = useMemo(() => {
    if (!includePassword) {
      return shareLink.substring(shareLink.lastIndexOf("/") + 1);
    }
    return undefined;
  }, [includePassword, shareLink]);

  return (
    <>
      <DraggableDialog
        title={t(`application:modals.${editTarget ? "edit" : "create"}ShareLink`)}
        showActions
        loading={loading}
        showCancel
        hideOk={!!shareLink}
        onAccept={onAccept}
        dialogProps={{
          open: open ?? false,
          onClose: onClose,
          fullWidth: true,
          maxWidth: "xs",
        }}
        cancelText={shareLink ? t("common:close") : undefined}
        secondaryAction={
          shareLink
            ? // @ts-ignore
              navigator.share && (
                <Tooltip title={t("application:modals.sendLink")}>
                  <IconButton onClick={() => sendLink(target?.name ?? "", finalShareLink)}>
                    <Share />
                  </IconButton>
                </Tooltip>
              )
            : undefined
        }
      >
        <DialogContent sx={{ pb: 0 }}>
          <AutoHeight>
            <SwitchTransition>
              <CSSTransition
                addEndListener={(node, done) => node.addEventListener("transitionend", done, false)}
                classNames="fade"
                key={`${shareLink}`}
              >
                <Box>
                  {!shareLink && (
                    <>
                      {multiCount > 0 && (
                        <FilledTextField
                          variant={"filled"}
                          inputProps={{ readOnly: true }}
                          label={t("application:modals.shareTargets")}
                          fullWidth
                          value={t("application:modals.shareTargetsCount", { count: multiCount })}
                          sx={{ mb: 1 }}
                        />
                      )}
                      <ShareSettingContent
                        editing={!!editTarget}
                        onSettingChange={setSetting}
                        setting={setting}
                        file={multiCount > 0 ? undefined : target}
                      />
                    </>
                  )}
                  {shareLink && (
                    <Stack spacing={1}>
                      <FilledTextField
                        variant={"filled"}
                        inputProps={{ readonly: true }}
                        label={t("modals.shareLink")}
                        fullWidth
                        value={finalShareLink ?? ""}
                        onFocus={(e) => e.target.select()}
                        slotProps={{
                          input: {
                            endAdornment: (
                              <IconButton
                                onClick={() => copyToClipboard(finalShareLink)}
                                size="small"
                                sx={{ marginRight: -1 }}
                              >
                                <CopyOutlined />
                              </IconButton>
                            ),
                          },
                        }}
                      />
                      {shareLinkPassword.password && (
                        <>
                          <Collapse in={!includePassword}>
                            <FilledTextField
                              variant={"filled"}
                              inputProps={{ readonly: true }}
                              label={t("modals.sharePassword")}
                              fullWidth
                              value={finalShareLinkPassword ?? ""}
                              onFocus={(e) => e.target.select()}
                              slotProps={{
                                input: {
                                  endAdornment: (
                                    <IconButton
                                      onClick={() => copyToClipboard(finalShareLinkPassword ?? "")}
                                      size="small"
                                      sx={{ marginRight: -1 }}
                                    >
                                      <CopyOutlined />
                                    </IconButton>
                                  ),
                                },
                              }}
                            />
                          </Collapse>
                          <Tooltip enterDelay={100} title={t("application:modals.includePasswordInShareLinkDes")}>
                            <SmallFormControlLabel
                              sx={{
                                mt: "0!important",
                              }}
                              control={
                                <Checkbox
                                  disableRipple
                                  sx={{
                                    pl: 0,
                                  }}
                                  size="small"
                                  checked={includePassword}
                                  onChange={() => {
                                    setIncludePassword(!includePassword);
                                  }}
                                />
                              }
                              label={t("application:modals.includePasswordInShareLink")}
                            />
                          </Tooltip>
                        </>
                      )}
                    </Stack>
                  )}
                </Box>
              </CSSTransition>
            </SwitchTransition>
          </AutoHeight>
        </DialogContent>
      </DraggableDialog>
    </>
  );
};
export default ShareDialog;
