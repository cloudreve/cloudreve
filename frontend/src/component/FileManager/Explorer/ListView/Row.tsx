import { alpha, Box, Skeleton, styled } from "@mui/material";
import { memo, useCallback, useEffect } from "react";
import { useAppDispatch } from "../../../../redux/hooks.ts";
import { navigateReconcile } from "../../../../redux/thunks/filemanager.ts";
import { NoWrapTypography } from "../../../Common/StyledComponents.tsx";
import useActionDisplayOpt from "../../ContextMenu/useActionDisplayOpt.ts";
import { FileBlockProps } from "../Explorer.tsx";
import { useFileBlockState } from "../GridView/GridFile.tsx";
import Cell from "./Cell.tsx";

const RowContainer = styled(Box)<{
  selected: boolean;
  transparent?: boolean;
  isDropOver?: boolean;
  disabled?: boolean;
  taller?: boolean;
}>(({ theme, disabled, transparent, isDropOver, selected, taller }) => {
  let bgColor = "initial";
  let bgColorHover = theme.palette.action.hover;

  if (selected) {
    bgColor = alpha(theme.palette.primary.main, 0.18);
    bgColorHover = bgColor;
  }
  return {
    minHeight: taller ? "60px" : "36px",
    borderBottom: `1px solid ${theme.palette.divider}`,
    display: "flex",
    backgroundColor: bgColor,
    "&:hover": {
      backgroundColor: bgColorHover,
    },
    pointerEvents: disabled ? "none" : "auto",
    opacity: transparent || disabled ? 0.5 : 1,
    transition: "all 250ms cubic-bezier(0.4, 0, 0.2, 1) 0ms",
    transitionProperty: "background-color,opacity,box-shadow",
    boxShadow: isDropOver ? `inset 0 0 0 2px ${theme.palette.primary.light}` : "none",
  };
});

const Column = styled(Box)<{ w: number }>(({ theme, w }) => ({
  display: "flex",
  alignItems: "center",
  width: `${w}px`,
  padding: "0 10px",
}));

const Row = memo((props: FileBlockProps) => {
  const { file, columns, search, isDragging, isDropOver, showThumb } = props;
  const dispatch = useAppDispatch();

  const {
    fmIndex,
    isSelected,
    isLoadingIndicator,
    noThumb,
    uploading,
    ref,
    inView,
    fileTag,
    onClick,
    onDoubleClicked,
    hoverStateOff,
    hoverStateOn,
    onContextMenu,
    setRefFunc,
    disabled,
    fileDisabled,
    thumbWidth,
    thumbHeight,
  } = useFileBlockState(props);

  const targetDisplayOptions = useActionDisplayOpt([file]);

  useEffect(() => {
    if (!inView) {
      return;
    }

    if (isLoadingIndicator) {
      if (file.first) {
        dispatch(navigateReconcile(fmIndex, { next_page: true }));
      }
      return;
    }
  }, [inView]);

  const stopPop = useCallback((e: React.MouseEvent<HTMLElement>) => {
    e.stopPropagation();
  }, []);

  return (
    <RowContainer
      transparent={isDragging || fileDisabled}
      isDropOver={isDropOver && !isDragging}
      ref={setRefFunc}
      selected={!!isSelected}
      taller={showThumb}
      onMouseDown={stopPop}
      onClick={onClick}
      onDoubleClick={onDoubleClicked}
      onMouseEnter={hoverStateOn}
      onMouseLeave={hoverStateOff}
      onContextMenu={onContextMenu}
      disabled={disabled}
    >
      {columns?.map((column, index) => (
        <Column w={column.width ?? column.defaults.width} key={index}>
          <NoWrapTypography
            sx={{
              width: "100%",
            }}
            variant={"body2"}
          >
            {!file.placeholder && (
              <Cell
                actionDisplayOpt={targetDisplayOptions}
                isSelected={!!isSelected}
                search={search}
                column={column}
                file={file}
                uploading={uploading}
                fileTag={fileTag}
                noThumb={noThumb}
                showThumb={showThumb}
                thumbWidth={thumbWidth}
                thumbHeight={thumbHeight}
              />
            )}

            {file.placeholder && <Skeleton variant={"text"} width={0.5 * (column.width ?? column.defaults.width)} />}
          </NoWrapTypography>
        </Column>
      ))}
    </RowContainer>
  );
});

export default Row;
