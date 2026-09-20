import { Box, Button, Divider, PopoverProps, Stack, styled, Typography } from "@mui/material";
import HoverPopover from "material-ui-popup-state/HoverPopover";
import { useCallback, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { Share } from "../../../api/explorer.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { saveShareToMyFiles } from "../../../redux/thunks/share.ts";
import SessionManager from "../../../session";
import TimeBadge from "../../Common/TimeBadge.tsx";
import UserBadge from "../../Common/User/UserBadge.tsx";
import Eye from "../../Icons/Eye.tsx";
import Save from "../../Icons/Save.tsx";
import Timer from "../../Icons/Timer.tsx";

interface ShareInfoPopoverProps extends PopoverProps {
  displayName: string;
  shareInfo: Share;
}

export const PropTypography = styled(Typography)({
  display: "flex",
  alignItems: "center",
  gap: 8,
});

export const separator = " • ";

export interface ShareExpiresProps {
  expires?: string;
  remain_downloads?: number;
}
export const ShareExpires = ({ expires, remain_downloads }: ShareExpiresProps) => {
  const { t } = useTranslation();
  return (
    <>
      {expires && (
        <span>
          <Trans
            i18nKey="application:share.expireAt"
            components={[<TimeBadge variant={"inherit"} color={"inherit"} datetime={expires} />]}
          />
        </span>
      )}
      {expires && remain_downloads != undefined && separator}
      {remain_downloads != undefined &&
        t("application:share.expireAfterDownloads", {
          downloads: remain_downloads,
        })}
    </>
  );
};

export const ShareStatistics = ({ shareInfo }: { shareInfo: Share }) => {
  const { t } = useTranslation();
  return (
    <>
      {t("application:share.statisticsViews", {
        views: shareInfo.visited,
      })}
      {shareInfo.downloaded &&
        separator +
          t("application:share.statisticsDownloads", {
            downloads: shareInfo.downloaded,
          })}
    </>
  );
};

const ShareInfoPopover = ({ displayName, shareInfo, ...rest }: ShareInfoPopoverProps) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const [saving, setSaving] = useState(false);
  const loggedIn = SessionManager.currentLoginOrNull() != null;
  const stopPropagation = useCallback((e: any) => e.stopPropagation(), []);

  const saveToMyFiles = useCallback(() => {
    setSaving(true);
    dispatch(saveShareToMyFiles(shareInfo, shareInfo.password, displayName)).finally(() => setSaving(false));
  }, [dispatch, shareInfo, displayName]);
  return (
    <HoverPopover
      onMouseDown={stopPropagation}
      onMouseUp={stopPropagation}
      onClick={stopPropagation}
      {...rest}
      anchorOrigin={{
        vertical: "bottom",
        horizontal: "center",
      }}
      transformOrigin={{
        vertical: "top",
        horizontal: "center",
      }}
    >
      <Box sx={{ maxWidth: "500px" }}>
        <Box sx={{ p: 1.5 }}>
          <Typography variant={"body2"}>{displayName}</Typography>
        </Box>
        <Stack spacing={1} sx={{ p: 1.5, pt: 0 }}>
          {(shareInfo.remain_downloads || shareInfo.expires) && (
            <PropTypography variant={"body2"} color={"text.secondary"}>
              <Timer />
              <ShareExpires expires={shareInfo.expires} remain_downloads={shareInfo.remain_downloads} />
            </PropTypography>
          )}
          {shareInfo.visited > 0 && (
            <PropTypography variant={"body2"} color={"text.secondary"}>
              <Eye />
              <ShareStatistics shareInfo={shareInfo} />
            </PropTypography>
          )}
        </Stack>
        <Divider />
        <Box
          sx={{
            px: 1.5,
            py: 1,
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
          }}
        >
          <UserBadge
            sx={{ width: 20, height: 20 }}
            textProps={{
              variant: "body2",
            }}
            user={shareInfo.owner}
          />
          {shareInfo.created_at && (
            <TimeBadge sx={{ ml: 1 }} variant={"body2"} color={"text.secondary"} datetime={shareInfo.created_at} />
          )}
        </Box>
        {loggedIn && (
          <>
            <Divider />
            <Box sx={{ px: 1.5, py: 1 }}>
              <Button
                size="small"
                variant="outlined"
                startIcon={<Save />}
                disabled={saving}
                onClick={saveToMyFiles}
              >
                {t("application:share.saveToMyFiles")}
              </Button>
            </Box>
          </>
        )}
      </Box>
    </HoverPopover>
  );
};

export default ShareInfoPopover;
