import { FileResponse, Metadata } from "../../../api/explorer.ts";
import { useTranslation } from "react-i18next";
import { Box, Typography } from "@mui/material";
import React, { useMemo } from "react";
import { useAppDispatch } from "../../../redux/hooks.ts";
import FileTag from "../Explorer/FileTag.tsx";

export interface TagsProps {
  target: FileResponse;
}

export const getFileTags = (file: FileResponse) => {
  if (file.metadata) {
    const tags: { key: string; value: string }[] = [];
    Object.keys(file.metadata).forEach((key: string) => {
      if (key.startsWith(Metadata.tag_prefix)) {
        // trim prefix for key
        tags.push({
          key: key.slice(Metadata.tag_prefix.length),
          value: file.metadata?.[key] ?? "",
        });
      }
    });
    return tags;
  }
};

const Tags = ({ target }: TagsProps) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();

  const fileTag = useMemo(() => getFileTags(target), [target]);

  if (!fileTag || fileTag.length === 0) {
    return null;
  }

  return (
    <>
      <Typography sx={{ pt: 1 }} color="textPrimary" fontWeight={500} variant={"subtitle1"}>
        {t("application:fileManager.tags")}
      </Typography>
      <Box>
        {fileTag.map((tag, i) => (
          <FileTag
            tagColor={tag.value == "" ? undefined : tag.value}
            label={tag.key}
            spacing={i === fileTag.length - 1 ? undefined : 1}
            key={tag.key}
          />
        ))}
      </Box>
    </>
  );
};

export default Tags;
