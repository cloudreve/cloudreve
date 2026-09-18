import { Box } from "@mui/material";
import MuiSnackbarContent from "@mui/material/SnackbarContent";
import { CustomContentProps, SnackbarAction } from "notistack";
import * as React from "react";
import { forwardRef, useEffect, useState } from "react";
import CircularProgress from "../CircularProgress.tsx";

declare module "notistack" {
  interface VariantOverrides {
    loading: {
      getProgress?: () => number;
      secondaryAction?: SnackbarAction;
    };
  }
}

interface LoadingSnackbarProps extends CustomContentProps {
  getProgress?: () => number;
  secondaryAction?: SnackbarAction;
}

const LoadingSnackbar = forwardRef<HTMLDivElement, LoadingSnackbarProps>((props, ref) => {
  const [progress, setProgress] = useState(0);
  const { message, action, id, getProgress, secondaryAction, ...other } = props;

  useEffect(() => {
    var intervalId: NodeJS.Timeout;
    if (getProgress) {
      intervalId = setInterval(() => {
        setProgress(getProgress());
      }, 1000);
    }

    return () => {
      clearInterval(intervalId);
    };
  }, [getProgress]);

  const resolveAction = (a: SnackbarAction): React.ReactNode => {
    if (typeof a === "function") return a(id);
    return a;
  };

  const primaryNode = resolveAction(action);
  const secondaryNode = resolveAction(secondaryAction);

  return (
    <MuiSnackbarContent
      ref={ref}
      sx={{ borderRadius: "12px", maxWidth: 600 }}
      message={
        <Box
          sx={{
            display: "flex",
            alignItems: "center",
            flexWrap: "wrap",
            width: "100%",
          }}
        >
          <Box>
            <CircularProgress
              size={20}
              sx={{ height: 20, mr: 2 }}
              variant={progress ? "determinate" : "indeterminate"}
              value={progress}
            />
          </Box>
          <Box sx={{ flex: 1 }}>{message}</Box>
          {primaryNode && (
            <Box
              sx={{
                display: "flex",
                alignItems: "center",
                marginLeft: "auto",
                paddingLeft: "16px",
                marginRight: "-8px",
              }}
            >
              {primaryNode}
            </Box>
          )}
          {secondaryNode && <Box sx={{ width: "100%" }}>{secondaryNode}</Box>}
        </Box>
      }
    />
  );
});

export default LoadingSnackbar;
