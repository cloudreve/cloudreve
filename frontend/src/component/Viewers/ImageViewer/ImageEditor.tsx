import { Backdrop, Box, useTheme } from "@mui/material";
import i18next from "i18next";
import { useSnackbar } from "notistack";
import React, { useCallback, useEffect, useState } from "react";
import FilerobotImageEditor from "react-filerobot-image-editor";
import { useTranslation } from "react-i18next";
import { getFileEntityUrl } from "../../../api/api.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { saveImage, switchToImageViewer } from "../../../redux/thunks/viewer.ts";
import { fileExtension, getFileLinkedUri } from "../../../util";
import "./editor.css";

export const editorSupportedExt = ["jpg", "jpeg", "png", "webp"];

const ImageEditor = () => {
  const theme = useTheme();
  const { t, i18n } = useTranslation();
  const [nsLoaded, setNsLoaded] = useState(false);

  useEffect(() => {
    i18next.loadNamespaces(["image_editor"]).then(() => {
      setNsLoaded(true);
    });
  }, []);

  const editorState = useAppSelector((state) => state.globalState.imageEditor);
  const [imageSrc, setImageSrc] = React.useState<string | undefined>(undefined);
  useEffect(() => {
    if (!editorState?.open) {
      setImageSrc(undefined);
    } else {
      dispatch(
        getFileEntityUrl({
          no_cache: true,
          uris: [getFileLinkedUri(editorState.file)],
          entity: editorState.version,
        }),
      )
        .then((res) => {
          setImageSrc(res.urls[0].url);
        })
        .catch(() => {
          dispatch(switchToImageViewer());
        });
    }
  }, [editorState]);

  const dispatch = useAppDispatch();
  const { enqueueSnackbar } = useSnackbar();

  const save = useCallback(
    async (name: string, data?: string) => {
      if (!data || !editorState?.file) {
        return;
      }

      await dispatch(saveImage(name, data, editorState.file, editorState.version));
      dispatch(switchToImageViewer());
    },
    [dispatch, editorState],
  );

  return (
    <>
      <Backdrop
        sx={{
          zIndex: (theme) => theme.zIndex.drawer + 1,
        }}
        open={true}
      >
        <Box sx={{ width: "100%", height: "100%" }}>
          {editorState && imageSrc && nsLoaded && (
            <FilerobotImageEditor
              translations={i18n.getResourceBundle(i18n.language, "image_editor")}
              useBackendTranslations={false}
              source={imageSrc}
              onSave={async (editedImageObject, _designState) => {
                await save(editedImageObject.name, editedImageObject.imageBase64);
              }}
              defaultSavedImageName={editorState.file.name}
              // @ts-ignore
              defaultSavedImageType={fileExtension(editorState.file.name) ?? "png"}
              onClose={() => dispatch(switchToImageViewer())}
              disableSaveIfNoChanges={true}
              previewPixelRatio={window.devicePixelRatio}
              savingPixelRatio={4}
            />
          )}
        </Box>
      </Backdrop>
    </>
  );
};

export default ImageEditor;
