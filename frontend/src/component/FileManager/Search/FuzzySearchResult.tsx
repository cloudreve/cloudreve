import { FileResponse, FileType, Metadata } from "../../../api/explorer.ts";
import { List } from "@mui/material";
import { useTranslation } from "react-i18next";
import { sizeToString } from "../../../util";
import { useCallback } from "react";
// @ts-ignore
import Highlighter from "react-highlight-words";
import SearchResultItem from "./SearchResultItem.tsx";

export interface FuzzySearchResultProps {
  files: FileResponse[];
  keyword: string;
}

const FuzzySearchResult = ({ files, keyword }: FuzzySearchResultProps) => {
  const { t } = useTranslation();
  const getFileTypeText = useCallback(
    (file: FileResponse) => {
      if (file.metadata?.[Metadata.share_redirect]) {
        return t("fileManager.symbolicFile");
      }

      if (file.type == FileType.folder) {
        return t("application:fileManager.folder");
      }
      return sizeToString(file.size);
    },
    [t],
  );

  return (
    <List sx={{ width: "100%", px: 1 }} dense>
      {files.map((file) => (
        <SearchResultItem
          key={file.id}
          file={file}
          primary={
            <Highlighter
              highlightClassName="highlight-marker"
              searchWords={keyword.split(" ")}
              autoEscape={true}
              textToHighlight={file.name}
            />
          }
          secondary={getFileTypeText(file)}
        />
      ))}
    </List>
  );
};

export default FuzzySearchResult;
