import { Badge, Box, IconButton, Stack, styled, Tooltip, useMediaQuery, useTheme } from "@mui/material";
import React, { forwardRef } from "react";
import { useTranslation } from "react-i18next";
import { FileResponse } from "../../../api/explorer.ts";
import { clearSelected, ContextMenuTypes } from "../../../redux/fileManagerSlice.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { downloadFiles } from "../../../redux/thunks/download.ts";
import {
  deleteFile,
  dialogBasedMoveCopy,
  openFileContextMenu,
  openShareDialog,
  renameFile,
  restoreFile,
} from "../../../redux/thunks/file.ts";
import { openViewers } from "../../../redux/thunks/viewer.ts";
import useActionDisplayOpt from "../../FileManager/ContextMenu/useActionDisplayOpt.ts";
import { FileManagerIndex } from "../../FileManager/FileManager.tsx";
import { ActionButton, ActionButtonGroup } from "../../FileManager/TopBar/TopActions.tsx";
import CopyOutlined from "../../Icons/CopyOutlined.tsx";
import DeleteOutlined from "../../Icons/DeleteOutlined.tsx";
import Dismiss from "../../Icons/Dismiss.tsx";
import Download from "../../Icons/Download.tsx";
import FolderArrowRightOutlined from "../../Icons/FolderArrowRightOutlined.tsx";
import HistoryOutlined from "../../Icons/HistoryOutlined.tsx";
import MoreHorizontal from "../../Icons/MoreHorizontal.tsx";
import Open from "../../Icons/Open.tsx";
import RenameOutlined from "../../Icons/RenameOutlined.tsx";
import ShareOutlined from "../../Icons/ShareOutlined.tsx";

export interface FileSelectedActionsProps {
  targets: FileResponse[];
}

const StyledActionButton = styled(ActionButton)(({ theme }) => ({
  // disabled
  "&.MuiButtonBase-root.Mui-disabled": {
    color: theme.palette.text.primary,
    fontWeight: theme.typography.fontWeightRegular,
    fontSize: theme.typography.body2.fontSize,
  },
}));

const StyledActionButtonGroup = styled(ActionButtonGroup)(({ theme }) => ({
  backgroundColor: theme.palette.background.default,
}));

const FileSelectedActions = forwardRef(({ targets }: FileSelectedActionsProps, ref: React.Ref<HTMLElement>) => {
  const dispatch = useAppDispatch();
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const isTablet = useMediaQuery(theme.breakpoints.down("md"));
  const { t } = useTranslation();
  const displayOpt = useActionDisplayOpt(targets, ContextMenuTypes.file);

  if (isMobile) {
    return (
      <Stack direction={"row"} spacing={1} sx={{ height: "100%" }}>
        <IconButton
          onClick={() =>
            dispatch(
              clearSelected({
                index: FileManagerIndex.main,
                value: undefined,
              }),
            )
          }
        >
          <Badge badgeContent={targets.length} color={"primary"}>
            <Dismiss />
          </Badge>
        </IconButton>
        <IconButton onClick={(e) => dispatch(openFileContextMenu(FileManagerIndex.main, targets[0], false, e))}>
          <Badge badgeContent={targets.length} color={"primary"}>
            <MoreHorizontal />
          </Badge>
        </IconButton>
      </Stack>
    );
  }

  return (
    <Box ref={ref} sx={{ height: "100%" }}>
      <Stack direction={"row"} spacing={1} sx={{ height: "100%" }}>
        <StyledActionButtonGroup variant="outlined">
          <ActionButton
            onClick={() =>
              dispatch(
                clearSelected({
                  index: FileManagerIndex.main,
                  value: undefined,
                }),
              )
            }
          >
            <Dismiss fontSize={"small"} />
          </ActionButton>
          <StyledActionButton disabled sx={{ color: (theme) => theme.palette.text.primary }}>
            {t("application:navbar.objectsSelected", {
              num: targets.length,
            })}
          </StyledActionButton>
        </StyledActionButtonGroup>
        {!isTablet && (
          <StyledActionButtonGroup variant="outlined">
            {displayOpt.showOpen && (
              <Tooltip title={t("application:fileManager.open")}>
                <ActionButton onClick={() => dispatch(openViewers(0, targets[0]))}>
                  <Open fontSize={"small"} />
                </ActionButton>
              </Tooltip>
            )}
            {displayOpt.showDownload && (
              <Tooltip title={t("application:fileManager.download")}>
                <ActionButton onClick={() => dispatch(downloadFiles(0, targets))}>
                  <Download fontSize={"small"} />
                </ActionButton>
              </Tooltip>
            )}
            {displayOpt.showCopy && (
              <Tooltip title={t("application:fileManager.copy")}>
                <ActionButton onClick={() => dispatch(dialogBasedMoveCopy(0, targets, true))}>
                  <CopyOutlined fontSize={"small"} />
                </ActionButton>
              </Tooltip>
            )}
            {displayOpt.showMove && (
              <Tooltip title={t("application:fileManager.move")}>
                <ActionButton onClick={() => dispatch(dialogBasedMoveCopy(0, targets, false))}>
                  <FolderArrowRightOutlined fontSize={"small"} />
                </ActionButton>
              </Tooltip>
            )}
            {displayOpt.showRestore && (
              <Tooltip title={t("application:fileManager.restore")}>
                <ActionButton onClick={() => dispatch(restoreFile(FileManagerIndex.main, targets))}>
                  <HistoryOutlined fontSize={"small"} />
                </ActionButton>
              </Tooltip>
            )}
            {displayOpt.showRename && (
              <Tooltip title={t("application:fileManager.rename")}>
                <ActionButton onClick={() => dispatch(renameFile(0, targets[0]))}>
                  <RenameOutlined fontSize={"small"} />
                </ActionButton>
              </Tooltip>
            )}
            {displayOpt.showShare && (
              <Tooltip title={t("application:fileManager.share")}>
                <ActionButton onClick={() => dispatch(openShareDialog(0, targets[0]))}>
                  <ShareOutlined fontSize={"small"} />
                </ActionButton>
              </Tooltip>
            )}
            {displayOpt.showDelete && (
              <Tooltip title={t("application:fileManager.delete")}>
                <ActionButton onClick={() => dispatch(deleteFile(0, targets))}>
                  <DeleteOutlined fontSize="small" />
                </ActionButton>
              </Tooltip>
            )}
          </StyledActionButtonGroup>
        )}
        <StyledActionButtonGroup variant="outlined">
          <ActionButton onClick={(e) => dispatch(openFileContextMenu(FileManagerIndex.main, targets[0], false, e))}>
            <MoreHorizontal fontSize={"small"} />
          </ActionButton>
        </StyledActionButtonGroup>
      </Stack>
    </Box>
  );
});

export default FileSelectedActions;
