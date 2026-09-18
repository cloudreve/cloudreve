import { Backdrop, useMediaQuery, useTheme } from "@mui/material";
import { grey } from "@mui/material/colors";
import { lazy, Suspense } from "react";
import { closeImageViewer } from "../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import FacebookCircularProgress from "../../Common/CircularProgress.tsx";

const Lightbox = lazy(() => import("./Lightbox.tsx"));
const ImageEditor = lazy(() => import("./ImageEditor.tsx"));

const Loading = (
  <Backdrop sx={{ color: "#fff", zIndex: (theme) => theme.zIndex.drawer + 1 }} open={true}>
    <FacebookCircularProgress fgColor={"#fff"} bgColor={grey[800]} />
  </Backdrop>
);

const ImageViewer = () => {
  const dispatch = useAppDispatch();
  const theme = useTheme();
  const viewer = useAppSelector((state) => state.globalState.imageViewer);
  const editorState = useAppSelector((state) => state.globalState.imageEditor);
  const sideBarOpen = useAppSelector((state) => state.globalState.sidebarOpen);
  const isTablet = useMediaQuery(theme.breakpoints.down("md"));

  const sidebarOpenOnTablet = isTablet && sideBarOpen;

  return (
    <div>
      <Suspense fallback={Loading}>
        {viewer && viewer.open && !sidebarOpenOnTablet && (
          <Lightbox onClose={() => dispatch(closeImageViewer())} viewer={viewer} />
        )}
      </Suspense>
      <Suspense fallback={Loading}>{editorState && editorState.open && <ImageEditor />}</Suspense>
    </div>
  );
};

export default ImageViewer;
