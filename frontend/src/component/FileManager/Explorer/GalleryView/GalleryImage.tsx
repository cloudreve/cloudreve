import { CheckCircle } from "@mui/icons-material";
import { Box, Fade, IconButton, ImageListItem, ImageListItemBar, lighten, styled } from "@mui/material";
import React, { memo, useCallback, useEffect, useState } from "react";
import { TransitionGroup } from "react-transition-group";
import { FileType, Metadata } from "../../../../api/explorer.ts";
import { useAppDispatch } from "../../../../redux/hooks.ts";
import { fileIconClicked, loadFileThumb } from "../../../../redux/thunks/file.ts";
import { navigateReconcile } from "../../../../redux/thunks/filemanager.ts";
import CheckUnchecked from "../../../Icons/CheckUnchecked.tsx";
import { FileBlockProps } from "../Explorer.tsx";
import FileIcon from "../FileIcon.tsx";
import {
  LargeIconContainer,
  ThumbBox,
  ThumbBoxContainer,
  ThumbLoadingPlaceholder,
  useFileBlockState,
} from "../GridView/GridFile.tsx";

const StyledImageListItem = styled(ImageListItem)<{
  transparent?: boolean;
  disabled?: boolean;
  isDropOver?: boolean;
}>(({ transparent, isDropOver, disabled, theme }) => {
  return {
    opacity: transparent || disabled ? 0.5 : 1,
    pointerEvents: disabled ? "none" : "auto",
    cursor: "pointer",
    boxShadow: isDropOver ? `0 0 0 2px ${theme.palette.primary.light}` : "none",
    transition: theme.transitions.create(["height", "width", "opacity", "box-shadow"]),
  };
});

const GalleryImage = memo((props: FileBlockProps) => {
  const { file, columns, search, isDragging, isDropOver } = props;
  const dispatch = useAppDispatch();

  const {
    fmIndex,
    isSelected,
    isLoadingIndicator,
    noThumb,
    uploading,
    ref,
    inView,
    showLock,
    fileTag,
    onClick,
    onDoubleClicked,
    hoverStateOff,
    hoverStateOn,
    onContextMenu,
    setRefFunc,
    disabled,
    fileDisabled,
  } = useFileBlockState(props);

  const [hovered, setHovered] = useState(false);

  // undefined: not loaded, null: no thumb
  const [thumbSrc, setThumbSrc] = useState<string | undefined | null>(noThumb ? null : undefined);
  const [imageLoading, setImageLoading] = useState(true);

  const tryLoadThumbSrc = useCallback(async () => {
    const thumbSrc = await dispatch(loadFileThumb(0, file));
    setThumbSrc(thumbSrc);
  }, [dispatch, file, setThumbSrc, setImageLoading]);

  const onImgLoadError = useCallback(() => {
    setImageLoading(false);
    setThumbSrc(null);
  }, [setImageLoading, setThumbSrc]);

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

    if (file.type == FileType.folder) {
      return;
    }

    if ((file.metadata && file.metadata[Metadata.thumbDisabled] !== undefined) || showLock) {
      // No thumb available
      setThumbSrc(null);
      return;
    }

    tryLoadThumbSrc();
  }, [inView]);

  const onIconClick = useCallback(
    (e: React.MouseEvent<HTMLElement>) => {
      e.stopPropagation();
      return dispatch(fileIconClicked(fmIndex, file, e));
    },
    [file, dispatch],
  );

  return (
    <StyledImageListItem
      onClick={file.type == FileType.folder ? onClick : onDoubleClicked}
      transparent={isDragging || fileDisabled}
      isDropOver={isDropOver && !isDragging}
      disabled={disabled}
      onContextMenu={onContextMenu}
      ref={setRefFunc}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      <TransitionGroup style={{ height: "100%" }}>
        {thumbSrc && (
          <Fade key={"image"}>
            <Box>
              <ThumbBoxContainer
                sx={{
                  p: isSelected ? "10%" : 0,
                  backgroundColor: (theme) => (isSelected ? lighten(theme.palette.primary.light, 0.85) : "initial"),
                  transition: (theme) =>
                    theme.transitions.create(["padding"], {
                      duration: theme.transitions.duration.shortest,
                    }),
                }}
              >
                <ThumbBox
                  sx={{
                    borderRadius: isSelected ? 1 : 0,
                  }}
                  loaded={!imageLoading}
                  src={thumbSrc}
                  onLoad={() => setImageLoading(false)}
                  onError={onImgLoadError}
                />
              </ThumbBoxContainer>
            </Box>
          </Fade>
        )}
        {(thumbSrc === undefined || (thumbSrc && imageLoading)) && (
          <Fade key={"loading"}>
            <ThumbLoadingPlaceholder
              sx={{
                borderRadius: 0,
              }}
              ref={isLoadingIndicator ? undefined : ref}
              variant={"rectangular"}
              height={"100%"}
            />
          </Fade>
        )}
        {thumbSrc === null && (
          <Fade key={"icon"}>
            <LargeIconContainer>
              <FileIcon
                variant={"largeMobile"}
                iconProps={{
                  sx: {
                    fontSize: "48px",
                    height: "64px",
                    width: "64px",
                  },
                }}
                file={file}
                loading={isLoadingIndicator}
              />
            </LargeIconContainer>
          </Fade>
        )}
      </TransitionGroup>
      <Fade in={!isLoadingIndicator && (hovered || !!isSelected)}>
        <ImageListItemBar
          sx={{
            background: "linear-gradient(to bottom, rgba(0,0,0,0.7) 0%, " + "rgba(0,0,0,0.3) 50%, rgba(0,0,0,0) 100%)",
          }}
          position="top"
          actionIcon={
            <IconButton onClick={onIconClick} size={"small"} sx={{ color: "white", mb: 1 }}>
              <TransitionGroup
                style={{
                  width: 20,
                  height: 20,
                }}
              >
                {!isSelected && (
                  <Fade>
                    <Box sx={{ position: "absolute" }}>
                      <CheckUnchecked fontSize={"small"} />
                    </Box>
                  </Fade>
                )}
                {isSelected && (
                  <Fade>
                    <Box sx={{ position: "absolute" }}>
                      <CheckCircle fontSize={"small"} />
                    </Box>
                  </Fade>
                )}
              </TransitionGroup>
            </IconButton>
          }
          actionPosition="left"
        />
      </Fade>
    </StyledImageListItem>
  );
});

export default GalleryImage;
