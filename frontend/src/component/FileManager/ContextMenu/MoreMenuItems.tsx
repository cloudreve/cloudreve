import { ListItemIcon, ListItemText } from "@mui/material";
import { useCallback, useContext } from "react";
import { useTranslation } from "react-i18next";
import { closeContextMenu } from "../../../redux/fileManagerSlice.ts";
import {
  setAclDialog,
  setActivityDialog,
  setCreateArchiveDialog,
  setDirectLinkManagementDialog,
  setManageShareDialog,
  setStoragePolicyDialog,
  setVersionControlDialog,
} from "../../../redux/globalStateSlice.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { resetThumbnails } from "../../../redux/thunks/file.ts";
import Archive from "../../Icons/Archive.tsx";
import BoxMultiple from "../../Icons/BoxMultiple.tsx";
import BranchForkLink from "../../Icons/BranchForkLink.tsx";
import CloudArrowIUp from "../../Icons/CloudArrowIUp.tsx";
import HistoryOutlined from "../../Icons/HistoryOutlined.tsx";
import ImageArrowCounterclockwise from "../../Icons/ImageAarowCounterclockwise.tsx";
import LinkSetting from "../../Icons/LinkSetting.tsx";
import PersonLock from "../../Icons/PersonLock.tsx";
import TaskListOutlined from "../../Icons/TaskListOutlined.tsx";
import { CascadingContext, CascadingMenuItem } from "./CascadingMenu.tsx";
import { SubMenuItemsProps } from "./OrganizeMenuItems.tsx";

const MoreMenuItems = ({ displayOpt, targets }: SubMenuItemsProps) => {
  const { rootPopupState } = useContext(CascadingContext);
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const onClick = useCallback(
    (f: () => void) => () => {
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
      {displayOpt.showAcl && (
        <CascadingMenuItem
          onClick={onClick(() =>
            dispatch(
              setAclDialog({
                open: true,
                file: targets[0],
              }),
            ),
          )}
        >
          <ListItemIcon>
            <PersonLock fontSize="small" />
          </ListItemIcon>
          <ListItemText>{t("application:fileManager.permissions")}</ListItemText>
        </CascadingMenuItem>
      )}
      {displayOpt.showActivity && (
        <CascadingMenuItem
          onClick={onClick(() =>
            dispatch(
              setActivityDialog({
                open: true,
                file: targets[0],
              }),
            ),
          )}
        >
          <ListItemIcon>
            <TaskListOutlined fontSize="small" />
          </ListItemIcon>
          <ListItemText>{t("application:fileManager.activity")}</ListItemText>
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
      {displayOpt.showDirPolicy && (
        <CascadingMenuItem
          onClick={onClick(() =>
            dispatch(
              setStoragePolicyDialog({
                open: true,
                mode: "dir",
                file: targets[0],
              }),
            ),
          )}
        >
          <ListItemIcon>
            <CloudArrowIUp fontSize="small" />
          </ListItemIcon>
          <ListItemText>{t("application:fileManager.dirPolicyTitle")}</ListItemText>
        </CascadingMenuItem>
      )}
      {displayOpt.showRelocate && (
        <CascadingMenuItem
          onClick={onClick(() =>
            dispatch(
              setStoragePolicyDialog({
                open: true,
                mode: "relocate",
                file: targets[0],
              }),
            ),
          )}
        >
          <ListItemIcon>
            <BoxMultiple fontSize="small" />
          </ListItemIcon>
          <ListItemText>{t("application:fileManager.relocateTitle")}</ListItemText>
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
