import { Box, Stack, useMediaQuery, useTheme } from "@mui/material";
import { useEffect } from "react";
import { useHotkeys } from "react-hotkeys-hook";
import useNavigation from "../../hooks/useNavigation.tsx";
import { clearSelected } from "../../redux/fileManagerSlice.ts";
import { resetDialogs } from "../../redux/globalStateSlice.ts";
import { useAppDispatch } from "../../redux/hooks.ts";
import { resetFm, selectAll, shortCutDelete } from "../../redux/thunks/filemanager.ts";
import ImageViewer from "../Viewers/ImageViewer/ImageViewer.tsx";
import Explorer from "./Explorer/Explorer.tsx";
import { FmIndexContext } from "./FmIndexContext.tsx";
import PaginationFooter from "./Pagination/PaginationFooter.tsx";
import { ReadMe } from "./ReadMe/ReadMe.tsx";
import Sidebar from "./Sidebar/Sidebar.tsx";
import SidebarDialog from "./Sidebar/SidebarDialog.tsx";
import NavHeader from "./TopBar/NavHeader.tsx";

export const FileManagerIndex = {
  main: 0,
  selector: 1,
};

export interface FileManagerProps {
  index?: number;
  initialPath?: string;
  skipRender?: boolean;
}

export const FileManager = ({ index = 0, initialPath, skipRender }: FileManagerProps) => {
  const dispatch = useAppDispatch();
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const isTablet = useMediaQuery(theme.breakpoints.down("md"));

  useNavigation(index, initialPath);

  useEffect(() => {
    if (index == FileManagerIndex.main) {
      dispatch(resetDialogs());
      return () => {
        dispatch(resetFm(index));
      };
    }
  }, []);

  const selectAllRef = useHotkeys<HTMLElement>(
    ["Control+a", "Meta+a"],
    () => {
      dispatch(selectAll(index));
    },
    { enabled: index == FileManagerIndex.main, preventDefault: true },
  );

  const delRef = useHotkeys<HTMLElement>(
    ["meta+backspace", "delete"],
    () => {
      dispatch(shortCutDelete(index));
    },
    { enabled: index == FileManagerIndex.main, preventDefault: true },
  );

  const escRef = useHotkeys<HTMLElement>(
    "esc",
    () => {
      dispatch(clearSelected({ index, value: {} }));
    },
    { enabled: index == FileManagerIndex.main, preventDefault: true },
  );

  if (skipRender) {
    return null;
  }

  return (
    <FmIndexContext.Provider value={index}>
      <Stack
        onClick={(e) => {
          e.currentTarget.focus();
        }}
        ref={(ref) => {
          selectAllRef(ref);
          delRef(ref);
          escRef(ref);
        }}
        direction={"column"}
        sx={{
          flexGrow: 1,
          mb: index == FileManagerIndex.main && !isMobile ? 1 : 0,
          overflow: "auto",
          "&:focus": {
            outline: "none",
          },
        }}
        tabIndex={0}
        spacing={1}
      >
        <NavHeader />
        <Box sx={{ display: "flex", flexGrow: 1, overflowY: "auto" }}>
          <Explorer />
          {index == FileManagerIndex.main && (isTablet ? <SidebarDialog /> : <Sidebar />)}
          {index == FileManagerIndex.main && <ReadMe />}
        </Box>
        <PaginationFooter />
      </Stack>
      {index == FileManagerIndex.main && <ImageViewer />}
    </FmIndexContext.Provider>
  );
};
