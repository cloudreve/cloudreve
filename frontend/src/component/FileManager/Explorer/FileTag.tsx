import { Chip, ChipProps, darken, styled, Tooltip, useTheme } from "@mui/material";
import { useCallback, useContext } from "react";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { Metadata } from "../../../api/explorer.ts";
import { searchMetadata } from "../../../redux/thunks/filemanager.ts";
import { FmIndexContext } from "../FmIndexContext.tsx";

export const TagChip = styled(Chip)<{ defaultStyle?: boolean }>(({ defaultStyle }) => {
  const base = {
    "& .MuiChip-deleteIcon": {},
  };
  if (!defaultStyle) return { ...base, height: 18, minWidth: 32 };
  return base;
});

export interface FileTagProps extends ChipProps {
  tagColor?: string;
  defaultStyle?: boolean;
  spacing?: number;
  openInNewTab?: boolean;
  disableClick?: boolean;
}

const FileTag = ({ disableClick, tagColor, sx, label, defaultStyle, spacing, openInNewTab, ...rest }: FileTagProps) => {
  const theme = useTheme();
  const fmIndex = useContext(FmIndexContext);
  const dispatch = useAppDispatch();
  const root = useAppSelector((state) => state.fileManager[fmIndex].path_root);
  const stopPropagation = useCallback(
    (e: any) => {
      if (!disableClick) e.stopPropagation();
    },
    [disableClick],
  );
  const onClick = useCallback(
    (e: any) => {
      if (disableClick) {
        return;
      }
      e.stopPropagation();
      dispatch(searchMetadata(fmIndex, Metadata.tag_prefix + label, tagColor, openInNewTab));
    },
    [root, dispatch, fmIndex, disableClick],
  );

  const hackColor =
    !!tagColor && theme.palette.getContrastText(tagColor) != theme.palette.text.primary ? "error" : undefined;
  return (
    <Tooltip title={label}>
      <TagChip
        defaultStyle={defaultStyle}
        sx={[
          !!tagColor && {
            backgroundColor: tagColor,
            color: (theme) => theme.palette.getContrastText(tagColor),
            "&:hover": {
              backgroundColor: darken(tagColor, 0.1),
            },
          },
          spacing !== undefined && { mr: spacing },
        ]}
        onClick={onClick}
        onMouseDown={stopPropagation}
        size="small"
        label={label}
        color={hackColor}
        {...rest}
      />
    </Tooltip>
  );
};

export default FileTag;
