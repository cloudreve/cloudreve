import { Box } from "@mui/material";
import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import useMediaQuery from "@mui/material/useMediaQuery";
import { useTheme } from "@mui/material/styles";

export interface GeneralImagePreviewProps {
  src: string;
  debounce?: number; // (可选) 防抖时间
}

const GeneralImagePreview = ({ src, debounce = 0 }: GeneralImagePreviewProps) => {
  const { t } = useTranslation("dashboard");
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));

  const [debouncedSrc, setDebouncedSrc] = useState(src);

  useEffect(() => {
    const handler = setTimeout(() => {
      setDebouncedSrc(src);
    }, debounce);

    return () => clearTimeout(handler);
  }, [src, debounce]);

  return (
    <Box sx={{ mt: isMobile ? 0 : 3 }}>
      <Box
        sx={{
          border: (t) => `1px solid ${t.palette.divider}`,
          p: 1,
          display: "inline-block",
          borderRadius: (theme) => `${theme.shape.borderRadius}px`,
        }}
      >
        <Box
          component={"img"}
          src={debouncedSrc}
          sx={{
            display: "block",
            maxWidth: "100%",
          }}
        />
      </Box>
    </Box>
  );
};

export default GeneralImagePreview;
