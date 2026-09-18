import { Box, Button, ButtonGroup, ListItemText, Menu } from "@mui/material";
import React, { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { getFileEntityUrl } from "../../../api/api.ts";
import { closePhotopeaViewer, GeneralViewerState } from "../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { savePhotopea } from "../../../redux/thunks/viewer.ts";
import { fileExtension, getFileLinkedUri } from "../../../util";
import { SquareMenuItem } from "../../FileManager/ContextMenu/ContextMenu.tsx";
import useActionDisplayOpt, { canUpdate } from "../../FileManager/ContextMenu/useActionDisplayOpt.ts";
import CaretDown from "../../Icons/CaretDown.tsx";
import ViewerDialog, { ViewerLoading } from "../ViewerDialog.tsx";
import SaveAsNewFormat from "./SaveAsNewFormat.tsx";

const photopeiaOrigin = "https://www.photopea.com";
const photopeiaUrl =
  "https://www.photopea.com#%7B%22environment%22%3A%7B%22customIO%22%3A%7B%22save%22%3A%22app.echoToOE(%5C%22SAVE%5C%22)%3B%22%2C%22saveAsPSD%22%3A%22app.echoToOE(%5C%22SAVEPSD%5C%22)%3B%22%7D%7D%7D";
const saveCommand = "SAVE";
const savePSDCommand = "SAVEPSD";

const appendBuffer = function (buffer1: ArrayBuffer, buffer2: ArrayBuffer): ArrayBuffer {
  var tmp = new Uint8Array(buffer1.byteLength + buffer2.byteLength);
  tmp.set(new Uint8Array(buffer1), 0);
  tmp.set(new Uint8Array(buffer2), buffer1.byteLength);
  return tmp.buffer;
};

const saveOpt = {
  started: 1,
  saveAs: 2,
};

const Photopea = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const viewerState = useAppSelector((state) => state.globalState.photopeaViewer);

  const displayOpt = useActionDisplayOpt(viewerState?.file ? [viewerState?.file] : []);
  const [loading, setLoading] = useState(false);
  const [loaded, setLoaded] = useState(false);
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const [newFormatDialog, setNewFormatDialog] = useState(false);

  const entityUrl = useRef<string>("");
  const pp = useRef<HTMLIFrameElement | undefined>();
  const [src, setSrc] = useState<string | undefined>(undefined);
  const doneCount = useRef(0);
  const saveStarted = useRef<number>(0);
  const currentState = useRef<GeneralViewerState | undefined>(undefined);
  const buffer = useRef(new ArrayBuffer(0));
  const supportUpdate = useRef(false);

  useEffect(() => {
    window.addEventListener("message", eventHandler);
    return () => {
      window.removeEventListener("message", eventHandler);
    };
  }, []);

  useEffect(() => {
    if (!viewerState || !viewerState.open) {
      return;
    }

    supportUpdate.current = canUpdate(displayOpt);
    setLoaded(false);
    currentState.current = viewerState;
    buffer.current = new ArrayBuffer(0);
    doneCount.current = 0;
    saveStarted.current = 0;
    setSrc(undefined);
    dispatch(
      getFileEntityUrl({
        uris: [getFileLinkedUri(viewerState.file)],
        entity: viewerState.version,
      }),
    )
      .then((res) => {
        entityUrl.current = res.urls[0].url;
        setSrc(photopeiaUrl);
      })
      .catch(() => {
        onClose();
      });
  }, [viewerState]);

  const save = (ext?: string, newFile?: boolean) => {
    setAnchorEl(null);
    if (!pp.current || !supportUpdate.current) {
      return;
    }

    setLoading(true);
    if (!ext) {
      ext = fileExtension(currentState.current?.file.name ?? "") ?? "jpg";
      if (ext == "psd") {
        ext += ":true";
      }
    }

    pp.current.contentWindow?.postMessage(`app.activeDocument.saveToOE("${ext}")`, "*");
    saveStarted.current = newFile ? saveOpt.saveAs : saveOpt.started;
  };

  const eventHandler = (e: MessageEvent) => {
    if (e.origin != photopeiaOrigin) {
      return;
    }
    console.log(e);
    if (e.data == "done") {
      if (doneCount.current == 0) {
        pp.current?.contentWindow?.postMessage(`app.open("${entityUrl.current}","",false)`, "*");
      } else if (doneCount.current == 2) {
        pp.current?.contentWindow?.postMessage(
          `app.activeDocument.name="${currentState.current?.file.name.replace(/"/g, '\\"') ?? ""}"`,
          "*",
        );
        setLoaded(true);
      } else if (saveStarted.current > 0 && currentState.current) {
        dispatch(
          savePhotopea(
            buffer.current,
            currentState.current.file,
            currentState.current.version,
            saveStarted.current == saveOpt.saveAs,
          ),
        );
        setLoading(false);
      }
      doneCount.current++;
    } else if (e.data == saveCommand) {
      save();
    } else if (e.data == savePSDCommand) {
      save("psd:true", true);
    } else if (e.data instanceof ArrayBuffer && saveStarted.current) {
      buffer.current = appendBuffer(buffer.current, e.data);
    }
  };

  const onClose = useCallback(() => {
    dispatch(closePhotopeaViewer());
  }, [dispatch]);

  const onLoad = useCallback((e: React.SyntheticEvent<HTMLIFrameElement>) => (pp.current = e.currentTarget), []);

  const openMore = useCallback(
    (e: React.MouseEvent<any>) => {
      setAnchorEl(e.currentTarget);
    },
    [dispatch],
  );

  const openSaveAsOtherFormat = () => {
    setNewFormatDialog(true);
    setAnchorEl(null);
  };

  const onSaveAsNewFormat = (ext: string, quality?: number) => {
    if (quality) {
      ext += `:${quality}`;
    }

    save(ext, true);
  };

  return (
    <>
      <SaveAsNewFormat
        onSaveSubmit={onSaveAsNewFormat}
        open={newFormatDialog}
        onClose={() => setNewFormatDialog(false)}
      />
      <ViewerDialog
        file={viewerState?.file}
        loading={loading}
        readOnly={!supportUpdate.current}
        actions={
          supportUpdate.current ? (
            <ButtonGroup disabled={loading || !loaded} disableElevation variant="contained">
              <Button onClick={() => save()} variant={"contained"}>
                {t("fileManager.save")}
              </Button>
              <Button size="small" onClick={openMore}>
                <CaretDown sx={{ fontSize: "12px!important" }} />
              </Button>
            </ButtonGroup>
          ) : undefined
        }
        fullScreenToggle
        dialogProps={{
          open: !!(viewerState && viewerState.open),
          onClose: onClose,
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
          <SquareMenuItem dense onClick={() => save(undefined, true)}>
            <ListItemText>{t("modals.saveAs")}</ListItemText>
          </SquareMenuItem>
          <SquareMenuItem onClick={openSaveAsOtherFormat} dense>
            <ListItemText>{t("modals.saveAsOtherFormat")}</ListItemText>
          </SquareMenuItem>
        </Menu>
        {!src && <ViewerLoading />}
        {src && (
          <Box
            ref={pp}
            sx={{
              width: "100%",
              height: "100%",
              border: "none",
              minHeight: "calc(100vh - 200px)",
            }}
            component={"iframe"}
            title={"ms"}
            src={src}
            allowFullScreen
          />
        )}
      </ViewerDialog>
    </>
  );
};

export default Photopea;
