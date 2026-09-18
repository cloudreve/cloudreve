import { Box, useMediaQuery, useTheme } from "@mui/material";
import { createContext, useEffect } from "react";
import { DndProvider } from "react-dnd";
import { HTML5Backend } from "react-dnd-html5-backend";
import { useLocation } from "react-router-dom";
import { setMobileDrawerOpen } from "../../redux/globalStateSlice.ts";
import { useAppDispatch } from "../../redux/hooks.ts";
import ContextMenu from "../FileManager/ContextMenu/ContextMenu.tsx";
import Dialogs from "../FileManager/Dialogs/Dialogs.tsx";
import DragLayer from "../FileManager/Dnd/DragLayer.tsx";
import { FileManagerIndex } from "../FileManager/FileManager.tsx";
import SearchPopup from "../FileManager/Search/SearchPopup.tsx";
import Uploader from "../Uploader/Uploader.tsx";
import AppDrawer from "./NavBar/AppDrawer.tsx";
import Main from "./NavBar/AppMain.tsx";
import SplitHandle from "./NavBar/SplitHandle.tsx";
import TopAppBar from "./NavBar/TopAppBar.tsx";

export enum PageVariant {
  default,
  dashboard,
}

export interface NavBarFrameProps {
  variant?: PageVariant;
}

export const PageVariantContext = createContext(PageVariant.default);

export const AutoNavbarFrame = () => {
  const path = useLocation().pathname;
  return <NavBarFrame variant={path.startsWith("/admin") ? PageVariant.dashboard : PageVariant.default} />;
};

const NavBarFrame = ({ variant }: NavBarFrameProps) => {
  const theme = useTheme();
  const dispatch = useAppDispatch();
  const isTouch = useMediaQuery("(pointer: coarse)");
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const location = useLocation();

  useEffect(() => {
    if (isMobile) {
      dispatch(setMobileDrawerOpen(false));
    }
  }, [location]);
  return (
    <PageVariantContext.Provider value={variant ?? PageVariant.default}>
      <Box
        sx={{
          bgcolor: (theme) => (theme.palette.mode === "light" ? theme.palette.grey[100] : theme.palette.grey[900]),
          display: "flex",
        }}
      >
        <DndProvider backend={HTML5Backend}>
          {!isMobile && variant != PageVariant.dashboard && !isTouch && <DragLayer />}
          {!isMobile && !isTouch && <SplitHandle />}
          <TopAppBar />
          {!isMobile && <AppDrawer />}
          {variant != PageVariant.dashboard && <ContextMenu fmIndex={FileManagerIndex.main} />}
          <Dialogs />
          <Uploader />
          {variant != PageVariant.dashboard && <SearchPopup />}
          <Main />
        </DndProvider>
      </Box>
    </PageVariantContext.Provider>
  );
};

export default NavBarFrame;
