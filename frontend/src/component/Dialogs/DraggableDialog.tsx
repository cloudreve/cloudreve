import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContentText,
  DialogProps,
  DialogTitle,
  IconButton,
  Paper,
  PaperProps,
  Stack,
  styled,
  useMediaQuery,
} from "@mui/material";

import { LoadingButton } from "@mui/lab";
import { useCallback } from "react";
import Draggable from "react-draggable";
import { useTranslation } from "react-i18next";
import Dismiss from "../Icons/Dismiss.tsx";

function PaperComponent(props: PaperProps) {
  return (
    <Draggable handle="#draggable-dialog-title" cancel={'[class*="MuiDialogContent-root"]'}>
      <Paper {...props} />
    </Draggable>
  );
}

export const StyledDialogActions = styled(DialogActions)<{
  denseAction?: boolean;
}>(({ theme, denseAction }) => ({
  padding: `${theme.spacing(denseAction ? 0.5 : 2)} ${theme.spacing(3)}`,
  justifyContent: "space-between",
}));

export const StyledDialogContentText = styled(DialogContentText)(({ theme }) => ({
  fontSize: theme.typography.body2.fontSize,
  wordBreak: "break-all",
}));

export const StyledDialogTitle = styled(DialogTitle)<{ moveable?: boolean }>(({ moveable }) => ({
  display: "flex",
  justifyContent: "space-between",
  alignItems: "center",
  cursor: moveable ? "move" : "initial",
}));

export interface DraggableDialogProps {
  dialogProps: DialogProps;
  children?: React.ReactNode;
  secondaryAction?: React.ReactNode;
  showActions?: boolean;
  showCancel?: boolean;
  hideOk?: boolean;
  okText?: string;
  cancelText?: string;
  title?: string | React.ReactNode;
  onAccept?: () => void;
  loading?: boolean;
  disabled?: boolean;
  denseAction?: boolean;
  secondaryFullWidth?: boolean;
}

const DraggableDialog = (props: DraggableDialogProps) => {
  const { t } = useTranslation();
  const isTouch = useMediaQuery("(pointer: coarse)");
  const onClose = useCallback(() => {
    props.dialogProps.onClose && props.dialogProps.onClose({}, "backdropClick");
  }, [props.dialogProps.onClose]);
  return (
    <Dialog
      PaperComponent={isTouch ? undefined : PaperComponent}
      {...props.dialogProps}
      onClose={props.loading ? undefined : props.dialogProps.onClose}
    >
      {props.title != undefined && (
        <Box>
          <StyledDialogTitle moveable id="draggable-dialog-title">
            <Box>{props.title}</Box>
            <IconButton disabled={props.loading} onClick={onClose}>
              <Dismiss fontSize={"small"} />
            </IconButton>
          </StyledDialogTitle>
        </Box>
      )}
      {props.children}
      {props.showActions && (
        <StyledDialogActions denseAction={props.denseAction}>
          <Box sx={{ flexGrow: props.secondaryFullWidth ? 1 : "unset" }}>{props.secondaryAction}</Box>
          <Stack direction={"row"} spacing={1}>
            {props.showCancel && (
              <Button disabled={props.loading} onClick={onClose}>
                {props.cancelText ?? t("common:cancel")}
              </Button>
            )}
            {!props.hideOk && (
              <LoadingButton
                disabled={props.disabled}
                loading={props.loading}
                variant={"contained"}
                onClick={props.onAccept}
                color="primary"
              >
                <span>{props.okText ?? t("common:ok")}</span>
              </LoadingButton>
            )}
          </Stack>
        </StyledDialogActions>
      )}
    </Dialog>
  );
};

export default DraggableDialog;
