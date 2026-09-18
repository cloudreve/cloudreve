import { Box, Chip, Dialog, DialogProps, Divider, IconButton, useMediaQuery, useTheme } from "@mui/material";
import React, { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { FileResponse } from "../../api/explorer.ts";
import FacebookCircularProgress from "../Common/CircularProgress.tsx";
import { NoWrapTypography } from "../Common/StyledComponents.tsx";
import { StyledDialogTitle } from "../Dialogs/DraggableDialog.tsx";
import FileIcon from "../FileManager/Explorer/FileIcon.tsx";
import Dismiss from "../Icons/Dismiss.tsx";
import FullScreenMaximize from "../Icons/FullScreenMaximize.tsx";
import FullScreenMinimize from "../Icons/FullScreenMinimize.tsx";

export interface ViewerDialogProps {
  file?: FileResponse;
  readOnly?: boolean;
  fullScreen?: boolean;
  actions?: React.ReactNode;
  fullScreenToggle?: boolean;
  dialogProps: DialogProps;
  loading?: boolean;
  children?: React.ReactNode;
  toggleFullScreen?: () => void;
}

export const ViewerLoading = ({ minHeight = "calc(100vh - 200px)" }: { minHeight?: string }) => (
  <Box
    sx={{
      width: "100%",
      height: "100%",
      border: "none",
      minHeight,
      display: "flex",
      justifyContent: "center",
      alignItems: "center",
    }}
  >
    <FacebookCircularProgress />
  </Box>
);

const ViewerDialog = (props: ViewerDialogProps) => {
  const { t } = useTranslation();
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const [fullScreen, setFullScreen] = useState(props.fullScreen || isMobile);
  const onClose = useCallback(() => {
    props.dialogProps.onClose && props.dialogProps.onClose({}, "closeButtonClick" as any);
  }, [props.dialogProps.onClose]);
  return (
    <Dialog
      fullScreen={fullScreen}
      {...props.dialogProps}
      onClose={props.loading ? undefined : props.dialogProps.onClose}
    >
      <Box>
        <StyledDialogTitle sx={{ py: "8px", px: "14px" }} id="draggable-dialog-title">
          {props.actions && props.actions}
          <Box sx={{ display: "flex", alignItems: "center", minWidth: 0 }}>
            <FileIcon variant={"default"} file={props.file} sx={{ px: 0, py: 0, pt: 0.5, mr: 1 }} fontSize={"small"} />
            <NoWrapTypography variant={"subtitle2"}>{props.file?.name}</NoWrapTypography>
            {props.readOnly && <Chip size="small" sx={{ ml: 1 }} label={t("fileManager.readOnly")} />}
          </Box>
          <Box sx={{ display: "flex" }}>
            {props.fullScreenToggle && (
              <IconButton
                onClick={() => {
                  props.toggleFullScreen ? props.toggleFullScreen() : setFullScreen((s) => !s);
                }}
              >
                {fullScreen ? <FullScreenMinimize fontSize={"small"} /> : <FullScreenMaximize fontSize={"small"} />}
              </IconButton>
            )}
            <IconButton disabled={props.loading} onClick={onClose}>
              <Dismiss fontSize={"small"} />
            </IconButton>
          </Box>
        </StyledDialogTitle>
        <Divider />
      </Box>
      {props.children}
    </Dialog>
  );
};

export default ViewerDialog;
