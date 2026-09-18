import { AppBarProps as MuiAppBarProps } from "@mui/material/AppBar";
import { AppBar, Box, Collapse, IconButton, Stack, Toolbar, Tooltip, useMediaQuery, useTheme } from "@mui/material";
import { Menu } from "@mui/icons-material";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { setDrawerOpen, setMobileDrawerOpen } from "../../../redux/globalStateSlice.ts";
import NewButton from "../../FileManager/NewButton.tsx";
import UserAction from "./UserAction.tsx";
import Setting from "../../Icons/Setting.tsx";
import DarkThemeSwitcher from "./DarkThemeSwitcher.tsx";
import NavBarMainActions from "./NavBarMainActions.tsx";
import MusicPlayer from "../../Viewers/MusicPlayer/MusicPlayer.tsx";
import { TaskListIconButton } from "../../Uploader/TaskListIconButton.tsx";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import SessionManager from "../../../session";
import { useContext, useState } from "react";
import { DrawerPopover } from "./AppDrawer.tsx";
import { PageVariant, PageVariantContext } from "../NavBarFrame.tsx";

interface AppBarProps extends MuiAppBarProps {
  open?: boolean;
}

const TopAppBar = () => {
  const dispatch = useAppDispatch();
  const theme = useTheme();
  const pageVariant = useContext(PageVariantContext);
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const isMainPage = pageVariant == PageVariant.default;
  const { t } = useTranslation();
  const navigate = useNavigate();
  const open = useAppSelector((state) => state.globalState.drawerOpen);
  const mobileDrawerOpen = useAppSelector((state) => state.globalState.mobileDrawerOpen);
  const drawerWidth = useAppSelector((state) => state.globalState.drawerWidth);
  const musicPlayer = useAppSelector((state) => state.globalState.musicPlayer);
  const [mobileMenuAnchor, setMobileMenuAnchor] = useState<null | HTMLElement>(null);

  const appBarBg = theme.palette.mode === "light" ? theme.palette.grey[100] : theme.palette.grey[900];
  const isLogin = !!SessionManager.currentLoginOrNull();

  const onMobileMenuClicked = (e: React.MouseEvent<HTMLElement>) => {
    setMobileMenuAnchor(e.currentTarget);
    dispatch(setMobileDrawerOpen(true));
  };

  const onCloseMobileMenu = () => {
    dispatch(setMobileDrawerOpen(false));
  };

  // @ts-ignore
  return (
    <AppBar
      elevation={0}
      enableColorOnDark
      sx={(theme) => ({
        transition: theme.transitions.create(["margin", "width"], {
          easing: theme.transitions.easing.sharp,
          duration: theme.transitions.duration.leavingScreen,
        }),
        backgroundColor: appBarBg,
        color: theme.palette.getContrastText(appBarBg),
        ...(open &&
          !isMobile && {
            width: `calc(100% - ${drawerWidth}px)`,
            marginLeft: `${drawerWidth}px`,
            transition: theme.transitions.create(["margin", "width"], {
              easing: theme.transitions.easing.easeOut,
              duration: theme.transitions.duration.enteringScreen,
            }),
          }),
      })}
      position="fixed"
    >
      <Toolbar
        sx={{
          "&.MuiToolbar-root.MuiToolbar-gutters": {
            paddingLeft: open && !isMobile ? theme.spacing(0) : theme.spacing(isMobile ? 2 : 3),
            transition: theme.transitions.create("padding", {
              easing: theme.transitions.easing.easeInOut,
              duration: theme.transitions.duration.standard,
            }),
          },
        }}
      >
        <Collapse orientation={"horizontal"} in={!open || isMobile}>
          <IconButton
            color="inherit"
            aria-label="open drawer"
            // @ts-ignore
            onClick={isMobile ? onMobileMenuClicked : () => dispatch(setDrawerOpen(true))}
            edge="start"
            sx={{
              mr: isMobile ? 1 : 2,
              ml: isMobile ? -1 : -1.5,
            }}
          >
            <Menu />
          </IconButton>
        </Collapse>
        {isMobile && (
          <>
            <DrawerPopover open={!!mobileDrawerOpen} anchorEl={mobileMenuAnchor} onClose={onCloseMobileMenu} />
          </>
        )}
        {!isMobile && isMainPage && (
          <Stack direction={"row"} spacing={1} sx={{ height: 42 }}>
            <NewButton />
            <NavBarMainActions />
          </Stack>
        )}
        <Box sx={{ flexGrow: 1 }} />
        <Stack
          direction={"row"}
          sx={{
            alignItems: "center",
          }}
          spacing={isMobile ? 1 : 0}
        >
          {!isMobile && <TaskListIconButton />}
          {musicPlayer && <MusicPlayer />}
          {!isMobile ? (
            <>
              <DarkThemeSwitcher />
              {isLogin && (
                <Tooltip title={t("navbar.setting")}>
                  <IconButton size="large" onClick={() => navigate("/settings")}>
                    <Setting />
                  </IconButton>
                </Tooltip>
              )}
              <UserAction />
            </>
          ) : (
            <>
              {isMainPage && <NavBarMainActions />}
              {isMainPage && <NewButton />}
              <UserAction />
            </>
          )}
        </Stack>
      </Toolbar>
    </AppBar>
  );
};

export default TopAppBar;
