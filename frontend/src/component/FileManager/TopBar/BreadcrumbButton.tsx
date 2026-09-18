import { Button, Skeleton, styled, SvgIconProps, Tooltip } from "@mui/material";
import { bindHover, bindPopover } from "material-ui-popup-state";
import { usePopupState } from "material-ui-popup-state/hooks";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Share } from "../../../api/explorer.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { navigateToPath, openContextUrlFromUri } from "../../../redux/thunks/filemanager.ts";
import { queueLoadShareInfo } from "../../../redux/thunks/share.ts";
import PageTitle from "../../../router/PageTitle.tsx";
import SessionManager from "../../../session";
import CrUri, { Filesystem, UriQuery, UriSearchCategory } from "../../../util/uri.ts";
import { NoWrapBox } from "../../Common/StyledComponents.tsx";
import UserAvatar from "../../Common/User/UserAvatar.tsx";
import CaretDown from "../../Icons/CaretDown.tsx";
import Delete from "../../Icons/Delete.tsx";
import DeleteOutlined from "../../Icons/DeleteOutlined.tsx";
import DocumentText from "../../Icons/DocumentText.tsx";
import DocumentTextOutlined from "../../Icons/DocumentTextOutlined.tsx";
import Home from "../../Icons/Home.tsx";
import HomeOutlined from "../../Icons/HomeOutlined.tsx";
import Image from "../../Icons/Image.tsx";
import ImageOutlined from "../../Icons/ImageOutlined.tsx";
import LinkDismiss from "../../Icons/LinkDismiss.tsx";
import MusicNote1 from "../../Icons/MusicNote1.tsx";
import MusicNote1Outlined from "../../Icons/MusicNote1Outlined.tsx";
import PeopleTeam from "../../Icons/PeopleTeam.tsx";
import PeopleTeamOutlined from "../../Icons/PeopleTeamOutlined.tsx";
import Video from "../../Icons/Video.tsx";
import VideoOutlined from "../../Icons/VideoOutlined.tsx";
import { useFileDrag } from "../Dnd/DndWrappedFile.tsx";
import { FileManagerIndex } from "../FileManager.tsx";
import { FmIndexContext } from "../FmIndexContext.tsx";
import ShareInfoPopover from "./ShareInfoPopover.tsx";

export const BreadcrumbButtonBase = styled(Button)<{ isDropOver?: boolean }>(({ theme, isDropOver }) => ({
  color: theme.palette.text.secondary,
  transition: "all 250ms cubic-bezier(0.4, 0, 0.2, 1) 0ms !important",
  transitionProperty: "background-color,opacity,box-shadow",
  boxShadow: isDropOver ? `inset 0 0 0 2px ${theme.palette.primary.light}` : "none",
  minHeight: theme.spacing(4),
}));

export interface BreadcrumbButtonProps {
  path: string;
  name?: string;
  is_latest?: boolean;
  displayOnly?: boolean;
  is_root?: boolean;
  count_share_views?: boolean;
  [key: string]: any;
}

export interface StartIcon {
  Element?: (props: { [key: string]: any }) => JSX.Element;
  Icons?: ((props: SvgIconProps) => JSX.Element)[];
}

