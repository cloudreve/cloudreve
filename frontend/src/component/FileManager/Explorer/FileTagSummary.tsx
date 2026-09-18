import * as React from "react";
import { memo, useCallback, useMemo } from "react";
import { Box, Popover, Stack, useMediaQuery, useTheme } from "@mui/material";
import { bindHover, bindPopover } from "material-ui-popup-state";
import { bindTrigger, usePopupState } from "material-ui-popup-state/hooks";
import HoverPopover from "material-ui-popup-state/HoverPopover";
import FileTag, { TagChip } from "./FileTag.tsx";

export interface FileTagSummaryProps {
  tags: { key: string; value: string }[];
  max?: number;
  [key: string]: any;
}

const FileTagSummary = memo(({ tags, sx, max = 1, ...restProps }: FileTagSummaryProps) => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("md"));
  const popupState = usePopupState({
    variant: "popover",
    popupId: "demoMenu",
  });

  const { open, ...rest } = bindPopover(popupState);
  const stopPropagation = useCallback((e: any) => e.stopPropagation(), []);
  const [shown, hidden] = useMemo(() => {
    if (tags.length <= max) {
      return [tags, []];
    }
    return [tags.slice(0, max), tags.slice(max)];
  }, [tags, max]);

  const { onClick, ...triggerProps } = bindTrigger(popupState);
  const onMobileClick = (e: React.MouseEvent<HTMLElement>) => {
    e.stopPropagation();
    onClick(e);
  };

  const PopoverComponent = isMobile ? Popover : HoverPopover;

  return (
    <Stack direction={"row"} spacing={1} sx={{ ...sx }} {...restProps}>
      {shown.map((tag) => (
        <FileTag tagColor={tag.value == "" ? undefined : tag.value} label={tag.key} key={tag.key} />
      ))}
      {hidden.length > 0 && (
        <TagChip
          size="small"
          variant={"outlined"}
          label={`+${hidden.length}`}
          {...(isMobile
            ? {
                onClick: onMobileClick,
                ...triggerProps,
              }
            : bindHover(popupState))}
        />
      )}

      {open && (
        <PopoverComponent
          onMouseDown={stopPropagation}
          onMouseUp={stopPropagation}
          onClick={stopPropagation}
          open={open}
          {...rest}
          anchorOrigin={{
            vertical: "bottom",
            horizontal: "center",
          }}
          transformOrigin={{
            vertical: "top",
            horizontal: "center",
          }}
        >
          <Box sx={{ p: 1, maxWidth: "300px" }}>
            {hidden.map((tag, i) => (
              <FileTag
                tagColor={tag.value == "" ? undefined : tag.value}
                label={tag.key}
                spacing={i === hidden.length - 1 ? undefined : 1}
                key={tag.key}
              />
            ))}
          </Box>
        </PopoverComponent>
      )}
    </Stack>
  );
});

export default FileTagSummary;
