import {
  Alert,
  Box,
  Button,
  ButtonGroup,
  Container,
  Divider,
  Link,
  Stack,
  styled,
  Typography,
  useMediaQuery,
  useTheme,
} from "@mui/material";
import { bindPopover, usePopupState } from "material-ui-popup-state/hooks";
import React, { forwardRef, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { Link as RouterLink } from "react-router-dom";
import { FileResponse, Share } from "../../../api/explorer.ts";
import { bindDelayedHover } from "../../../hooks/delayedHover.tsx";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { downloadSingleFile } from "../../../redux/thunks/download.ts";
import { createShareShortcut, openFileContextMenu } from "../../../redux/thunks/file.ts";
import { queueLoadShareInfo } from "../../../redux/thunks/share.ts";
import { openViewers } from "../../../redux/thunks/viewer.ts";
import SessionManager from "../../../session/index.ts";
import { sizeToString } from "../../../util/index.ts";
import CrUri from "../../../util/uri.ts";
import { SecondaryButton } from "../../Common/StyledComponents.tsx";
import UserAvatar from "../../Common/User/UserAvatar.tsx";
import CaretDown from "../../Icons/CaretDown.tsx";
import Download from "../../Icons/Download.tsx";
import Eye from "../../Icons/Eye.tsx";
import FolderLink from "../../Icons/FolderLink.tsx";
import Open from "../../Icons/Open.tsx";
import Timer from "../../Icons/Timer.tsx";
import useActionDisplayOpt from "../ContextMenu/useActionDisplayOpt.ts";
import { FmIndexContext } from "../FmIndexContext.tsx";
import { PropTypography, ShareExpires, ShareStatistics } from "../TopBar/ShareInfoPopover.tsx";
import FileIcon from "./FileIcon.tsx";
import FileTagSummary from "./FileTagSummary.tsx";
import { useFileBlockState } from "./GridView/GridFile.tsx";
import { ThumbPopover } from "./ListView/Cell.tsx";

const ShareContainer = styled(Box)(({ theme }) => ({
  border: `1px solid ${theme.palette.divider}`,
  borderRadius: theme.shape.borderRadius,
  padding: theme.spacing(2),
  width: "100%",
  backgroundColor: theme.palette.background.default,
  boxShadow: `0 0 10px 0 rgba(0, 0, 0, 0.1)`,
}));

const FileList = ({ file }: { file: FileResponse }) => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const isTouch = useMediaQuery("(pointer: coarse)");

  const { uploading, noThumb, fileTag, isSelected, thumbWidth, thumbHeight } = useFileBlockState({
    file,
  });

  const user = useMemo(() => {
    return SessionManager.currentLoginOrNull();
  }, []);

  const popupState = usePopupState({
    variant: "popover",
    popupId: "thumbPreview" + file.id,
  });

  const hoverState = bindDelayedHover(popupState, 800);
  const stopPropagation = useCallback((e: React.MouseEvent<HTMLElement>) => e.stopPropagation(), []);

  return (
    <>
      <Box
        {...(noThumb || isMobile || isTouch ? {} : hoverState)}
        sx={{ display: "flex", alignItems: "flex-start", my: 3 }}
      >
        <Box>
          <FileIcon
            variant={"shareSingle"}
            sx={{ py: 0 }}
            iconProps={{
              sx: {
                fontSize: "32px",
                height: "32px",
                width: "32px",
              },
            }}
            file={file}
          />
        </Box>
        <Box sx={{ flexGrow: 1 }}>
          <Typography variant={"body2"} sx={{ display: "flex", flexWrap: "wrap", columnGap: 1 }}>
            {file?.name}{" "}
            {fileTag && fileTag.length > 0 && (
              <FileTagSummary onMouseOver={stopPropagation} sx={{ maxWidth: "50%" }} tags={fileTag} />
            )}
          </Typography>
          <Typography variant={"body2"} color={"text.secondary"}>
            {sizeToString(file?.size ?? 0)}
          </Typography>
        </Box>
      </Box>
      {!noThumb && (
        <ThumbPopover
          key={file.id}
          file={file}
          thumbWidth={thumbWidth}
          thumbHeight={thumbHeight}
          popupState={bindPopover(popupState)}
        />
      )}
    </>
  );
};
const SingleFileView = forwardRef((_props, ref: React.Ref<any>) => {
  const { t } = useTranslation();
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const dispatch = useAppDispatch();
  const fmIndex = useContext(FmIndexContext);
  const file = useAppSelector((state) => state.fileManager[fmIndex].list?.files[0]);
  const [loading, setLoading] = useState(false);
  const [shareInfo, setShareInfo] = useState<Share | null>(null);

  const displayOpt = useActionDisplayOpt(file ? [file] : []);

  useEffect(() => {
    if (file) {
      dispatch(queueLoadShareInfo(new CrUri(file.path)))
        .then((info) => {
          setShareInfo(info);
        })
        .catch((_e) => {
          setShareInfo(null);
        })
        .finally(() => {
          setLoading(false);
        });
    } else {
      setShareInfo(null);
    }
  }, [file]);

  const openMore = useCallback(
    (e: React.MouseEvent<any>) => {
      if (file) {
        dispatch(openFileContextMenu(fmIndex, file, true, e));
      }
    },
    [dispatch, file],
  );
  const download = useCallback(async () => {
    if (!file) {
      return;
    }

    setLoading(true);
    try {
      await dispatch(downloadSingleFile(file));
    } finally {
      setLoading(false);
    }
  }, [file, dispatch]);

  const user = useMemo(() => {
    return SessionManager.currentLoginOrNull();
  }, []);

  return (
    <Stack
      ref={ref}
      spacing={2}
      sx={{
        p: 2,
        height: "100%",
        display: "flex",
        justifyContent: "center",
        alignItems: "center",
      }}
    >
      {shareInfo && (
        <Container maxWidth="sm">
          <ShareContainer>
            <Box sx={{ display: "flex", flexDirection: "column", alignItems: "center", py: 2, px: 1 }}>
              <UserAvatar enablePopover overwriteTextSize sx={{ width: 56, height: 56 }} user={shareInfo.owner} />
              <Typography variant={isMobile ? "subtitle1" : "h6"} sx={{ mt: 2, fontWeight: 600 }}>
                <Trans
                  i18nKey="application:share.sharedBy"
                  components={[
                    <Link
                      underline="hover"
                      color="inherit"
                      component={RouterLink}
                      to={`/profile/${shareInfo.owner.id}`}
                    >
                      {shareInfo.owner.nickname}
                    </Link>,
                  ]}
                  values={{ nick: shareInfo.owner.nickname, num: 1 }}
                />
              </Typography>
              <Box
                sx={{
                  display: "flex",
                  alignItems: "center",
                  flexWrap: "wrap",
                  mt: isMobile ? 1 : 0,
                  gap: 1,
                  columnGap: 2,
                }}
              >
                <PropTypography variant={"caption"} color={"text.secondary"}>
                  <Eye sx={{ fontSize: "20px!important" }} />
                  <ShareStatistics shareInfo={shareInfo} />
                </PropTypography>
              </Box>
            </Box>
            <Divider />
            {file && <FileList file={file} />}
            <Divider />
            <Divider />
            {(shareInfo.remain_downloads || shareInfo.expires) && (
              <Alert sx={{ mt: 2 }} severity="info" icon={<Timer />}>
                <ShareExpires expires={shareInfo.expires} remain_downloads={shareInfo.remain_downloads} />
              </Alert>
            )}
            <Box
              sx={{
                display: "flex",
                flexDirection: isMobile ? "column" : "row",
                justifyContent: isMobile ? "flex-start" : "space-between",
                alignItems: isMobile ? "flex-start" : "center",

                mt: 2,
                gap: 1,
              }}
            >
              <Box></Box>
              <Box sx={{ display: "flex", flexWrap: "wrap", gap: 1 }}>
                {!!user && file && (
                  <SecondaryButton
                    variant="contained"
                    onClick={() => dispatch(createShareShortcut(fmIndex))}
                    disabled={loading}
                    startIcon={<FolderLink />}
                  >
                    {t("application:fileManager.save")}
                  </SecondaryButton>
                )}
                {displayOpt.showOpen && file && (
                  <SecondaryButton
                    variant="contained"
                    onClick={() => dispatch(openViewers(0, file))}
                    disabled={loading}
                    startIcon={<Open />}
                  >
                    {t("application:fileManager.open")}
                  </SecondaryButton>
                )}
                <ButtonGroup disableElevation variant="contained">
                  <Button onClick={download} disabled={loading} startIcon={<Download />}>
                    {t("application:fileManager.download")}
                  </Button>
                  <Button size="small" onClick={openMore}>
                    <CaretDown sx={{ fontSize: "12px!important" }} />
                  </Button>
                </ButtonGroup>
              </Box>
            </Box>
          </ShareContainer>
        </Container>
      )}
    </Stack>
  );
});

export default SingleFileView;
