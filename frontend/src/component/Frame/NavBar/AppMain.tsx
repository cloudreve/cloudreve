import { Box, styled, useMediaQuery, useTheme } from "@mui/material";
import { useContext, useEffect, useMemo, useState } from "react";
import { Navigate, Outlet, useNavigation } from "react-router-dom";
import { CSSTransition, SwitchTransition } from "react-transition-group";
import { GroupPermission } from "../../../api/user.ts";
import { useAppSelector } from "../../../redux/hooks.ts";
import SessionManager from "../../../session";
import { GroupBS } from "../../../session/utils.ts";
import FacebookCircularProgress from "../../Common/CircularProgress.tsx";
import { PageVariant, PageVariantContext } from "../NavBarFrame.tsx";
import { DrawerHeaderContainer } from "./DrawerHeader.tsx";

const StyledLoadingContainer = styled(Box)(() => ({
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  width: "100%",
  height: "100%",
}));

export const PageLoading = () => {
  return (
    <StyledLoadingContainer>
      <FacebookCircularProgress />
    </StyledLoadingContainer>
  );
};

const AppMain = () => {
  const open = useAppSelector((state) => state.globalState.drawerOpen);
  const drawerWidth = useAppSelector((state) => state.globalState.drawerWidth);
  const [innerHeight, setInnerHeight] = useState(window.innerHeight);
  let navigation = useNavigation();
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const pageVariant = useContext(PageVariantContext);
  const isDashboard = pageVariant == PageVariant.dashboard;
  const user = SessionManager.currentLoginOrNull();
  const isAdmin = useMemo(() => {
    return GroupBS(user?.user).enabled(GroupPermission.is_admin);
  }, [user?.user?.group?.permission]);

  useEffect(() => {
    const handleResize = () => {
      setInnerHeight(window.innerHeight);
    };
    window.addEventListener("resize", handleResize);
    return () => window.removeEventListener("resize", handleResize);
  }, []);

  return (
    <Box
      sx={(theme) => ({
        flexGrow: 1,
        transition: theme.transitions.create("margin", {
          easing: theme.transitions.easing.sharp,
          duration: theme.transitions.duration.leavingScreen,
        }),
        marginRight: isMobile ? 0 : 2,
        marginLeft: isMobile ? 0 : `-${drawerWidth - 16}px`,
        ...(open && {
          transition: theme.transitions.create("margin", {
            easing: theme.transitions.easing.easeOut,
            duration: theme.transitions.duration.enteringScreen,
          }),
          marginLeft: 0,
        }),
        height: isMobile ? "100%" : window.innerHeight,
        minHeight: window.innerHeight,
        display: "flex",
        flexDirection: "column",
        width: "100%",
        overflow: "hidden",
      })}
      component={"main"}
    >
      <DrawerHeaderContainer />
      <SwitchTransition>
        <CSSTransition
          addEndListener={(node, done) => node.addEventListener("transitionend", done, false)}
          classNames="fade"
          key={navigation.state !== "idle" ? "loading" : "idle"}
        >
          {navigation.state !== "idle" ? (
            <PageLoading />
          ) : isDashboard && !isAdmin ? (
            <Navigate to={"/home"} />
          ) : (
            <Outlet />
          )}
        </CSSTransition>
      </SwitchTransition>
    </Box>
  );
};

export default AppMain;
