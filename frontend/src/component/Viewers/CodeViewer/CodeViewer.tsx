import { LoadingButton } from "@mui/lab";
import { Box, Button, ButtonGroup, IconButton, ListItemIcon, ListItemText, Menu, useTheme } from "@mui/material";
import React, { lazy, Suspense, useCallback, useEffect, useRef, useState } from "react";
import { useHotkeys } from "react-hotkeys-hook";
import { useTranslation } from "react-i18next";
import { closeCodeViewer } from "../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { confirmOperation } from "../../../redux/thunks/dialog.ts";
import { getEntityContent } from "../../../redux/thunks/file.ts";
import { saveCode } from "../../../redux/thunks/viewer.ts";
import { fileExtension } from "../../../util";
import { CascadingSubmenu } from "../../FileManager/ContextMenu/CascadingMenu.tsx";
import { DenseDivider, SquareMenuItem } from "../../FileManager/ContextMenu/ContextMenu.tsx";
import useActionDisplayOpt, { canUpdate } from "../../FileManager/ContextMenu/useActionDisplayOpt.ts";
import CaretDown from "../../Icons/CaretDown.tsx";
import Checkmark from "../../Icons/Checkmark.tsx";
import Setting from "../../Icons/Setting.tsx";
import ViewerDialog, { ViewerLoading } from "../ViewerDialog.tsx";

const MonacoEditor = lazy(() => import("./MonacoEditor.tsx"));

export const codePreviewSuffix: {
  [key: string]: string;
} = {
  md: "markdown",
  json: "json",
  php: "php",
  py: "python",
  bat: "bat",
  cpp: "cpp",
  c: "cpp",
  h: "cpp",
  cs: "csharp",
  css: "css",
  dockerfile: "dockerfile",
  go: "go",
  html: "html",
  ini: "ini",
  java: "java",
  js: "javascript",
  jsx: "javascript",
  less: "less",
  lua: "lua",
  sh: "shell",
  sql: "sql",
  xml: "xml",
  yaml: "yaml",
};

const allCharsets = [
  "utf-8",
  "ibm866",
  "iso-8859-2",
  "iso-8859-3",
  "iso-8859-4",
  "iso-8859-5",
  "iso-8859-6",
  "iso-8859-7",
  "iso-8859-8",
  "iso-8859-8i",
  "iso-8859-10",
  "iso-8859-13",
  "iso-8859-14",
  "iso-8859-15",
  "iso-8859-16",
  "koi8-r",
  "koi8-u",
  "macintosh",
  "windows-874",
  "windows-1250",
  "windows-1251",
  "windows-1252",
  "windows-1253",
  "windows-1254",
  "windows-1255",
  "windows-1256",
  "windows-1257",
  "windows-1258",
  "x-mac-cyrillic",
  "gbk",
  "gb18030",
  "hz-gb-2312",
  "big5",
  "euc-jp",
  "iso-2022-jp",
  "shift-jis",
  "euc-kr",
  "iso-2022-kr",
  "utf-16be",
  "utf-16le",
];

const allLng = Array.from(new Set(Object.values(codePreviewSuffix))).sort();

