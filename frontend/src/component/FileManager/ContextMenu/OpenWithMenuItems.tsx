import React, { useCallback, useContext, useMemo } from "react";
import { CascadingContext, CascadingMenuItem } from "./CascadingMenu.tsx";
import { useTranslation } from "react-i18next";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { closeContextMenu } from "../../../redux/fileManagerSlice.ts";
import { ListItemIcon, ListItemText } from "@mui/material";
import { SubMenuItemsProps } from "./OrganizeMenuItems.tsx";
import { fileExtension } from "../../../util";
import { Viewers } from "../../../redux/siteConfigSlice.ts";
import { Viewer } from "../../../api/explorer.ts";
import { ViewerIcon } from "../Dialogs/OpenWith.tsx";
import { openViewer, openViewers } from "../../../redux/thunks/viewer.ts";

const OpenWithMenuItems = ({ targets }: SubMenuItemsProps) => {
  const { rootPopupState } = useContext(CascadingContext);
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const onClick = useCallback(
    (v: Viewer) => () => {
      dispatch(openViewer(targets[0], v, targets[0].size));
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

  const openSelector = useCallback(() => {
    dispatch(openViewers(0, targets[0], targets[0].size, undefined, true));
  }, [targets]);

  const viewers = useMemo(() => {
    if (targets.length == 0) {
      return [];
    }

    const firstFileSuffix = fileExtension(targets[0].name);
    return Viewers[firstFileSuffix ?? ""];
  }, [targets]);

  return (
    <>
      {viewers.map((viewer) => (
        <CascadingMenuItem key={viewer.id} onClick={onClick(viewer)}>
          <ListItemIcon>
            <ViewerIcon size={20} viewer={viewer} py={0} />
          </ListItemIcon>
          <ListItemText>{t(viewer.display_name)}</ListItemText>
        </CascadingMenuItem>
      ))}
      <CascadingMenuItem onClick={openSelector}>
        <ListItemIcon></ListItemIcon>
        <ListItemText>{t("fileManager.selectApplications")}</ListItemText>
      </CascadingMenuItem>
    </>
  );
};

export default OpenWithMenuItems;
