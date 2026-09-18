import {
  Box,
  IconButton,
  ListItemIcon,
  ListItemText,
  Menu,
  Tooltip,
  Typography,
  useMediaQuery,
  useTheme,
} from "@mui/material";
import Artplayer from "artplayer";
import dayjs from "dayjs";
import React, { lazy, Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { getFileEntityUrl } from "../../../api/api.ts";
import { FileResponse } from "../../../api/explorer.ts";
import { closeVideoViewer } from "../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { findSubtitleOptions } from "../../../redux/thunks/viewer.ts";
import SessionManager, { UserSettings } from "../../../session";
import { fileExtension, fileNameNoExt, getFileLinkedUri } from "../../../util";
import CrUri from "../../../util/uri.ts";
import { DenseDivider, SquareMenuItem } from "../../FileManager/ContextMenu/ContextMenu.tsx";
import Checkmark from "../../Icons/Checkmark.tsx";
import Subtitles from "../../Icons/Subtitles.tsx";
import TextEditStyle from "../../Icons/TextEditStyle.tsx";
import ViewerDialog, { ViewerLoading } from "../ViewerDialog.tsx";
import SubtitleStyleDialog from "./SubtitleStyleDialog.tsx";

const Player = lazy(() => import("./Artplayer.tsx"));

export const CrMaskedPrefix = "https://cloudreve_masked/";

export interface SubtitleStyle {
  fontSize?: number;
  fontColor?: string;
}

const srcRefreshMargin = 5 * 1000;
const VideoViewer = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const viewerState = useAppSelector((state) => state.globalState.videoViewer);

  const [loaded, setLoaded] = useState(false);
  const [art, setArt] = useState<Artplayer | null>(null);
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const currentExpire = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const [subtitles, setSubtitles] = useState<FileResponse[]>([]);
  const [subtitleSelected, setSubtitleSelected] = useState<FileResponse | null>(null);
  const [subtitleStyleOpen, setSubtitleStyleOpen] = useState(false);
  const currentUrl = useRef<string | null>(null);

  const subtitleStyle = useMemo(() => {
    return SessionManager.getWithFallback(UserSettings.SubtitleStyle) as SubtitleStyle;
  }, []);

  const switchSubtitle = useCallback(
    async (subtitle?: FileResponse) => {
      if (!art) {
        return;
      }

      setAnchorEl(null);

      if (!subtitle) {
        setSubtitleSelected(null);
        art.subtitle.show = false;
        return;
      }

      setSubtitleSelected(subtitle);
      try {
        const subtitleUrl = await dispatch(
          getFileEntityUrl({
            uris: [getFileLinkedUri(subtitle)],
          }),
        );

        art.subtitle.switch(subtitleUrl.urls[0].url, {
          type: fileExtension(subtitle.name) ?? "",
        });
        art.subtitle.show = true;
      } catch (e) {
        console.error(e);
        return;
      }
    },
    [art],
  );

  const loadSubtitles = useCallback(() => {
    if (!viewerState?.file) {
      return;
    }

    const subs = dispatch(findSubtitleOptions());
    setSubtitles(subs);
    if (subs.length > 0 && subs[0].name.startsWith(fileNameNoExt(viewerState.file.name) + ".")) {
      switchSubtitle(subs[0]);
    }
  }, [viewerState?.file, switchSubtitle]);

  // refresh video src before entity url expires
  const refreshSrc = useCallback(() => {
    if (!viewerState || !viewerState.file || !art) {
      return;
    }

    const firstLoad = !currentExpire.current;
    const isM3u8 = fileExtension(viewerState.file.name) === "m3u8";
    if (isM3u8) {
      // For m3u8, use masked url
      const crFileUrl = new CrUri(getFileLinkedUri(viewerState.file));
      const maskedUrl = `${CrMaskedPrefix}${crFileUrl.path()}`;
      art.switchUrl(maskedUrl);
      loadSubtitles();
      return;
    }

    dispatch(
      getFileEntityUrl({
        uris: [getFileLinkedUri(viewerState.file)],
        entity: viewerState.version,
      }),
    )
      .then((res) => {
        const current = art.currentTime;
        currentUrl.current = res.urls[0].url;

        let timeOut = dayjs(res.expires).diff(dayjs(), "millisecond") - srcRefreshMargin;
        if (timeOut < 0) {
          timeOut = 2000;
        }
        currentExpire.current = setTimeout(refreshSrc, timeOut);

        art.switchUrl(res.urls[0].url).then(() => {
          art.currentTime = current;
        });

        if (firstLoad) {
          const subs = dispatch(findSubtitleOptions());
          setSubtitles(subs);
          if (subs.length > 0 && subs[0].name.startsWith(fileNameNoExt(viewerState.file.name) + ".")) {
            switchSubtitle(subs[0]);
          }
        }
      })
      .catch((e) => {
        console.error(e);
        onClose();
      });
  }, [viewerState?.file, art, loadSubtitles]);

  const chapters = useMemo(() => {
    if (!viewerState || !viewerState.file?.metadata) {
      return undefined;
    }

    const chapterMap: {
      [key: string]: {
        start: number;
        end: number;
        title: string;
      };
    } = {};

    Object.keys(viewerState.file.metadata).map((k) => {
      if (k.startsWith("stream:chapter_")) {
        const id = k.split("_")[1];
        // type = remove prefix
        const type = k.replace(`stream:chapter_${id}_`, "");
        if (!chapterMap[id]) {
          chapterMap[id] = {
            start: 0,
            end: 0,
            title: "",
          };
        }
        switch (type) {
          case "start_time":
            chapterMap[id].start = parseFloat(viewerState.file?.metadata?.[k] ?? "0");
            break;
          case "end_time":
            chapterMap[id].end = parseFloat(viewerState.file?.metadata?.[k] ?? "0");
            break;
          case "name":
            chapterMap[id].title = viewerState.file?.metadata?.[k] ?? "";
            break;
        }
      }
    });

    return Object.values(chapterMap).map((c) => ({
      start: c.start,
      end: c.end,
      title: c.title,
    }));
  }, [viewerState]);

  useEffect(() => {
    if (!art) {
      return;
    }
    art.on("ready", () => {
      art.autoHeight();
      art.autoSize();
    });
    art.query(".art-video").addEventListener(
      "leavepictureinpicture",
      () => {
        art.pause();
      },
      false,
    );
    refreshSrc();
  }, [art]);

  useEffect(() => {
    if (!viewerState || !viewerState.open) {
      if (currentExpire.current) {
        clearTimeout(currentExpire.current);
        currentExpire.current = undefined;
      }
      return;
    }

    setArt(null);
    setSubtitles([]);
    setSubtitleSelected(null);
  }, [viewerState?.open]);

  const onClose = useCallback(() => {
    dispatch(closeVideoViewer());
  }, [dispatch]);

  const openOption = useCallback(
    (e: React.MouseEvent<any>) => {
      setAnchorEl(e.currentTarget);
    },
    [dispatch],
  );

  const openSubtitleStyle = useCallback(() => {
    setSubtitleStyleOpen(true);
    setAnchorEl(null);
  }, []);

  const applySubtitleStyle = useCallback(
    (style: SubtitleStyle) => {
      SessionManager.set(UserSettings.SubtitleStyle, style);
      setSubtitleStyleOpen(false);
      if (art) {
        art.subtitle.style({
          color: style.fontColor ?? "#fff",
          fontSize: `${style.fontSize ?? 20}px`,
        });
      }
    },
    [art],
  );

  const m3u8UrlTransform = useCallback(
    async (url: string, isPlaylist?: boolean): Promise<string> => {
      let realUrl = "";
      if (isPlaylist) {
        // Loading playlist
        if (!currentUrl.current) {
          return url;
        }

        const currentParsed = new URL(currentUrl.current);
        const requestParsed = new URL(url);
        if (currentParsed.origin != requestParsed.origin) {
          // Playlist is from different origin, return original URL
          return url;
        }

        // Trim pfrefix(currentParsed.pathname) of requestParsed.pathname to get relative path
        const currentPathParts = currentParsed.pathname.split("/");
        const requestPathParts = requestParsed.pathname.split("/");

        // Find where paths diverge
        let i = 0;
        while (
          i < currentPathParts.length &&
          i < requestPathParts.length &&
          currentPathParts[i] === requestPathParts[i]
        ) {
          i++;
        }

        // Get relative path by joining remaining parts
        const relativePath = requestPathParts.slice(i).join("/");

        if (!viewerState?.file) {
          return url;
        }

        const currentFileUrl = new CrUri(getFileLinkedUri(viewerState?.file));
        const base = i == 0 ? new CrUri(currentFileUrl.base()) : currentFileUrl.parent();
        realUrl = base.join(relativePath).path();
        return `${CrMaskedPrefix}${realUrl}`;
      } else {
        // Loading fragment
        if (url.startsWith("http://") || url.startsWith("https://") || !viewerState?.file) {
          // If fragment URL is not a path, return it
          return url;
        }

        // Request real fragment/playlist URL
        const currentFileUrl = new CrUri(getFileLinkedUri(viewerState?.file));
        const base = url.startsWith("/") ? new CrUri(currentFileUrl.base()) : currentFileUrl.parent();
        realUrl = base.join(url).path();
        return `${CrMaskedPrefix}${realUrl}`;
      }
    },
    [viewerState?.file],
  );

  const getUnmaskedEntityUrl = useCallback(
    async (url: string) => {
      if (!viewerState?.file) {
        return url;
      }
      // remove cloudreve_masked prefix of url
      if (!url.startsWith(CrMaskedPrefix)) {
        return url;
      }
      url = url.replace(CrMaskedPrefix, "");
      const currentFileUrl = new CrUri(getFileLinkedUri(viewerState.file));
      const base = new CrUri(currentFileUrl.base());
      const realUrl = base.join(...url.split("/"));
      try {
        const res = await dispatch(getFileEntityUrl({ uris: [realUrl.toString()] }));
        return res.urls[0].url;
      } catch (e) {
        console.error(e);
        return url;
      }
    },
    [dispatch, viewerState?.file],
  );

  // TODO: Add artplayer-plugin-chapter after it's released to npm
  return (
    <ViewerDialog
      file={viewerState?.file}
      actions={
        <Box sx={{ display: "flex", gap: 1 }}>
          <Tooltip title={t("fileManager.subtitles")}>
            <IconButton onClick={openOption}>
              <Subtitles fontSize={"small"} />
            </IconButton>
          </Tooltip>
        </Box>
      }
      fullScreenToggle
      toggleFullScreen={() => art && (art.fullscreenWeb = true)}
      dialogProps={{
        open: !!(viewerState && viewerState.open),
        onClose: onClose,
        fullWidth: true,
        maxWidth: "md",
      }}
    >
      <SubtitleStyleDialog
        onSaveSubmit={applySubtitleStyle}
        open={subtitleStyleOpen}
        onClose={() => setSubtitleStyleOpen(false)}
      />
      <Menu
        anchorEl={anchorEl}
        open={Boolean(anchorEl)}
        onClose={() => setAnchorEl(null)}
        slotProps={{
          paper: {
            sx: {
              minWidth: 150,
              maxWidth: 250,
            },
          },
        }}
      >
        <SquareMenuItem dense onClick={() => openSubtitleStyle()}>
          <ListItemIcon>
            <TextEditStyle fontSize={"small"} />{" "}
          </ListItemIcon>
          <ListItemText>{t("application:fileManager.subtitleStyles")}</ListItemText>
        </SquareMenuItem>
        <DenseDivider />
        {subtitles.length == 0 && (
          <Box sx={{ p: 1 }}>
            <Typography variant={"caption"} color={"text.secondary"}>
              {t("application:fileManager.noSubtitle")}
            </Typography>
          </Box>
        )}
        {subtitles.length > 0 && (
          <SquareMenuItem onClick={() => switchSubtitle()} dense>
            <em>
              <ListItemText primary={t("application:fileManager.disableSubtitle")} />
            </em>
          </SquareMenuItem>
        )}
        {subtitles.map((sub) => (
          <Tooltip title={sub.name} key={sub.id}>
            <SquareMenuItem onClick={() => switchSubtitle(sub)} dense>
              <ListItemText
                primary={sub.name}
                slotProps={{
                  primary: {
                    sx: {
                      textOverflow: "ellipsis",
                      whiteSpace: "nowrap",
                      overflow: "hidden",
                    },
                  },
                }}
              />
              {subtitleSelected?.id == sub.id && (
                <ListItemIcon sx={{ minWidth: "0!important" }}>
                  <Checkmark />
                </ListItemIcon>
              )}
            </SquareMenuItem>
          </Tooltip>
        ))}
      </Menu>
      <Suspense fallback={<ViewerLoading minHeight={"calc(100vh - 350px)"} />}>
        <Player
          key={viewerState?.file?.path}
          m3u8UrlTransform={m3u8UrlTransform}
          getEntityUrl={getUnmaskedEntityUrl}
          sx={{
            width: "100%",
            height: "100%",
            minHeight: "calc(100vh - 350px)",
          }}
          chapters={chapters}
          getInstance={(instance) => setArt(instance)}
          option={{
            title: viewerState?.file?.name,
            theme: theme.palette.primary.main,
            id: viewerState?.file?.path,
            autoPlayback: true,
            subtitleOffset: true,
            fastForward: true,
            flip: true,
            setting: true,
            playbackRate: true,
            aspectRatio: true,
            hotkey: true,
            pip: !isMobile,
            fullscreen: true,
            fullscreenWeb: true,
            autoHeight: true,
            whitelist: ["*"],
            moreVideoAttr: {
              "webkit-playsinline": true,
              playsInline: true,
            },
            subtitle: {
              style: {
                color: subtitleStyle.fontColor ?? "#fff",
                fontSize: `${subtitleStyle.fontSize ?? 20}px`,
              },
            },
            plugins: [],
            lang: t("artPlayerLocaleCode", { ns: "common" }), // TODO: review
          }}
        />
      </Suspense>
    </ViewerDialog>
  );
};

export default VideoViewer;
