import { useTranslation } from "react-i18next";
import { useAppDispatch, useAppSelector } from "../../redux/hooks.ts";
import ViewerDialog, { ViewerLoading } from "./ViewerDialog.tsx";
import React, { useCallback, useEffect, useState } from "react";
import { closeCustomViewer } from "../../redux/globalStateSlice.ts";
import { Box, useTheme } from "@mui/material";

const CustomViewer = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const theme = useTheme();
  const viewerState = useAppSelector((state) => state.globalState.customViewer);

  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!viewerState || !viewerState.open) {
      return;
    }

    setLoading(true);
  }, [viewerState]);

  const onClose = useCallback(() => {
    dispatch(closeCustomViewer());
  }, [dispatch]);

  return (
    <>
      <ViewerDialog
        file={viewerState?.file}
        fullScreenToggle
        dialogProps={{
          open: !!(viewerState && viewerState.open),
          onClose: onClose,
          fullWidth: true,
          maxWidth: "lg",
        }}
      >
        {loading && <ViewerLoading />}
        {viewerState && (
          <Box
            onLoad={() => setLoading(false)}
            src={viewerState.url}
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

export default CustomViewer;
