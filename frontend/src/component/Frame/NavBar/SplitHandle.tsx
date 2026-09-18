import { Box, Fade } from "@mui/material";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { useEffect, useRef, useState } from "react";
import { setDrawerWidth } from "../../../redux/globalStateSlice.ts";
import SessionManager, { UserSettings } from "../../../session";

export interface SplitHandleProps {}

const minDrawerWidth = 236;

const SplitHandle = (_props: SplitHandleProps) => {
  const dispatch = useAppDispatch();
  const [moving, setMoving] = useState(false);
  const [cursor, setCursor] = useState(0);
  const finalWidth = useRef(0);

  const drawerWidth = useAppSelector((state) => state.globalState.drawerWidth);
  const drawerOpen = useAppSelector((state) => state.globalState.drawerOpen);

  useEffect(() => {
    setCursor(drawerWidth - 4);
    finalWidth.current = drawerWidth - 4;
  }, []);

  const handler = () => {
    setMoving(true);
    document.body.style.userSelect = "none";
    function onMouseMove(e: MouseEvent) {
      e.preventDefault();
      const newWidth = e.clientX - document.body.offsetLeft;
      const cappedWidth = Math.max(Math.min(newWidth, window.innerWidth / 2), minDrawerWidth);
      setCursor(cappedWidth);
      finalWidth.current = cappedWidth;
    }
    function onMouseUp() {
      document.body.removeEventListener("mousemove", onMouseMove);
      setMoving(false);
      dispatch(setDrawerWidth(finalWidth.current + 4));
      SessionManager.set(UserSettings.DrawerWidth, finalWidth.current + 4);
      document.body.style.userSelect = "initial";
    }

    document.body.addEventListener("mousemove", onMouseMove);
    document.body.addEventListener("mouseup", onMouseUp, { once: true });
  };

  return (
    <>
      {drawerOpen && (
        <Box
          onMouseDown={handler}
          sx={{
            cursor: "ew-resize",
            height: "100%",
            position: "fixed",
            width: 8,
            left: cursor,
            zIndex: (theme) => theme.zIndex.drawer + 2,
          }}
        />
      )}
      <Fade in={moving} unmountOnExit>
        <Box
          sx={{
            height: "100%",
            position: "fixed",
            width: 8,
            left: cursor,
            bgcolor: "divider",
            zIndex: (theme) => theme.zIndex.drawer + 1,
          }}
        />
      </Fade>
    </>
  );
};

export default SplitHandle;
