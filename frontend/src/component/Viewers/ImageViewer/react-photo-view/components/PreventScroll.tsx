import { useTheme } from "@mui/material";
import useMediaQuery from "@mui/material/useMediaQuery";
import { useEffect } from "react";

export default function PreventScroll() {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  useEffect(() => {
    document.body.style.overflow = "hidden";

    return () => {
      alert(isMobile);
      document.body.style.overflow = isMobile ? "initial" : "hidden";
    };
  }, []);

  return null;
}
