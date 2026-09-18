import { LoadingButton } from "@mui/lab";
import { Box, Button, ButtonGroup, ListItemText, Menu, useTheme } from "@mui/material";
import React, { lazy, Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { closeMarkdownViewer } from "../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { confirmOperation } from "../../../redux/thunks/dialog.ts";
import { getEntityContent } from "../../../redux/thunks/file.ts";
import {
  markdownImageAutocompleteSuggestions,
  markdownImagePreviewHandler,
  saveMarkdown,
  uploadMarkdownImage,
} from "../../../redux/thunks/viewer.ts";
import { SquareMenuItem } from "../../FileManager/ContextMenu/ContextMenu.tsx";
import useActionDisplayOpt, { canUpdate } from "../../FileManager/ContextMenu/useActionDisplayOpt.ts";
import CaretDown from "../../Icons/CaretDown.tsx";
import ViewerDialog, { ViewerLoading } from "../ViewerDialog.tsx";

const MarkdownEditor = lazy(() => import("./Editor.tsx"));

const MarkdownViewer = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const theme = useTheme();
  const viewerState = useAppSelector((state) => state.globalState.markdownViewer);

  const displayOpt = useActionDisplayOpt(viewerState?.file ? [viewerState?.file] : []);
  const supportUpdate = canUpdate(displayOpt);
  const [loading, setLoading] = useState(false);
  const [value, setValue] = useState("");
  const [changedValue, setChangedValue] = useState("");
  const [loaded, setLoaded] = useState(false);
  const [saved, setSaved] = useState(true);
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const [optionAnchorEl, setOptionAnchorEl] = useState<null | HTMLElement>(null);
  const saveFunction = useRef(() => {});

  const closeViewer = useCallback(() => {
    dispatch(closeMarkdownViewer());
  }, [dispatch]);

  const handleDialogClose = useCallback(
    (_: React.SyntheticEvent | object, _reason?: string) => {
      if (!saved && supportUpdate) {
        dispatch(confirmOperation(t("application:modals.discardUnsavedConfirm")))
          .then(() => {
            closeViewer();
          })
          .catch(() => {});

        return;
      }

      closeViewer();
    },
    [closeViewer, saved, supportUpdate, t, dispatch],
  );

  const loadContent = useCallback(() => {
    if (!viewerState || !viewerState.open) {
      return;
    }

    setLoaded(false);
    setOptionAnchorEl(null);
    dispatch(getEntityContent(viewerState.file, viewerState.version))
      .then((res) => {
        const content = new TextDecoder().decode(res);
        setValue(content);
        setChangedValue(content);
        setLoaded(true);
      })
      .catch(() => {
        closeViewer();
      });
  }, [viewerState, closeViewer]);

  useEffect(() => {
    if (!viewerState || !viewerState.open) {
      return;
    }

    setSaved(true);
    loadContent();
  }, [viewerState?.open]);

  const imageAutocompleteSuggestions = useMemo(() => {
    if (!viewerState?.open) {
      return null;
    }
    return dispatch(markdownImageAutocompleteSuggestions());
  }, [viewerState?.open]);

  const openMore = useCallback(
    (e: React.MouseEvent<any>) => {
      setAnchorEl(e.currentTarget);
    },
    [dispatch],
  );

  const onSave = useCallback(
    (saveAs?: boolean) => {
      if (!viewerState?.file) {
        return;
      }

      setLoading(true);
      dispatch(saveMarkdown(changedValue, viewerState.file, viewerState.version, saveAs))
        .then(() => {
          setSaved(true);
        })
        .finally(() => {
          setLoading(false);
        });
    },
    [changedValue, viewerState],
  );

  const onChange = useCallback((v: string) => {
    setChangedValue(v);
    setSaved(false);
  }, []);

  const onSaveShortcut = useCallback(() => {
    if (!saved && supportUpdate) {
      onSave(false);
    }
  }, [saved, supportUpdate, onSave]);

  useEffect(() => {
    saveFunction.current = () => {
      if (!saved && supportUpdate) {
        onSave(false);
      }
    };
  }, [saved, supportUpdate, onSave]);

  const imagePreviewHandler = useCallback(
    (imageSource: string) => {
      return dispatch(markdownImagePreviewHandler(imageSource, viewerState?.file?.path ?? ""));
    },
    [dispatch, viewerState?.file?.path],
  );

  const onImageUpload = useCallback(
    async (file: File): Promise<string> => {
      return dispatch(uploadMarkdownImage(file));
    },
    [dispatch],
  );

  return (
    <ViewerDialog
      file={viewerState?.file}
      loading={loading}
      readOnly={!supportUpdate}
      actions={
        <Box sx={{ display: "flex", gap: 1 }}>
          {supportUpdate && (
            <ButtonGroup disabled={loading || !loaded || saved} disableElevation variant="contained">
              <LoadingButton loading={loading} variant={"contained"} onClick={() => onSave(false)}>
                <span>{t("fileManager.save")}</span>
              </LoadingButton>
              <Button size="small" onClick={openMore}>
                <CaretDown sx={{ fontSize: "12px!important" }} />
              </Button>
            </ButtonGroup>
          )}
        </Box>
      }
      fullScreenToggle
      dialogProps={{
        open: !!(viewerState && viewerState.open),
        onClose: handleDialogClose,
        fullWidth: true,
        maxWidth: "lg",
      }}
    >
      <Menu
        anchorEl={anchorEl}
        open={Boolean(anchorEl)}
        onClose={() => setAnchorEl(null)}
        slotProps={{
          paper: {
            sx: {
              minWidth: 150,
            },
          },
        }}
      >
        <SquareMenuItem onClick={() => onSave(true)} dense>
          <ListItemText>{t("modals.saveAs")}</ListItemText>
        </SquareMenuItem>
      </Menu>
      {!loaded && <ViewerLoading />}
      {loaded && (
        <Suspense fallback={<ViewerLoading />}>
          <MarkdownEditor
            value={changedValue}
            readOnly={!supportUpdate}
            darkMode={theme.palette.mode === "dark"}
            initialValue={value}
            onChange={(v) => onChange(v as string)}
            onSaveShortcut={onSaveShortcut}
            imagePreviewHandler={imagePreviewHandler}
            imageAutocompleteSuggestions={imageAutocompleteSuggestions}
            imageUploadHandler={onImageUpload}
          />
        </Suspense>
      )}
    </ViewerDialog>
  );
};

export default MarkdownViewer;
