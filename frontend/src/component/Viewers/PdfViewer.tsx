import { Box, useTheme } from "@mui/material";
import i18next from "i18next";
import { useCallback, useEffect, useState } from "react";
import { getFileEntityUrl } from "../../api/api.ts";
import { closePdfViewer } from "../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../redux/hooks.ts";
import { getFileLinkedUri } from "../../util";
import ViewerDialog, { ViewerLoading } from "./ViewerDialog.tsx";

const viewerBase = "/pdfviewer.html";

const PdfViewer = () => {
  const dispatch = useAppDispatch();
  const theme = useTheme();
  const viewerState = useAppSelector((state) => state.globalState.pdfViewer);

  const [loading, setLoading] = useState(false);
  const [src, setSrc] = useState("");

  useEffect(() => {
    if (!viewerState || !viewerState.open) {
      return;
    }

    setSrc("");
    dispatch(
      getFileEntityUrl({
        uris: [getFileLinkedUri(viewerState.file)],
        entity: viewerState.version,
      }),
    )
      .then((res) => {
        const search = new URLSearchParams();
        search.set("file", res.urls[0].url);
        search.set("lng", i18next.language);
        search.set("darkMode", theme.palette.mode == "dark" ? "2" : "1");
        setSrc(`${viewerBase}?${search.toString()}`);
      })
      .catch(() => {
        onClose();
      });
  }, [viewerState]);

  const onClose = useCallback(() => {
    dispatch(closePdfViewer());
  }, [dispatch]);

  return (
    <>
      <ViewerDialog
        file={viewerState?.file}
        loading={loading}
        fullScreenToggle
        dialogProps={{
          open: !!(viewerState && viewerState.open),
          onClose: onClose,
          fullWidth: true,
          maxWidth: "lg",
        }}
      >
        {!src && <ViewerLoading />}
        {src && (
          <Box
            onLoad={() => setLoading(false)}
            src={src}
            sx={{
              width: "100%",
              height: loading ? 0 : "100%",
              border: "none",
              minHeight: loading ? 0 : "calc(100vh - 200px)",
            }}
            component={"iframe"}
          />
        )}
      </ViewerDialog>
    </>
  );
};

export default PdfViewer;
