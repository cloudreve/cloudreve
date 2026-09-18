import { Suspense, useCallback, useEffect, useState } from "react";
import { enqueueSnackbar } from "notistack";
import React from "react";
import { useTranslation } from "react-i18next";
import { getFileEntityUrl } from "../../../api/api.ts";
import { closeModelViewer } from "../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { getFileLinkedUri } from "../../../util";
import { DefaultCloseAction } from "../../Common/Snackbar/snackbar.tsx";
import ViewerDialog, { ViewerLoading } from "../ViewerDialog.tsx";

const ModelViewerCanvas = React.lazy(() => import("./ModelViewerCanvas.tsx"));

const ModelViewer = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const viewerState = useAppSelector((state) => state.globalState.modelViewer);

  const [loading, setLoading] = useState(false);
  const [src, setSrc] = useState("");

  useEffect(() => {
    if (!viewerState || !viewerState.open) {
      return;
    }

    setSrc("");
    setLoading(true);
    dispatch(
      getFileEntityUrl({
        uris: [getFileLinkedUri(viewerState.file)],
        entity: viewerState.version,
      }),
    )
      .then((res) => setSrc(res.urls[0].url))
      .catch(() => onClose())
      .finally(() => setLoading(false));
  }, [viewerState]);

  const onClose = useCallback(() => {
    dispatch(closeModelViewer());
  }, [dispatch]);

  const onError = useCallback(() => {
    enqueueSnackbar({
      message: t("fileManager.modelLoadFailed"),
      variant: "error",
      action: DefaultCloseAction,
    });
    onClose();
  }, [onClose, t]);

  return (
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
          <ModelViewerCanvas src={src} fileName={viewerState?.file.name ?? ""} onError={onError} />
        </Suspense>
      )}
    </ViewerDialog>
  );
};

export default ModelViewer;
