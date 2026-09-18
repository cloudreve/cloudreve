import { Box } from "@mui/material";
import { useTheme } from "@mui/material/styles";
import logoDark from "../assets/logo.svg";
import logoLight from "../assets/logo_light.svg";

interface CloudreveLogoProps {
  height?: number;
}

export default function CloudreveLogo({ height = 28 }: CloudreveLogoProps) {
  const theme = useTheme();
  const logo = theme.palette.mode === "dark" ? logoLight : logoDark;

  return (
    <Box
      component="img"
      src={logo}
      alt="Cloudreve"
      sx={{ height }}
    />
  );
}
