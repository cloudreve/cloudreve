import { Box, useMediaQuery, useTheme } from "@mui/material";
import React, { RefCallback, useCallback, useContext, useEffect, useMemo } from "react";
import { CSSTransition, SwitchTransition } from "react-transition-group";
import { useAreaSelection } from "../../../hooks/areaSelection.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { ConfigLoadState } from "../../../redux/siteConfigSlice.ts";
import { openEmptyContextMenu } from "../../../redux/thunks/filemanager.ts";
import { loadSiteConfig } from "../../../redux/thunks/site.ts";
import CircularProgress from "../../Common/CircularProgress.tsx";
import "../../Common/FadeTransition.css";
import { RadiusFrame } from "../../Frame/RadiusFrame.tsx";
import ExplorerError from "./ExplorerError.tsx";
import GridView, { FmFile } from "./GridView/GridView.tsx";

import { Layouts } from "../../../redux/fileManagerSlice.ts";
import { SearchParam } from "../../../util/uri.ts";
import { FileManagerIndex } from "../FileManager.tsx";
import { FmIndexContext } from "../FmIndexContext.tsx";
import EmptyFileList, { SearchLimitReached } from "./EmptyFileList.tsx";
import GalleryView from "./GalleryView/GalleryView.tsx";
import { ListViewColumn } from "./ListView/Column.tsx";
import ListView from "./ListView/ListView.tsx";
import SingleFileView from "./SingleFileView.tsx";

export const ExplorerPage = {
  Error: 1,
  Loading: 2,
  GridView: 0,
  SingleFileView: 3,
  Empty: 4,
  ListView: 5,
  GalleryView: 6,
};

export interface FileBlockProps {
  showThumb?: boolean;
  file: FmFile;
  isDragging?: boolean;
  isDropOver?: boolean;
  dragRef?: RefCallback<any>;
  index?: number;
  search?: SearchParam;
  columns?: ListViewColumn[];
  boxHeight?: number;
}

const Explorer = () => {
  const dispatch = useAppDispatch();
  const theme = useTheme();
  const isTouch = useMediaQuery("(pointer: coarse)");
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const fmIndex = useContext(FmIndexContext);
  const loading = useAppSelector((state) => state.fileManager[fmIndex].loading);
  const error = useAppSelector((state) => state.fileManager[fmIndex].error);
  const showError = useAppSelector((state) => state.fileManager[fmIndex].showError);
  const singleFileView = useAppSelector((state) => state.fileManager[fmIndex].list?.single_file_view);
  const explorerConfigLoading = useAppSelector((state) => state.siteConfig.explorer.loaded);
  const files = useAppSelector((state) => state.fileManager[fmIndex].list?.files);
  const recursion_limit_reached = useAppSelector((state) => state.fileManager[fmIndex].list?.recursion_limit_reached);
  const layout = useAppSelector((state) => state.fileManager[fmIndex].layout);

  const selectContainerRef = React.useRef<HTMLElement | null>(null);

  useEffect(() => {
    dispatch(loadSiteConfig("explorer"));
  }, []);

  const index = useMemo(() => {
    if (showError) {
      return ExplorerPage.Error;
    } else if (loading || explorerConfigLoading == ConfigLoadState.NotLoaded) {
      return ExplorerPage.Loading;
    } else {
      if (files?.length === 0) {
        return ExplorerPage.Empty;
      }

      if (singleFileView && fmIndex == FileManagerIndex.main) {
        return ExplorerPage.SingleFileView;
      }

      switch (layout) {
        case Layouts.grid:
          return ExplorerPage.GridView;
        case Layouts.list:
          return ExplorerPage.ListView;
        case Layouts.gallery:
          return ExplorerPage.GalleryView;
        default:
          return ExplorerPage.GridView;
      }
    }
  }, [loading, showError, explorerConfigLoading, singleFileView, fmIndex, files?.length, layout]);

  const enableAreaSelection = index == ExplorerPage.GridView;

  const [handleMouseDown, handleMouseUp, handleMouseMove] = useAreaSelection(
    selectContainerRef,
    fmIndex,
    enableAreaSelection,
  );

  const onContextMenu = useCallback(
    (e: React.MouseEvent<HTMLElement>) => {
      if (index == ExplorerPage.Error || index == ExplorerPage.Loading) return;
      dispatch(openEmptyContextMenu(fmIndex, e));
    },
    [dispatch, index],
  );

  return (
    <RadiusFrame
      withBorder={!isMobile}
      square={isMobile}
      sx={{ flexGrow: 1, overflow: "auto" }}
      ref={selectContainerRef}
      onContextMenu={onContextMenu}
      onMouseDown={isMobile || isTouch ? undefined : handleMouseDown}
      onMouseUp={isMobile || isTouch ? undefined : handleMouseUp}
      onMouseMove={isMobile || isTouch ? undefined : handleMouseMove}
    >
      <SwitchTransition>
        <CSSTransition
          timeout={500}
          addEndListener={(node, done) => node.addEventListener("transitionend", done, false)}
          classNames="fade"
          key={index}
        >
          <Box sx={{ height: "100%" }}>
            {index == ExplorerPage.Error && <ExplorerError error={error} />}
            {index == ExplorerPage.Loading && (
              <Box
                sx={{
                  height: "100%",
                  display: "flex",
                  justifyContent: "center",
                  alignItems: "center",
                }}
              >
                <CircularProgress />
              </Box>
            )}
            {index == ExplorerPage.GridView && <GridView />}
            {index == ExplorerPage.SingleFileView && <SingleFileView />}
            {index == ExplorerPage.Empty && <EmptyFileList />}
            {index == ExplorerPage.ListView && <ListView />}
            {index == ExplorerPage.GalleryView && <GalleryView />}
            {recursion_limit_reached && (index == ExplorerPage.GridView || index == ExplorerPage.GalleryView) && (
              <Box sx={{ px: 2, pb: 1 }}>
                <SearchLimitReached />
              </Box>
            )}
          </Box>
        </CSSTransition>
      </SwitchTransition>
    </RadiusFrame>
  );
};

export default Explorer;
