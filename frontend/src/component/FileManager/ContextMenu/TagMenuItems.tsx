import { Box, IconButton, ListItemIcon, ListItemText } from "@mui/material";
import React, { useCallback, useContext, useState } from "react";
import { useTranslation } from "react-i18next";
import { FileResponse, Metadata } from "../../../api/explorer.ts";
import { closeContextMenu } from "../../../redux/fileManagerSlice.ts";
import { setTagsDialog } from "../../../redux/globalStateSlice.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { patchFileMetadata } from "../../../redux/thunks/file.ts";
import SessionManager, { UserSettings } from "../../../session";
import { UsedTags } from "../../../session/utils.ts";
import Checkmark from "../../Icons/Checkmark.tsx";
import DeleteOutlined from "../../Icons/DeleteOutlined.tsx";
import Tags from "../../Icons/Tags.tsx";
import { getUniqueTagsFromFiles, Tag as TagItem } from "../Dialogs/Tags.tsx";
import FileTag from "../Explorer/FileTag.tsx";
import { FileManagerIndex } from "../FileManager.tsx";
import { CascadingContext, CascadingMenuItem } from "./CascadingMenu.tsx";
import { DenseDivider, SquareMenuItem } from "./ContextMenu.tsx";
import { SubMenuItemsProps } from "./OrganizeMenuItems.tsx";

interface TagOption extends TagItem {
  selected?: boolean;
}

const getTagOptions = (targets: FileResponse[]): TagOption[] => {
  const tags: {
    [key: string]: TagOption;
  } = {};
  getUniqueTagsFromFiles(targets).forEach((tag) => {
    tags[tag.key] = { ...tag, selected: true };
  });

  const existing = SessionManager.get(UserSettings.UsedTags) as UsedTags;
  if (existing) {
    Object.keys(existing).forEach((key) => {
      if (!tags[key]) {
        tags[key] = { key, color: existing[key] ?? undefined, selected: false };
      }
    });
  }

  return Object.values(tags);
};

const TagMenuItems = ({ displayOpt, targets }: SubMenuItemsProps) => {
  const { rootPopupState } = useContext(CascadingContext);
  const [tags, setTags] = useState<TagOption[]>(getTagOptions(targets));
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

  const onTagChange = useCallback(
    async (tag: TagOption, selected: boolean) => {
      setTags((tags) =>
        tags.map((t) => {
          if (t.key == tag.key) {
            return { ...t, selected };
          }
          return t;
        }),
      );
      try {
        await dispatch(
          patchFileMetadata(FileManagerIndex.main, targets, [
            {
              key: Metadata.tag_prefix + tag.key,
              value: tag.color,
              remove: !selected,
            },
          ]),
        );
      } catch (e) {
        return;
      }
    },
    [targets, setTags],
  );

  const onTagDelete = useCallback(
    (tag: TagOption, event: React.MouseEvent) => {
      event.stopPropagation();

      // Remove tag from session cache
      const existing = SessionManager.get(UserSettings.UsedTags) as UsedTags;
      if (existing && existing[tag.key] !== undefined) {
        delete existing[tag.key];
        SessionManager.set(UserSettings.UsedTags, existing);
      }

      // Remove tag from local state
      setTags((tags) => tags.filter((t) => t.key !== tag.key));
    },
    [setTags],
  );

  return (
    <>
      <CascadingMenuItem
        onClick={onClick(() =>
          dispatch(
            setTagsDialog({
              open: true,
              file: targets,
            }),
          ),
        )}
      >
        <ListItemIcon>
          <Tags fontSize="small" />
        </ListItemIcon>
        <ListItemText>{t("application:modals.manageTags")}</ListItemText>
      </CascadingMenuItem>
      {tags.length > 0 && <DenseDivider />}
      {tags.map((tag) => (
        <SquareMenuItem key={tag.key} onClick={() => onTagChange(tag, !tag.selected)}>
          {tag.selected && (
            <>
              <ListItemIcon>
                <Checkmark />
              </ListItemIcon>
              <ListItemText>
                <FileTag disableClick spacing={1} label={tag.key} tagColor={tag.color} />
              </ListItemText>
            </>
          )}
          {!tag.selected && (
            <Box
              sx={{
                display: "flex",
                alignItems: "center",
                width: "100%",
                "&:hover .delete-button": {
                  opacity: 1,
                },
              }}
            >
              <ListItemText inset>
                <FileTag disableClick spacing={1} label={tag.key} tagColor={tag.color} />
              </ListItemText>
              <IconButton
                className="delete-button"
                size="small"
                onClick={(event) => onTagDelete(tag, event)}
                sx={{
                  opacity: 0,
                  transition: "opacity 0.2s",
                  marginLeft: "auto",
                  marginRight: 1,
                  padding: "2px",
                  "&:hover": {
                    backgroundColor: "action.hover",
                  },
                }}
              >
                <DeleteOutlined fontSize="small" />
              </IconButton>
            </Box>
          )}
        </SquareMenuItem>
      ))}
    </>
  );
};

export default TagMenuItems;
