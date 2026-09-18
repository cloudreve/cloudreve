import { ListItemIcon, ListItemText } from "@mui/material";
import { useCallback, useContext } from "react";
import { useTranslation } from "react-i18next";
import { closeContextMenu } from "../../../redux/fileManagerSlice.ts";
import {
  setCreateArchiveDialog,
  setDirectLinkManagementDialog,
  setManageShareDialog,
  setVersionControlDialog,
} from "../../../redux/globalStateSlice.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { resetThumbnails } from "../../../redux/thunks/file.ts";
import Archive from "../../Icons/Archive.tsx";
import BranchForkLink from "../../Icons/BranchForkLink.tsx";
import HistoryOutlined from "../../Icons/HistoryOutlined.tsx";
import ImageArrowCounterclockwise from "../../Icons/ImageAarowCounterclockwise.tsx";
import LinkSetting from "../../Icons/LinkSetting.tsx";
import { CascadingContext, CascadingMenuItem } from "./CascadingMenu.tsx";
import { SubMenuItemsProps } from "./OrganizeMenuItems.tsx";

const MoreMenuItems = ({ displayOpt, targets }: SubMenuItemsProps) => {
  const { rootPopupState } = useContext(CascadingContext);
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const onClick = useCallback(
    (f: () => any) => () => {
      f();
      if (rootPopupState) {
        rootPopupState.close();
      }
      dispatch(
        closeContextMenu({
          index: 0,
          value: undefined,
        }),
      );
    },
    [dispatch, targets],
  );
  return (
    <>
      {displayOpt.showVersionControl && (
        <CascadingMenuItem
          onClick={onClick(() =>
            dispatch(
              setVersionControlDialog({
                open: true,
                file: targets[0],
              }),
            ),
          )}
        >
          <ListItemIcon>
            <HistoryOutlined fontSize="small" />
          </ListItemIcon>
          <ListItemText>{t("application:fileManager.manageVersions")}</ListItemText>
        </CascadingMenuItem>
      )}
      {displayOpt.showManageShares && (
        <CascadingMenuItem
          onClick={onClick(() =>
            dispatch(
              setManageShareDialog({
                open: true,
                file: targets[0],
              }),
            ),
          )}
        >
          <ListItemIcon>
            <BranchForkLink fontSize="small" />
          </ListItemIcon>
          <ListItemText>{t("application:fileManager.manageShares")}</ListItemText>
        </CascadingMenuItem>
      )}
      {displayOpt.showDirectLinkManagement && (
        <CascadingMenuItem
          onClick={onClick(() =>
            dispatch(
              setDirectLinkManagementDialog({
                open: true,
                file: targets[0],
              }),
            ),
          )}
        >
          <ListItemIcon>
            <LinkSetting fontSize="small" />
          </ListItemIcon>
          <ListItemText>{t("application:fileManager.manageDirectLinks")}</ListItemText>
        </CascadingMenuItem>
      )}
      {displayOpt.showCreateArchive && (
        <CascadingMenuItem
          onClick={onClick(() =>
            dispatch(
              setCreateArchiveDialog({
                open: true,
                files: targets,
              }),
            ),
          )}
        >
          <ListItemIcon>
            <Archive fontSize="small" />
          </ListItemIcon>
          <ListItemText>{t("application:fileManager.createArchive")}</ListItemText>
        </CascadingMenuItem>
      )}
      {displayOpt.showResetThumb && (
        <CascadingMenuItem onClick={onClick(() => dispatch(resetThumbnails(targets)))}>
          <ListItemIcon>
            <ImageArrowCounterclockwise fontSize="small" />
          </ListItemIcon>
          <ListItemText>{t("application:fileManager.resetThumbnail")}</ListItemText>
        </CascadingMenuItem>
      )}
    </>
  );
};

export default MoreMenuItems;
