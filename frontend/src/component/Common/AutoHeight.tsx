import React, { useEffect, useRef, useState } from "react";
import AnimateHeight, { Height } from "react-animate-height";
import { useTheme } from "@mui/material";

// @ts-ignore
const AutoHeight = ({ children, ...props }) => {
  const [height, setHeight] = useState<Height>("auto");
  const contentDiv = useRef<HTMLDivElement | null>(null);
  const theme = useTheme();

  useEffect(() => {
    const element = contentDiv.current as HTMLDivElement;

    const resizeObserver = new ResizeObserver(() => {
      setHeight(element.clientHeight);
    });

    resizeObserver.observe(element);

    return () => resizeObserver.disconnect();
  }, [contentDiv]);

  return (
    <AnimateHeight
      {...props}
      height={height}
      duration={theme.transitions.duration.standard}
      easing={theme.transitions.easing.easeInOut}
      contentClassName="auto-content"
      contentRef={contentDiv}
      disableDisplayNone
    >
      {children}
    </AnimateHeight>
  );
};

export default AutoHeight;
