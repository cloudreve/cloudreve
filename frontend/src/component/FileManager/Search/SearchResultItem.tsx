import { ListItem, ListItemAvatar, ListItemButton, ListItemText } from "@mui/material";
import React from "react";
import { FileResponse, FileType } from "../../../api/explorer.ts";
import { ContextMenuTypes } from "../../../redux/fileManagerSlice.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { openFileContextMenu } from "../../../redux/thunks/file.ts";
import CrUri from "../../../util/uri.ts";
import FileIcon from "../Explorer/FileIcon.tsx";
import FileBadge from "../FileBadge.tsx";
import { FileManagerIndex } from "../FileManager.tsx";

export interface SearchResultItemProps {
  file: FileResponse;
  primary: React.ReactNode;
  secondary: React.ReactNode;
}

const SearchResultItem = ({ file, primary, secondary }: SearchResultItemProps) => {
  const dispatch = useAppDispatch();

  return (
    <ListItem disablePadding dense>
      <ListItemButton
        sx={{ py: 0 }}
        onClick={(e) =>
          dispatch(openFileContextMenu(FileManagerIndex.main, file, true, e, ContextMenuTypes.searchResult))
        }
      >
        <ListItemAvatar sx={{ minWidth: 48 }}>
          <FileIcon
            variant={"default"}
            file={file}
            sx={{ p: 0 }}
            iconProps={{
              sx: {
                fontSize: "24px",
                height: "32px",
                width: "32px",
              },
            }}
          />
        </ListItemAvatar>
        <ListItemText
          primary={primary}
          secondary={secondary}
          slotProps={{
            primary: {
              variant: "body2",
            },
            secondary: {
              variant: "body2",
            },
          }}
        />
        <FileBadge
          clickable
          variant={"outlined"}
          sx={{ px: 1, ml: 1, mt: "2px" }}
          simplifiedFile={{
            path: new CrUri(file.path).parent().toString(),
            type: FileType.folder,
          }}
        />
      </ListItemButton>
    </ListItem>
  );
};

export default SearchResultItem;
