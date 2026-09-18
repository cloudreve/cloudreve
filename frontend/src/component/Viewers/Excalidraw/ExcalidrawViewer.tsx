import { LoadingButton } from "@mui/lab";
import { Box, Button, ButtonGroup, Fade, ListItemText, Menu, useTheme } from "@mui/material";
import { TransitionProps } from "@mui/material/transitions";
import React, { lazy, Suspense, useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import i18next from "../../../i18n.ts";
import { closeExcalidrawViewer } from "../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { getEntityContent } from "../../../redux/thunks/file.ts";
import { saveExcalidraw } from "../../../redux/thunks/viewer.ts";
import { SquareMenuItem } from "../../FileManager/ContextMenu/ContextMenu.tsx";
import useActionDisplayOpt, { canUpdate } from "../../FileManager/ContextMenu/useActionDisplayOpt.ts";
import CaretDown from "../../Icons/CaretDown.tsx";
import ViewerDialog, { ViewerLoading } from "../ViewerDialog.tsx";

const Excalidraw = lazy(() => import("./Excalidraw.tsx"));

const Transition = React.forwardRef(function Transition(
  props: TransitionProps & {
    children: React.ReactElement<any, any>;
  },
  ref: React.Ref<unknown>,
) {
  return <Fade ref={ref} {...props} />;
});

const ExcalidrawViewer = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const theme = useTheme();
  const viewerState = useAppSelector((state) => state.globalState.excalidrawViewer);

  const displayOpt = useActionDisplayOpt(viewerState?.file ? [viewerState?.file] : []);
  const supportUpdate = canUpdate(displayOpt);
  const [loading, setLoading] = useState(false);
  const [value, setValue] = useState("");
  const [changedValue, setChangedValue] = useState("");
  const [loaded, setLoaded] = useState(false);
  const [saved, setSaved] = useState(true);
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const saveFunction = useRef(() => {});

  const loadContent = useCallback(() => {
    if (!viewerState || !viewerState.open) {
      return;
    }

    setLoaded(false);
    dispatch(getEntityContent(viewerState.file, viewerState.version))
      .then((res) => {
        const content = new TextDecoder().decode(res);
        setValue(content);
        setChangedValue(content);
        setLoaded(true);
      })
      .catch(() => {
        onClose();
      });
  }, [viewerState]);

  useEffect(() => {
    if (!viewerState || !viewerState.open) {
      return;
    }

    setSaved(true);
    loadContent();
  }, [viewerState?.open]);

  const onClose = useCallback(() => {
    dispatch(closeExcalidrawViewer());
  }, [dispatch]);

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
      dispatch(saveExcalidraw(changedValue, viewerState.file, viewerState.version, saveAs))
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
        onClose: onClose,
        fullWidth: true,
        maxWidth: "lg",
        TransitionComponent: Transition,
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
          <Excalidraw
            language={i18next.language}
            value={changedValue}
            initialValue={value}
            readOnly={!supportUpdate}
            darkMode={theme.palette.mode === "dark"}
            onChange={(v) => onChange(v as string)}
            onSaveShortcut={onSaveShortcut}
          />
        </Suspense>
      )}
    </ViewerDialog>
  );
};

export default ExcalidrawViewer;
