import * as React from "react";
import { useCallback, useState } from "react";
import { Viewer, ViewerPlatform, ViewerType } from "../../../../api/explorer.ts";
import { useTranslation } from "react-i18next";
import { IconButton, ListItemText, TableRow } from "@mui/material";
import { DenseFilledTextField, DenseSelect, NoWrapCell, StyledCheckbox } from "../../../Common/StyledComponents.tsx";
import { SquareMenuItem } from "../../../FileManager/ContextMenu/ContextMenu.tsx";
import { ViewerIcon } from "../../../FileManager/Dialogs/OpenWith.tsx";
import Dismiss from "../../../Icons/Dismiss.tsx";
import Edit from "../../../Icons/Edit.tsx";
import FileViewerEditDialog from "./FileViewerEditDialog.tsx";
import ArrowDown from "../../../Icons/ArrowDown.tsx";

export interface FileViewerRowProps {
  viewer: Viewer;
  onChange: (viewer: Viewer) => void;
  onDelete: (e: React.MouseEvent<HTMLElement>) => void;
  onMoveUp?: () => void;
  onMoveDown?: () => void;
  isFirst?: boolean;
  isLast?: boolean;
  style?: React.CSSProperties;
}

const FileViewerRow = React.memo(
  React.forwardRef<HTMLTableRowElement, FileViewerRowProps>(
    ({ viewer, onChange, onDelete, onMoveUp, onMoveDown, isFirst, isLast, style }, ref) => {
      const { t } = useTranslation("dashboard");
      const [extCached, setExtCached] = useState("");
      const [editOpen, setEditOpen] = useState(false);
      const onClose = useCallback(() => {
        setEditOpen(false);
      }, [setEditOpen]);
      return (
        <TableRow sx={{ "&:last-child td, &:last-child th": { border: 0 } }} hover ref={ref} style={style}>
          <FileViewerEditDialog viewer={viewer} onChange={onChange} open={editOpen} onClose={onClose} />
          <NoWrapCell>
            <ViewerIcon viewer={viewer} />
          </NoWrapCell>
          <NoWrapCell>{t(`settings.${viewer.type}ViewerType`)}</NoWrapCell>
          <NoWrapCell>
            {t(viewer.display_name, {
              ns: "application",
            })}
          </NoWrapCell>
          <NoWrapCell>
            <DenseFilledTextField
              fullWidth
              multiline
              required
              value={extCached == "" ? viewer.exts.join() : extCached}
              onBlur={() => {
                onChange({
                  ...viewer,
                  exts: extCached == "" ? viewer.exts : extCached?.split(",")?.map((ext) => ext.trim()),
                });
                setExtCached("");
              }}
              onChange={(e) => setExtCached(e.target.value)}
            />
          </NoWrapCell>
          <NoWrapCell>
            <DenseSelect
              value={viewer.platform || ViewerPlatform.all}
              onChange={(e) =>
                onChange({
                  ...viewer,
                  platform: e.target.value as ViewerPlatform,
                })
              }
            >
              <SquareMenuItem value="pc">
                <ListItemText
                  slotProps={{
                    primary: { variant: "body2" },
                  }}
                >
                  {t("settings.viewerPlatformPC")}
                </ListItemText>
              </SquareMenuItem>
              <SquareMenuItem value="mobile">
                <ListItemText
                  slotProps={{
                    primary: { variant: "body2" },
                  }}
                >
                  {t("settings.viewerPlatformMobile")}
                </ListItemText>
              </SquareMenuItem>
              <SquareMenuItem value="all">
                <ListItemText
                  slotProps={{
                    primary: { variant: "body2" },
                  }}
                >
                  {t("settings.viewerPlatformAll")}
                </ListItemText>
              </SquareMenuItem>
            </DenseSelect>
          </NoWrapCell>
          <NoWrapCell>
            {viewer.templates?.length ? t("settings.nMapping", { num: viewer.templates?.length }) : t("share.none")}
          </NoWrapCell>
          <NoWrapCell>
            <StyledCheckbox
              size={"small"}
              checked={!viewer.disabled}
              onChange={(e) =>
                onChange({
                  ...viewer,
                  disabled: !e.target.checked,
                })
              }
            />
          </NoWrapCell>
          <NoWrapCell>
            <IconButton size={"small"} onClick={() => setEditOpen(true)}>
              <Edit fontSize={"small"} />
            </IconButton>
            {viewer.type != ViewerType.builtin && (
              <IconButton size={"small"} onClick={onDelete}>
                <Dismiss fontSize={"small"} />
              </IconButton>
            )}
          </NoWrapCell>
          <NoWrapCell>
            <IconButton size="small" onClick={onMoveUp} disabled={isFirst}>
              <ArrowDown
                sx={{
                  width: "18px",
                  height: "18px",
                  transform: "rotate(180deg)",
                }}
              />
            </IconButton>
            <IconButton size="small" onClick={onMoveDown} disabled={isLast}>
              <ArrowDown
                sx={{
                  width: "18px",
                  height: "18px",
                }}
              />
            </IconButton>
          </NoWrapCell>
        </TableRow>
      );
    },
  ),
);

export default FileViewerRow;
