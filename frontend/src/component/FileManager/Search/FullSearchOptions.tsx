import { Box, List, ListItem, ListItemAvatar, ListItemButton, ListItemText } from "@mui/material";
import { Trans, useTranslation } from "react-i18next";
import React, { useCallback } from "react";
// @ts-ignore
import Highlighter from "react-highlight-words";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { SearchOutlined } from "@mui/icons-material";
import { FileType } from "../../../api/explorer.ts";
import FileBadge from "../FileBadge.tsx";
import { quickSearch } from "../../../redux/thunks/filemanager.ts";
import { FileManagerIndex } from "../FileManager.tsx";

export interface FullSearchOptionProps {
  options: string[];
  keyword: string;
}

const FullSearchOption = ({ options, keyword }: FullSearchOptionProps) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();

  const onClick = useCallback(
    (base: string) => () => dispatch(quickSearch(FileManagerIndex.main, base, keyword)),
    [keyword, dispatch],
  );

  return (
    <List sx={{ width: "100%", px: 1 }} dense>
      {options.map((option) => (
        <ListItem disablePadding dense>
          <ListItemButton onClick={onClick(option)} sx={{ py: 0 }}>
            <ListItemAvatar sx={{ minWidth: 48 }}>
              <SearchOutlined
                sx={{
                  color: (theme) => theme.palette.action.active,
                  width: 24,
                  height: 24,
                  mt: "7px",
                  ml: "5px",
                }}
              />
            </ListItemAvatar>
            <ListItemText
              primary={
                <Trans
                  ns={"application"}
                  i18nKey={"navbar.searchIn"}
                  values={{
                    keywords: keyword,
                  }}
                  components={[<Box component={"span"} sx={{ fontWeight: 600 }} />]}
                />
              }
              slotProps={{
                primary: {
                  variant: "body2",
                },
              }}
            />
            <FileBadge
              clickable
              variant={"outlined"}
              sx={{ px: 1, my: "4px" }}
              simplifiedFile={{
                path: option,
                type: FileType.folder,
              }}
            />
          </ListItemButton>
        </ListItem>
      ))}
    </List>
  );
};

export default FullSearchOption;
