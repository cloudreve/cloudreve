import { Box, CircularProgress, circularProgressClasses, CircularProgressProps } from "@mui/material";
import { forwardRef } from "react";

export interface FacebookCircularProgressProps extends CircularProgressProps {
  bgColor?: string;
  fgColor?: string;
}

const FacebookCircularProgress = forwardRef(({ sx, bgColor, fgColor, ...rest }: FacebookCircularProgressProps, ref) => {
  return (
    <Box sx={{ position: "relative", ...sx }} ref={ref}>
      <CircularProgress
        variant="determinate"
        sx={{
          color: (theme) => bgColor ?? theme.palette.grey[theme.palette.mode === "light" ? 200 : 800],
        }}
        size={40}
        thickness={4}
        {...rest}
        value={100}
      />
      <CircularProgress
        sx={{
          color: (theme) => fgColor ?? theme.palette.primary.main,
          position: "absolute",
          left: 0,
          [`& .${circularProgressClasses.circle}`]: {
            strokeLinecap: "round",
          },
        }}
        size={40}
        thickness={4}
        {...rest}
      />
    </Box>
  );
});

export default FacebookCircularProgress;
