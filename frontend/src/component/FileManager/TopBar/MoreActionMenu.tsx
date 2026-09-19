import { ListItemIcon, ListItemText, MenuProps, Typography, useMediaQuery, useTheme } from "@mui/material";
import { usePopupState } from "material-ui-popup-state/hooks";
import { useCallback, useContext } from "react";
import { useTranslation } from "react-i18next";
import { clearSelected } from "../../../redux/fileManagerSlice.ts";
import { setReportAbuseDialog } from "../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { NavigatorCapability } from "../../../api/explorer.ts";
import { downloadAll } from "../../../redux/thunks/download.ts";
import { createShareShortcut, isMacbook } from "../../../redux/thunks/file.ts";
import { inverseSelection, pinCurrentView, refreshFileList, selectAll } from "../../../redux/thunks/filemanager.ts";
import SessionManager from "../../../session";
import Boolset from "../../../util/boolset.ts";
import CrUri, { Filesystem } from "../../../util/uri.ts";
import { KeyIndicator } from "../../Frame/NavBar/SearchBar.tsx";
import ArrowSync from "../../Icons/ArrowSync.tsx";
import Border from "../../Icons/Border.tsx";
import BorderAll from "../../Icons/BorderAll.tsx";
import BorderInside from "../../Icons/BorderInside.tsx";
import CloudDownloadOutlined from "../../Icons/CloudDownloadOutlined.tsx";
import FolderLink from "../../Icons/FolderLink.tsx";
import PinOutlined from "../../Icons/PinOutlined.tsx";
import WarningOutlined from "../../Icons/WarningOutlined.tsx";
import { DenseDivider, SquareMenu, SquareMenuItem } from "../ContextMenu/ContextMenu.tsx";
import { FmIndexContext } from "../FmIndexContext.tsx";

const MoreActionMenu = ({ onClose, ...rest }: MenuProps) => {
  const { t } = useTranslation();
  const fmIndex = useContext(FmIndexContext);
  const fs = useAppSelector((state) => state.fileManager[fmIndex].current_fs);
  const path = useAppSelector((state) => state.fileManager[fmIndex].path);
  const parentCapability = useAppSelector((state) => state.fileManager[fmIndex].list?.parent?.capability);
  const canDownloadAll = new Boolset(parentCapability).enabled(NavigatorCapability.download_file);
  const dispatch = useAppDispatch();
  const isLogin = !!SessionManager.currentLoginOrNull();
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));

  const mountPopupState = usePopupState({
    variant: "popover",
    popupId: "mount",
  });

  const onPinClicked = useCallback(() => {
    dispatch(pinCurrentView(fmIndex));
    onClose && onClose({}, "escapeKeyDown");
  }, [dispatch, onClose, fmIndex]);

  const onCreateShortcutClicked = useCallback(() => {
    dispatch(createShareShortcut(fmIndex));
    onClose && onClose({}, "escapeKeyDown");
  }, [dispatch, onClose, fmIndex]);

  const onSelectAllClicked = useCallback(() => {
    onClose && onClose({}, "escapeKeyDown");
    dispatch(selectAll(fmIndex));
  }, [dispatch, onClose, fmIndex]);

  const onSelectNoneClicked = useCallback(() => {
    onClose && onClose({}, "escapeKeyDown");
    dispatch(clearSelected({ index: fmIndex, value: undefined }));
  }, [dispatch, onClose, fmIndex]);

  const onInverseSelectionClicked = useCallback(() => {
    onClose && onClose({}, "escapeKeyDown");
    dispatch(inverseSelection(fmIndex));
  }, [dispatch, onClose, fmIndex]);

  const onRefreshClicked = useCallback(() => {
    onClose && onClose({}, "escapeKeyDown");
    dispatch(refreshFileList(fmIndex));
  }, [dispatch, onClose, fmIndex]);

  const onDownloadAllClicked = useCallback(() => {
    onClose && onClose({}, "escapeKeyDown");
    dispatch(downloadAll(fmIndex));
  }, [dispatch, onClose, fmIndex]);

  const onReportClicked = useCallback(() => {
    onClose && onClose({}, "escapeKeyDown");
    if (!path) {
      return;
    }
    const shareId = new CrUri(path).id();
    if (shareId) {
      dispatch(setReportAbuseDialog({ open: true, targetType: "share", target: shareId }));
    }
  }, [dispatch, onClose, path]);

  return (
    <SquareMenu
      MenuListProps={{
        dense: true,
      }}
      anchorOrigin={{
        vertical: "bottom",
        horizontal: "right",
      }}
      transformOrigin={{
        vertical: "top",
        horizontal: "right",
      }}
      onClose={onClose}
      {...rest}
    >
      {isMobile && (
        <>
          <SquareMenuItem onClick={onRefreshClicked}>
            <ListItemIcon>
              <ArrowSync fontSize="small" />
            </ListItemIcon>
            <ListItemText>{t("application:fileManager.refresh")}</ListItemText>
          </SquareMenuItem>
        </>
      )}
      {isLogin && (
        <SquareMenuItem onClick={onPinClicked}>
          <ListItemIcon>
            <PinOutlined fontSize="small" />
          </ListItemIcon>
          <ListItemText>{t("application:fileManager.pin")}</ListItemText>
        </SquareMenuItem>
      )}
      {isLogin && fs == Filesystem.share && (
        <SquareMenuItem onClick={onCreateShortcutClicked}>
          <ListItemIcon>
            <FolderLink fontSize="small" />
          </ListItemIcon>
          <ListItemText>{t("application:fileManager.saveShortcut")}</ListItemText>
        </SquareMenuItem>
      )}
      {fs == Filesystem.share && canDownloadAll && (
        <SquareMenuItem onClick={onDownloadAllClicked}>
          <ListItemIcon>
            <CloudDownloadOutlined fontSize="small" />
          </ListItemIcon>
          <ListItemText>{t("application:fileManager.downloadAll")}</ListItemText>
        </SquareMenuItem>
      )}
      {fs == Filesystem.share && (
        <SquareMenuItem onClick={onReportClicked}>
          <ListItemIcon>
            <WarningOutlined fontSize="small" />
          </ListItemIcon>
          <ListItemText>{t("application:vas.report")}</ListItemText>
        </SquareMenuItem>
      )}
      {isLogin && <DenseDivider />}
      <SquareMenuItem onClick={onSelectAllClicked}>
        <ListItemIcon>
          <BorderAll fontSize="small" />
        </ListItemIcon>
        <ListItemText>{t("application:fileManager.selectAll")}</ListItemText>
        <Typography variant="body2" color="text.secondary">
          <KeyIndicator>{isMacbook ? "⌘" : "Ctrl"}</KeyIndicator>+<KeyIndicator>A</KeyIndicator>
        </Typography>
      </SquareMenuItem>
      <SquareMenuItem onClick={onSelectNoneClicked}>
        <ListItemIcon>
          <Border fontSize="small" />
        </ListItemIcon>
        <ListItemText>{t("application:fileManager.selectNone")}</ListItemText>
      </SquareMenuItem>
      <SquareMenuItem onClick={onInverseSelectionClicked}>
        <ListItemIcon>
          <BorderInside fontSize="small" />
        </ListItemIcon>
        <ListItemText>{t("application:fileManager.invertSelection")}</ListItemText>
      </SquareMenuItem>
    </SquareMenu>
  );
};

export default MoreActionMenu;
