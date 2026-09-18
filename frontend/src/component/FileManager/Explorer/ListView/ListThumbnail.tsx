import { alpha, Box, Fade, Skeleton, styled } from "@mui/material";
import React, { memo, useCallback, useContext, useEffect, useState } from "react";
import { useInView } from "react-intersection-observer";
import { TransitionGroup } from "react-transition-group";
import { useAppDispatch, useAppSelector } from "../../../../redux/hooks.ts";
import { fileIconClicked, loadFileThumb } from "../../../../redux/thunks/file.ts";
import CheckmarkCircle from "../../../Icons/CheckmarkCircle.tsx";
import CheckUnchecked from "../../../Icons/CheckUnchecked.tsx";
import { FmIndexContext } from "../../FmIndexContext.tsx";
import FileIcon from "../FileIcon.tsx";
import { FmFile } from "../GridView/GridView.tsx";

const ThumbSquareSize = 48;

const ThumbSquare = styled(Box)(({ theme }) => ({
  position: "relative",
  width: ThumbSquareSize,
  height: ThumbSquareSize,
  borderRadius: 6,
  overflow: "hidden",
  flexShrink: 0,
  backgroundColor: theme.palette.mode === "light" ? theme.palette.grey[200] : theme.palette.grey[800],
  cursor: "pointer",
}));

const ThumbSquareImg = styled("img")<{ loaded: boolean }>(({ theme, loaded }) => ({
  display: "block",
  objectFit: "cover",
  width: "100%",
  height: "100%",
  opacity: loaded ? 1 : 0,
  transition: theme.transitions.create("opacity", {
    duration: theme.transitions.duration.short,
  }),
  userSelect: "none",
  WebkitUserDrag: "none",
  MozUserDrag: "none",
  msUserDrag: "none",
}));

const ThumbSquareLayer = styled(Box)(() => ({
  position: "absolute",
  inset: 0,
  width: "100%",
  height: "100%",
}));

const ThumbSquareFallback = styled(ThumbSquareLayer)(({ theme }) => ({
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  backgroundColor: theme.palette.background.default,
}));

const FileTypeBadge = styled(Box)(({ theme }) => ({
  position: "absolute",
  bottom: 2,
  right: 2,
  width: 20,
  height: 20,
  borderRadius: 4,
  backgroundColor: alpha(theme.palette.background.paper, 0.62),
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  pointerEvents: "none",
}));

const SelectOverlay = styled(Box)(({ theme }) => ({
  position: "absolute",
  top: 2,
  left: 2,
  width: 22,
  height: 22,
  borderRadius: "50%",
  backgroundColor: alpha(theme.palette.background.paper, 0.62),
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
}));

export interface ListThumbnailProps {
  file: FmFile;
  isSelected?: boolean;
  noThumb?: boolean;
  showLock?: boolean;
}

const ListThumbnail = memo(({ file, isSelected, noThumb, showLock }: ListThumbnailProps) => {
  const dispatch = useAppDispatch();
  const fmIndex = useContext(FmIndexContext);
  const hovered = useAppSelector((state) => state.fileManager[fmIndex].multiSelectHovered[file.path]);

  const skipLoad = !!noThumb || !!showLock;
  const { ref, inView } = useInView({
    triggerOnce: true,
    rootMargin: "200px 0px",
    skip: skipLoad,
  });

  // undefined: not loaded, null: no thumb
  const [thumbSrc, setThumbSrc] = useState<string | undefined | null>(skipLoad ? null : undefined);
  const [imageLoading, setImageLoading] = useState(true);

  useEffect(() => {
    if (!inView || skipLoad) {
      return;
    }
    let cancelled = false;
    dispatch(loadFileThumb(0, file)).then((src) => {
      if (!cancelled) setThumbSrc(src);
    });
    return () => {
      cancelled = true;
    };
  }, [inView, skipLoad, file, dispatch]);

  const onImgLoadError = useCallback(() => {
    setImageLoading(false);
    setThumbSrc(null);
  }, []);

  const onIconClick = useCallback(
    (e: React.MouseEvent<HTMLElement>) => {
      dispatch(fileIconClicked(fmIndex, file, e));
    },
    [dispatch, file, fmIndex],
  );

  const showSelectOverlay = isSelected || hovered;

  return (
    <ThumbSquare ref={ref} onClick={onIconClick}>
      <TransitionGroup component={null}>
        {thumbSrc && (
          <Fade key="image">
            <ThumbSquareLayer>
              <ThumbSquareImg
                loaded={!imageLoading}
                src={thumbSrc}
                draggable={false}
                onLoad={() => setImageLoading(false)}
                onError={onImgLoadError}
              />
            </ThumbSquareLayer>
          </Fade>
        )}
        {(thumbSrc === undefined || (thumbSrc && imageLoading)) && (
          <Fade key="loading">
            <ThumbSquareLayer>
              <Skeleton variant="rectangular" width="100%" height="100%" />
            </ThumbSquareLayer>
          </Fade>
        )}
        {thumbSrc === null && (
          <Fade key="icon">
            <ThumbSquareFallback>
              <FileIcon
                file={file}
                variant={"medium"}
                iconProps={{
                  sx: {
                    fontSize: 28,
                    height: 32,
                    width: 32,
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "center",
                  },
                }}
              />
            </ThumbSquareFallback>
          </Fade>
        )}
      </TransitionGroup>
      {thumbSrc && (
        <FileTypeBadge>
          <FileIcon
            file={file}
            variant={"small"}
            sx={{ p: 0, display: "flex", alignItems: "center", justifyContent: "center" }}
            iconProps={{
              sx: {
                fontSize: 14,
                height: 16,
                width: 16,
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
              },
            }}
          />
        </FileTypeBadge>
      )}
      {showSelectOverlay && (
        <SelectOverlay>
          {isSelected ? (
            <CheckmarkCircle color={"primary"} sx={{ width: 20, height: 20 }} />
          ) : (
            <CheckUnchecked color={"action"} sx={{ width: 18, height: 18 }} />
          )}
        </SelectOverlay>
      )}
    </ThumbSquare>
  );
});

export default ListThumbnail;
