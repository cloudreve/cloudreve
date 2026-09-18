import { Box, Stack } from "@mui/material";
import React from "react";
import { useTranslation } from "react-i18next";
import useMediaQuery from "@mui/material/useMediaQuery";
import { useTheme } from "@mui/material/styles";

export interface LogoPreviewProps {
  logoLight: string;
  logoDark: string;
}

const LogoPreview = ({ logoLight, logoDark }: LogoPreviewProps) => {
  const { t } = useTranslation("dashboard");
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));

  return (
    <Stack spacing={1} direction={"row"} sx={{ mt: isMobile ? 0 : 3 }}>
      <Box
        sx={{
          backgroundColor: (theme) => theme.palette.grey[100],
          p: 1,
          borderRadius: (theme) => `${theme.shape.borderRadius}px`,
        }}
      >
        <Box
          component={"img"}
          src={logoDark}
          sx={{
            display: "block",
            height: "auto",
            maxWidth: 160,
            maxHeight: 35,
            width: "100%",
            objectPosition: "left",
            objectFit: "contain",
          }}
        />
      </Box>
      <Box
        sx={{
          backgroundColor: (theme) => theme.palette.grey[900],
          p: 1,
          borderRadius: (theme) => `${theme.shape.borderRadius}px`,
        }}
      >
        <Box
          component={"img"}
          src={logoLight}
          sx={{
            display: "block",
            height: "auto",
            maxWidth: 160,
            maxHeight: 35,
            width: "100%",
            objectPosition: "left",
            objectFit: "contain",
          }}
        />
      </Box>
    </Stack>
  );
};

export default LogoPreview;
