import { PaginationItem, PaginationItemProps, styled } from "@mui/material";
import { NoOpDropUri, useFileDrag } from "../Dnd/DndWrappedFile.tsx";
import { useCallback, useEffect, useRef } from "react";
import { mergeRefs } from "../../../util";

let timeOut: ReturnType<typeof setTimeout> | undefined = undefined;

const StyledPaginationItem = styled(PaginationItem)<{ isDropOver?: boolean }>(({ theme, isDropOver }) => ({
  transition: "all 250ms cubic-bezier(0.4, 0, 0.2, 1) 0ms !important",
  transitionProperty: "background-color,opacity,box-shadow",
  boxShadow: isDropOver ? `inset 0 0 0 2px ${theme.palette.primary.light}` : "none",
}));

const CustomPaginationItem = (props: PaginationItemProps) => {
  const [drag, drop, isOver, isDragging] = useFileDrag({
    dropUri: props.type !== "start-ellipsis" && props.type !== "end-ellipsis" ? NoOpDropUri : undefined,
  });
  const buttonRef = useRef<HTMLElement>();

  useEffect(() => {
    if (
      isOver &&
      props.onClick &&
      props.type !== "start-ellipsis" &&
      props.type !== "end-ellipsis" &&
      buttonRef.current &&
      !props.selected
    ) {
      if (timeOut) {
        clearTimeout(timeOut);
      }
      timeOut = setTimeout(() => buttonRef.current?.click(), 500);
    }
  }, [isOver]);

  const mergedRef = useCallback(
    (val: any) => {
      mergeRefs(drop, buttonRef)(val);
    },
    [drop, buttonRef],
  );

  return <StyledPaginationItem isDropOver={isOver} ref={mergedRef} {...props} />;
};

export default CustomPaginationItem;
