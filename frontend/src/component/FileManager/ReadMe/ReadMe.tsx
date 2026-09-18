import { useMediaQuery, useTheme } from "@mui/material";
import { useContext, useEffect } from "react";
import { closeShareReadme } from "../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { detectReadMe } from "../../../redux/thunks/share.ts";
import { FmIndexContext } from "../FmIndexContext.tsx";
import ReadMeDialog from "./ReadMeDialog.tsx";
import ReadMeSideBar from "./ReadMeSideBar.tsx";

export const ReadMe = () => {
  const fmIndex = useContext(FmIndexContext);
  const dispatch = useAppDispatch();
  const detect = useAppSelector((state) => state.globalState.shareReadmeDetect);
  const theme = useTheme();
  const isTablet = useMediaQuery(theme.breakpoints.down("md"));

  useEffect(() => {
    if (detect) {
      dispatch(detectReadMe(fmIndex, isTablet));
    }
  }, [detect, dispatch]);

  useEffect(() => {
    if (detect === 0) {
      setTimeout(() => {
        dispatch(closeShareReadme());
      }, 500);
    }
  }, [detect]);

  if (isTablet) {
    return <ReadMeDialog />;
  }

  return <ReadMeSideBar />;
};
