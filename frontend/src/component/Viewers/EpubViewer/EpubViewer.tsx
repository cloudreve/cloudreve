import { Box, useTheme } from "@mui/material";
import React, { Suspense, useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { getFileEntityUrl } from "../../../api/api.ts";
import { closeEpubViewer } from "../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import SessionManager, { UserSettings } from "../../../session";
import { getFileLinkedUri } from "../../../util";
import ViewerDialog, { ViewerLoading } from "../ViewerDialog.tsx";

const Epub = React.lazy(() => import("./Epub.tsx"));

const EpubViewer = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const theme = useTheme();
  const viewerState = useAppSelector((state) => state.globalState.epubViewer);

  const [loading, setLoading] = useState(false);
  const [src, setSrc] = useState("");
  const [currentLocation, setLocation] = useState<string | null>(null);
  const locationChanged = useCallback(
    (epubcifi: string) => {
      setLocation(epubcifi);
      if (viewerState?.file) {
        SessionManager.set(`${UserSettings.BookLocationPrefix}_${viewerState.file.id}`, epubcifi);
      }
    },
    [viewerState?.file],
  );

  useEffect(() => {
    if (!viewerState || !viewerState.open) {
      setLocation(null);
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
        setSrc(res.urls[0].url);
        const location = SessionManager.get(`${UserSettings.BookLocationPrefix}_${viewerState.file.id}`);
        if (location) {
          setLocation(location);
        }
      })
      .catch(() => {
        onClose();
      });
  }, [viewerState]);

  const onClose = useCallback(() => {
    dispatch(closeEpubViewer());
  }, [dispatch]);

  return (
    <>
      <ViewerDialog
        file={viewerState?.file}
        loading={loading}
        dialogProps={{
          open: !!(viewerState && viewerState.open),
          onClose: onClose,
          fullWidth: true,
          fullScreen: true,
          maxWidth: "lg",
        }}
      >
        {!src && <ViewerLoading />}
        {src && (
          <Suspense fallback={<ViewerLoading />}>
            <Box
              sx={{
                width: "100%",
                height: "100%",
                minHeight: "calc(100vh - 200px)",
              }}
            >
              <Epub
                loadingView={<ViewerLoading />}
                location={currentLocation}
                locationChanged={locationChanged}
                epubInitOptions={{
                  openAs: "epub",
                }}
                showToc={true}
                url={src}
              />
            </Box>
          </Suspense>
        )}
      </ViewerDialog>
    </>
  );
};

export default EpubViewer;
