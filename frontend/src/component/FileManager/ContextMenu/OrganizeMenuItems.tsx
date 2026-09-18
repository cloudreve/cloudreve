import { DisplayOption } from "./useActionDisplayOpt.ts";
import { FileResponse } from "../../../api/explorer.ts";
import { useCallback, useContext } from "react";
import { CascadingContext, CascadingMenuItem } from "./CascadingMenu.tsx";
import { useTranslation } from "react-i18next";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { closeContextMenu } from "../../../redux/fileManagerSlice.ts";
import { applyIconColor, dialogBasedMoveCopy } from "../../../redux/thunks/file.ts";
import { ListItemIcon, ListItemText } from "@mui/material";
import FolderArrowRightOutlined from "../../Icons/FolderArrowRightOutlined.tsx";
import { setChangeIconDialog, setPinFileDialog } from "../../../redux/globalStateSlice.ts";
import { getFileLinkedUri } from "../../../util";
import PinOutlined from "../../Icons/PinOutlined.tsx";
import EmojiEdit from "../../Icons/EmojiEdit.tsx";
import FolderColorQuickAction from "../FileInfo/FolderColorQuickAction.tsx";
import { DenseDivider } from "./ContextMenu.tsx";

export interface SubMenuItemsProps {
  displayOpt: DisplayOption;
  targets: FileResponse[];
}
const OrganizeMenuItems = ({ displayOpt, targets }: SubMenuItemsProps) => {
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
  const showDivider =
    (displayOpt.showMove || displayOpt.showPin || displayOpt.showChangeIcon) && displayOpt.showChangeFolderColor;
  return (
    <>
      {displayOpt.showMove && (
        <CascadingMenuItem onClick={onClick(() => dispatch(dialogBasedMoveCopy(0, targets, false)))}>
          <ListItemIcon>
            <FolderArrowRightOutlined fontSize="small" />
          </ListItemIcon>
          <ListItemText>{t("application:fileManager.move")}</ListItemText>
        </CascadingMenuItem>
      )}
      {displayOpt.showPin && (
        <CascadingMenuItem
          onClick={onClick(() =>
            dispatch(
              setPinFileDialog({
                open: true,
                uri: getFileLinkedUri(targets[0]),
              }),
            ),
          )}
        >
          <ListItemIcon>
            <PinOutlined fontSize="small" />
          </ListItemIcon>
          <ListItemText>{t("application:fileManager.pin")}</ListItemText>
        </CascadingMenuItem>
      )}
      {displayOpt.showChangeIcon && (
        <CascadingMenuItem
          onClick={onClick(() =>
            dispatch(
              setChangeIconDialog({
                open: true,
                file: targets,
              }),
            ),
          )}
        >
          <ListItemIcon>
            <EmojiEdit fontSize="small" />
          </ListItemIcon>
          <ListItemText>{t("application:fileManager.customizeIcon")}</ListItemText>
        </CascadingMenuItem>
      )}
      {showDivider && <DenseDivider />}
      {displayOpt.showChangeFolderColor && (
        <FolderColorQuickAction
          file={targets[0]}
          onColorChange={(color) => onClick(() => dispatch(applyIconColor(0, targets, color, true)))()}
          sx={{
            maxWidth: "204px",
            margin: (theme) => `0 ${theme.spacing(0.5)}`,
            padding: (theme) => `${theme.spacing(0.5)} ${theme.spacing(1)}`,
          }}
        />
      )}
    </>
  );
};

export default OrganizeMenuItems;
