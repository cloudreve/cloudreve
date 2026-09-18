import { Backdrop, Box, ThemeProvider } from "@mui/material";
import { useSnackbar } from "notistack";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useGeneratedTheme } from "../../../App.tsx";
import { setSelected } from "../../../redux/fileManagerSlice.ts";
import { ImageViewerState } from "../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { onImageViewerIndexChange } from "../../../redux/thunks/viewer.ts";
import { fileExtension } from "../../../util";
import { FileManagerIndex } from "../../FileManager/FileManager.tsx";
import { usePaginationState } from "../../FileManager/Pagination/PaginationFooter.tsx";
import Sidebar from "../../FileManager/Sidebar/Sidebar.tsx";
import ImageOffOutlined from "../../Icons/ImageOffOutlined.tsx";
import { PhotoSlider } from "./react-photo-view";
import type { DataType } from "./react-photo-view/types.ts";

export interface LightboxProps {
  viewer?: ImageViewerState;
  onClose: () => void;
}

const Lightbox = ({ viewer, onClose }: LightboxProps) => {
  const theme = useGeneratedTheme(true, true);
  const container = useRef<HTMLElement | undefined>(undefined);
  const { enqueueSnackbar, closeSnackbar } = useSnackbar();
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const [open, setOpen] = useState(true);
  const [current, setCurrent] = useState(viewer?.index ?? 0);
  const indexChanged = useRef(false);
  const files = useAppSelector((state) => state.fileManager[FileManagerIndex.main].list?.files ?? []);
  const paginationState = usePaginationState(FileManagerIndex.main);

  const viewerFiles = useMemo(() => {
    if (!viewer) return [];
    let imagesCount = 0;
    const res: DataType[] = [];
    let updatedCurrent: number | undefined = undefined;
    if (viewer.version) {
      setCurrent(imagesCount);
      res.push({
        key: viewer.file.id + viewer.version,
        file: viewer.file,
        version: viewer.version,
      });
      imagesCount++;
    } else if (viewer.index == -1) {
      setCurrent(imagesCount);
      res.push({
        key: viewer.file.id,
        file: viewer.file,
      });
      imagesCount++;
    } else {
      files.forEach((f) => {
        if (!indexChanged.current && f.path == viewer.file.path) {
          updatedCurrent = imagesCount;
          setCurrent(imagesCount);
          res.push({
            key: f.id,
            file: f,
          });
          imagesCount++;
        } else if (viewer.exts.includes(fileExtension(f.name) ?? "")) {
          imagesCount++;
          res.push({
            key: f.id,
            file: f,
          });
        }
      });
    }

    if (paginationState.moreItems) {
      res.push({
        loadMorePlaceholder: true,
        key: `${paginationState.currentPage} - ${paginationState.nextToken}`,
      });
    }

    if (paginationState.useEndlessLoading && (updatedCurrent ?? current) >= res.length) {
      setCurrent(res.length - 1);
    }

    if (paginationState.usePagination && indexChanged.current) {
      setCurrent(0);
      if (res.length == 0 || res[0].loadMorePlaceholder) {
        setOpen(false);
        enqueueSnackbar(t("application:fileManager.noMoreImages"), {
          variant: "warning",
          preventDuplicate: true,
        });
      }
    }

    indexChanged.current = true;
    return res;
  }, [viewer?.file, viewer?.index, viewer?.version, viewer?.exts, files, paginationState.moreItems]);

  const onIndexChange = useCallback(
    (index: number) => {
      setCurrent(index);
    },
    [setCurrent, dispatch, paginationState, viewerFiles],
  );

  useEffect(() => {
    const file = viewerFiles[current]?.file;
    if (file) {
      dispatch(onImageViewerIndexChange(file));
    }
  }, [viewerFiles[current]?.file]);

  useEffect(() => {
    if (viewerFiles.length == 0) {
      beforeClose();
    }
  }, [viewerFiles.length]);

  const beforeClose = useCallback(() => {
    const file = viewerFiles[current]?.file;
    if (viewer?.index != -1 && file) {
      dispatch(setSelected({ index: FileManagerIndex.main, value: [file] }));
    }
    onClose();
  }, [onClose, viewer?.index, viewerFiles, current, dispatch]);

  return (
    <ThemeProvider theme={theme}>
      <PhotoSlider
        moreFiles={paginationState.moreItems}
        images={viewerFiles}
        onClose={() => setOpen(false)}
        afterClose={beforeClose}
        visible={open}
        index={current}
        onIndexChange={onIndexChange}
        brokenElement={
          <Box
            sx={{
              color: "white",
              textAlign: "center",
            }}
          >
            <ImageOffOutlined fontSize={"large"} />
          </Box>
        }
        portalContainer={container.current}
      />
      <Backdrop
        ref={container}
        sx={{
          bgcolor: "rgb(0 0 0 / 0%)",
          zIndex: (theme) => theme.zIndex.drawer + 1,
        }}
        open={open}
      >
        <Box>
          <Sidebar inPhotoViewer />
        </Box>
      </Backdrop>
    </ThemeProvider>
  );
};

export default Lightbox;