export const useBreadcrumbButtons = ({
  name,
  is_latest,
  path,
  displayOnly,
  is_root,
  count_share_views,
}: BreadcrumbButtonProps) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const fmIndex = useContext(FmIndexContext);
  const [loading, setLoading] = useState(false);
  const [shareInfo, setShareInfo] = useState<Share | undefined | null>(undefined);

  const uri = useMemo(() => {
    return new CrUri(path);
  }, [path]);

  useEffect(() => {
    if (uri.is_root() && uri.fs() == Filesystem.share) {
      setLoading(true);
      dispatch(queueLoadShareInfo(uri, count_share_views))
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
      setShareInfo(undefined);
    }
  }, [uri]);

  const onClick = useCallback(
    (e?: React.MouseEvent<HTMLElement>) => {
      e && e.stopPropagation();
      if (is_latest && !displayOnly && !is_root && e) {
        dispatch(openContextUrlFromUri(fmIndex, path, e));
        return;
      }
      dispatch(navigateToPath(fmIndex, path, undefined, displayOnly));
    },
    [dispatch, fmIndex, path, is_latest, displayOnly, is_root],
  );

  const displayName = useMemo(() => {
    if (uri.is_root()) {
      switch (uri.fs()) {
        case Filesystem.my:
          if (uri.is_search()) {
            const searchCategory = uri.query(UriQuery.category);
            if (searchCategory.length == 1) {
              switch (searchCategory[0]) {
                case UriSearchCategory.image:
                  return t("navbar.photos");
                case UriSearchCategory.video:
                  return t("navbar.videos");
                case UriSearchCategory.audio:
                  return t("navbar.music");
                case UriSearchCategory.document:
                  return t("navbar.documents");
              }
            }
          }
          if (uri.id()) {
            const current = SessionManager.currentLoginOrNull();
            if (!current || current.user.id != uri.id()) {
              return t("navbar.hisFiles");
            }
          }
          return t("navbar.myFiles");
        case Filesystem.trash:
          return t("navbar.trash");
        case Filesystem.shared_by_me:
          return t("navbar.myShare");
        case Filesystem.shared_with_me:
          return t("navbar.sharedWithMe");
        case Filesystem.share:
          if (shareInfo) {
            return shareInfo.name
              ? shareInfo.name
              : t("application:share.somebodyShare", {
                  name: shareInfo.owner.nickname,
                });
          } else if (shareInfo === null) {
            return t("application:share.expiredLink");
          }

          return "";
        default:
          "Root";
      }
    }

    return name;
  }, [name, uri, t, shareInfo]);

  const startIcon = useMemo(() => {
    if (uri.is_root()) {
      switch (uri.fs()) {
        case Filesystem.my:
          if (uri.is_search()) {
            const searchCategory = uri.query(UriQuery.category);
            if (searchCategory.length == 1) {
              switch (searchCategory[0]) {
                case UriSearchCategory.image:
                  return { Icons: [Image, ImageOutlined] };
                case UriSearchCategory.video:
                  return { Icons: [Video, VideoOutlined] };
                case UriSearchCategory.audio:
                  return { Icons: [MusicNote1, MusicNote1Outlined] };
                case UriSearchCategory.document:
                  return { Icons: [DocumentText, DocumentTextOutlined] };
              }
            }
          }
          const uid = uri.id();
          if (uid) {
            const current = SessionManager.currentLoginOrNull();
            if (!current || current.user.id != uri.id()) {
              return {
                Element: (props: { [key: string]: any }) => (
                  <UserAvatar sx={{ width: 20, height: 20 }} overwriteTextSize key={uid} uid={uid} {...props} />
                ),
              };
            }
          }
          return { Icons: [Home, HomeOutlined] };
        case Filesystem.trash:
          return { Icons: [Delete, DeleteOutlined] };
        case Filesystem.shared_with_me:
          return { Icons: [PeopleTeam, PeopleTeamOutlined] };
        case Filesystem.share:
          if (shareInfo) {
            return {
              Element: (props: { [key: string]: any }) => (
                <UserAvatar
                  sx={{ width: 20, height: 20 }}
                  overwriteTextSize
                  key={shareInfo.owner.id}
                  user={shareInfo.owner}
                  {...props}
                />
              ),
            };
          } else if (shareInfo === null) {
            return { Icons: [LinkDismiss, LinkDismiss] };
          }
          return undefined;
        default:
          return undefined;
      }
    }
  }, [uri, shareInfo]);

  return [loading, displayName, startIcon, onClick, shareInfo] as const;
};

const BreadcrumbButton = ({ name, is_root, is_latest, path, displayOnly, ...rest }: BreadcrumbButtonProps) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const fmIndex = useContext(FmIndexContext);

  const [loading, displayName, startIcon, onClick, shareInfo] = useBreadcrumbButtons({
    name,
    is_latest,
    path,
    displayOnly,
    is_root,
    count_share_views: true,
  });
  const maxWidth = is_latest ? "300px" : is_root ? "initial" : "100px";
  const StartIcon = useMemo(() => {
    if (loading) {
      return <Skeleton width={18} height={18} variant={"rounded"} />;
    }
    if (startIcon?.Icons?.[0]) {
      const Icon = startIcon?.Icons?.[0];
      return <Icon sx={{ ml: 0.5 }} />;
    }
    if (startIcon?.Element) {
      return startIcon.Element({ sx: { ml: 0.5, width: 20, height: 20 } });
    }
    return null;
  }, [startIcon, loading]);

  const [drag, drop, isOver, isDragging] = useFileDrag({
    dropUri: path,
  });

  const popupState = usePopupState({
    variant: "popover",
    popupId: "shareInfo",
  });

  return (
    <>
      {is_latest && !displayOnly && fmIndex == FileManagerIndex.main && <PageTitle title={displayName} />}
      <BreadcrumbButtonBase
        onClick={onClick}
        size={"small"}
        isDropOver={isOver}
        sx={{
          transition: (theme) =>
            theme.transitions.create(["max-width", "color", "background-color"], {
              easing: theme.transitions.easing.easeInOut,
              duration: theme.transitions.duration.standard,
            }),
          color: (theme) => (is_latest && !displayOnly ? theme.palette.text.primary : theme.palette.text.secondary),
          maxWidth: maxWidth,
        }}
        startIcon={StartIcon}
        endIcon={is_latest && !is_root && !displayOnly ? <CaretDown sx={{ fontSize: "12px!important" }} /> : undefined}
        ref={drop}
        {...(shareInfo ? bindHover(popupState) : {})}
        {...rest}
      >
        {loading && <Skeleton width={75} variant="text" />}
        {!loading && (
          <Tooltip title={shareInfo ? "" : displayName}>
            <NoWrapBox>{displayName}</NoWrapBox>
          </Tooltip>
        )}
      </BreadcrumbButtonBase>
      {shareInfo && displayName && (
        <ShareInfoPopover displayName={displayName} shareInfo={shareInfo} {...bindPopover(popupState)} />
      )}
    </>
  );
};

export default BreadcrumbButton;
