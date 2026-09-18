import { useTranslation } from "react-i18next";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import ViewerDialog, { ViewerLoading } from "../ViewerDialog.tsx";
import React, { useCallback, useEffect, useRef, useState } from "react";
import { closeDrawIOViewer } from "../../../redux/globalStateSlice.ts";
import { Box, ListItemText, Menu, useTheme } from "@mui/material";
import useActionDisplayOpt, { canUpdate } from "../../FileManager/ContextMenu/useActionDisplayOpt.ts";
import { SquareMenuItem } from "../../FileManager/ContextMenu/ContextMenu.tsx";
import { generateIframeSrc, handleRemoteInvoke } from "./drawio.ts";
import { getEntityContent } from "../../../redux/thunks/file.ts";
import { saveDrawIO } from "../../../redux/thunks/viewer.ts";
import dayjs from "dayjs";
import { formatLocalTime } from "../../../util/datetime.ts";

const DrawIOViewer = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const theme = useTheme();
  const viewerState = useAppSelector((state) => state.globalState.drawIOViewer);
  const instanceID = useAppSelector((state) => state.siteConfig.basic.config.instance_id);

  const displayOpt = useActionDisplayOpt(viewerState?.file ? [viewerState?.file] : []);
  const supportUpdate = useRef(false);
  const [loading, setLoading] = useState(false);
  const [src, setSrc] = useState("");
  const [loaded, setLoaded] = useState(false);
  const pp = useRef<HTMLIFrameElement | undefined>();
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);

  useEffect(() => {
    if (!viewerState) {
      return;
    }

    if (!viewerState.open) {
      setSrc("");
      return;
    }

    setLoaded(false);
    supportUpdate.current = canUpdate(displayOpt) && !viewerState?.version;
    pp.current = undefined;
    const src = generateIframeSrc(
      viewerState.host,
      viewerState.file,
      !supportUpdate.current,
      theme.palette.mode == "dark",
    );
    setSrc(src);
    window.addEventListener("message", handlePostMessage, false);
    return () => {
      window.removeEventListener("message", handlePostMessage, false);
    };
  }, [viewerState?.open]);

  const handlePostMessage = async (e: MessageEvent) => {
    console.log("Received PostMessage from " + e.origin, e.data);
    let msg;
    try {
      msg = JSON.parse(e.data);
    } catch (e) {
      return;
    }

    if (!viewerState?.file) {
      return;
    }

    switch (msg.event) {
      case "exit":
        onClose();
        break;
      case "configure":
        pp.current?.contentWindow?.postMessage(
          // TODO: use userv config
          JSON.stringify({ action: "configure", config: {} }),
          "*",
        );
        break;
      case "remoteInvoke":
        pp.current?.contentWindow &&
          handleRemoteInvoke(pp.current?.contentWindow, msg, dispatch, viewerState, supportUpdate.current, instanceID);
        break;
      case "save":
        try {
          dispatch(saveDrawIO(msg.xml, viewerState.file, false));
          pp.current?.contentWindow?.postMessage(
            JSON.stringify({
              action: "status",
              message: t("fileManager.saveSuccess", {
                time: formatLocalTime(dayjs()),
              }),
              modified: false,
            }),
            "*",
          );
        } catch (e) {}
        break;

      case "init":
        try {
          const content = await dispatch(getEntityContent(viewerState.file, viewerState.version));
          const contentStr = new TextDecoder().decode(content);
          pp.current?.contentWindow?.postMessage(
            JSON.stringify({
              action: "load",
              autosave: supportUpdate.current,
              title: viewerState.file.name,
              xml: contentStr,
              desc: {
                xml: contentStr,
                id: viewerState.file.id,
                size: content.byteLength,
                etag: viewerState.file.primary_entity,
                writeable: supportUpdate.current,
                name: viewerState.file.name,
                versionEnabled: true,
                ver: 2,
                instanceId: instanceID ?? "",
              },
              disableAutoSave: !supportUpdate.current,
            }),
            "*",
          );
          if (supportUpdate.current) {
            pp.current?.contentWindow?.postMessage(JSON.stringify({ action: "remoteInvokeReady" }), "*");
          }
        } catch (e) {
          onClose();
        }
        break;
    }
  };

  const onClose = useCallback(() => {
    dispatch(closeDrawIOViewer());
  }, [dispatch]);

  const openMore = useCallback(
    (e: React.MouseEvent<any>) => {
      setAnchorEl(e.currentTarget);
    },
    [dispatch],
  );

  const handleIframeOnload = useCallback((e: React.SyntheticEvent<HTMLIFrameElement>) => {
    setLoaded(true);
    pp.current = e.currentTarget;
  }, []);

  return (
    <ViewerDialog
      file={viewerState?.file}
      loading={loading}
      readOnly={!supportUpdate.current}
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
        <SquareMenuItem dense>
          <ListItemText>{t("modals.saveAs")}</ListItemText>
        </SquareMenuItem>
      </Menu>
      {(!loaded || !src) && <ViewerLoading />}
      {src && (
        <Box
          onLoad={handleIframeOnload}
          src={src}
          component={"iframe"}
          allow="clipboard-read; clipboard-write"
          sx={{
            width: "100%",
            height: loaded ? "100%" : 0,
            border: "none",
            minHeight: loaded ? "calc(100vh - 200px)" : 0,
            visibility: loaded ? "visible" : "hidden",
          }}
        />
      )}
    </ViewerDialog>
  );
};

export default DrawIOViewer;
