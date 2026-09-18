import { Stack } from "@mui/material";
import { memo, useCallback, useContext } from "react";
import { useTranslation } from "react-i18next";
import { Metadata } from "../../../api/explorer.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { searchMetadata } from "../../../redux/thunks/filemanager.ts";
import { FmIndexContext } from "../FmIndexContext.tsx";
import { TagChip } from "./FileTag.tsx";

export interface UploadingTagProps {
  disabled?: boolean;
  [key: string]: any;
}

const FileTagSummary = memo(({ sx, disabled, ...restProps }: UploadingTagProps) => {
  const fmIndex = useContext(FmIndexContext);
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const stopPropagation = useCallback((e: any) => {
    e.stopPropagation();
  }, []);
  const onClick = useCallback(
    (e: any) => {
      e.stopPropagation();
      dispatch(searchMetadata(fmIndex, Metadata.upload_session_id));
    },
    [dispatch, fmIndex],
  );
  return (
    <Stack direction={"row"} spacing={1} sx={{ ...sx }} {...restProps}>
      <TagChip
        onClick={disabled ? undefined : onClick}
        onMouseDown={stopPropagation}
        size="small"
        label={t("fileManager.uploading")}
      />
    </Stack>
  );
});

export default FileTagSummary;