const CodeViewer = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const theme = useTheme();
  const viewerState = useAppSelector((state) => state.globalState.codeViewer);

  const displayOpt = useActionDisplayOpt(viewerState?.file ? [viewerState?.file] : []);
  const supportUpdate = canUpdate(displayOpt);
  const [loading, setLoading] = useState(false);
  const [value, setValue] = useState("");
  const [loaded, setLoaded] = useState(false);
  const [saved, setSaved] = useState(true);
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const [optionAnchorEl, setOptionAnchorEl] = useState<null | HTMLElement>(null);
  const [language, setLng] = useState<string | null>(null);
  const [wordWrap, setWordWrap] = useState<"off" | "on" | "wordWrapColumn" | "bounded">("off");
  const saveFunction = useRef<() => void>(() => {});

  const closeViewer = useCallback(() => {
    dispatch(closeCodeViewer());
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

  const loadContent = useCallback(
    (charset?: string) => {
      if (!viewerState || !viewerState.open) {
        return;
      }

      setLoaded(false);
      setOptionAnchorEl(null);
      dispatch(getEntityContent(viewerState.file, viewerState.version))
        .then((res) => {
          setValue(new TextDecoder(charset).decode(res));
          setLoaded(true);
        })
        .catch(() => {
          closeViewer();
        });
    },
    [viewerState, closeViewer],
  );

  useEffect(() => {
    if (!viewerState || !viewerState.open) {
      return;
    }

    setLng(null);
    setSaved(true);
    setLng(codePreviewSuffix[fileExtension(viewerState.file.name) ?? ""] ?? "");
    loadContent();
  }, [viewerState?.open]);

  const openMore = useCallback(
    (e: React.MouseEvent<any>) => {
      setAnchorEl(e.currentTarget);
    },
    [dispatch],
  );

  const openOption = useCallback(
    (e: React.MouseEvent<any>) => {
      setOptionAnchorEl(e.currentTarget);
    },
    [dispatch],
  );

  const toggleWordWrap = useCallback(() => {
    setOptionAnchorEl(null);
    setWordWrap((prev) => (prev == "off" ? "on" : "off"));
  }, []);

  const onSave = useCallback(
    (saveAs?: boolean) => {
      if (!viewerState?.file) {
        return;
      }

      setLoading(true);
      dispatch(saveCode(value, viewerState.file, viewerState.version, saveAs))
        .then(() => {
          setSaved(true);
        })
        .finally(() => {
          setLoading(false);
        });
    },
    [value, viewerState],
  );

  const onChange = useCallback((v: string) => {
    setValue(v);
    setSaved(false);
  }, []);

  useEffect(() => {
    saveFunction.current = () => {
      if (!saved && supportUpdate) {
        onSave(false);
      }
    };
  }, [saved, supportUpdate, onSave]);

  useHotkeys(
    ["Control+s", "Meta+s"],
    () => {
      saveFunction.current();
    },
    { preventDefault: true },
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
          <IconButton onClick={openOption}>
            <Setting fontSize={"small"} />
          </IconButton>
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
      <Menu
        anchorEl={optionAnchorEl}
        open={Boolean(optionAnchorEl)}
        onClose={() => setOptionAnchorEl(null)}
        slotProps={{
          paper: {
            sx: {
              minWidth: 200,
            },
          },
        }}
      >
        <CascadingSubmenu popupId={"lng"} title={t("application:fileManager.charset")}>
          {allCharsets.map((charset) => (
            <SquareMenuItem key={charset} onClick={() => loadContent(charset)}>
              <ListItemText>{charset}</ListItemText>
            </SquareMenuItem>
          ))}
        </CascadingSubmenu>
        <CascadingSubmenu popupId={"lng"} title={t("application:fileManager.textType")}>
          {allLng.map((l) => (
            <SquareMenuItem key={l} onClick={() => setLng(l)}>
              <ListItemText>{l}</ListItemText>
              {l == language && (
                <ListItemIcon>
                  <Checkmark />
                </ListItemIcon>
              )}
            </SquareMenuItem>
          ))}
        </CascadingSubmenu>
        <DenseDivider />
        <SquareMenuItem onClick={toggleWordWrap} dense>
          <ListItemText>{t("fileManager.wordWrap")}</ListItemText>
          {wordWrap === "on" && (
            <ListItemIcon sx={{ minWidth: "0!important" }}>
              <Checkmark />
            </ListItemIcon>
          )}
        </SquareMenuItem>
      </Menu>
      {!loaded && <ViewerLoading />}
      {loaded && (
        <Suspense fallback={<ViewerLoading />}>
          <Box
            sx={{
              width: "100%",
              height: "100%",
              minHeight: "calc(100vh - 200px)",
            }}
          >
            <MonacoEditor
              onSave={saveFunction}
              theme={theme.palette.mode === "dark" ? "vs-dark" : "vs"}
              options={{
                readOnly: !supportUpdate,
                wordWrap: wordWrap,
                automaticLayout: true,
              }}
              value={value}
              language={language ?? ""}
              onChange={(v) => onChange(v as string)}
            />
          </Box>
        </Suspense>
      )}
    </ViewerDialog>
  );
};

export default CodeViewer;
