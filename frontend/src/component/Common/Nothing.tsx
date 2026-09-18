import React from "react";
import { Box, Typography } from "@mui/material";
import PackageOpen from "../Icons/PackageOpen.tsx";

export interface NothingProps {
  primary: string;
  secondary?: string;
  top?: number;
  size?: number;
}

export default function Nothing({ primary, secondary, top = 20, size = 1 }: NothingProps) {
  return (
    <Box
      sx={{
        margin: `${50 * size}px auto`,
        paddingTop: `${top}px`,
        bottom: "0",

        color: (theme) => theme.palette.action.disabled,
        textAlign: "center",
      }}
    >
      <PackageOpen
        sx={{
          fontSize: 100 * size,
        }}
      />
      <Typography
        variant={"h5"}
        sx={{
          fontWeight: 500,
          color: (theme) => theme.palette.action.disabled,
        }}
      >
        {primary}
      </Typography>
      {secondary && (
        <Typography variant={"body2"} sx={{ color: (theme) => theme.palette.action.disabled }}>
          {secondary}
        </Typography>
      )}
    </Box>
  );
}
