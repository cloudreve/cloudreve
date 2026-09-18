import { IReactReaderProps, IReactReaderStyle, ReactReader, ReactReaderStyle } from "react-reader";

import { type Rendition } from "epubjs";
import { useEffect, useRef } from "react";
import { useTheme } from "@mui/material";

function updateTheme(rendition: Rendition, theme: string) {
  const themes = rendition.themes;
  switch (theme) {
    case "dark": {
      themes.override("color", "#fff");
      themes.override("background", "#000");
      break;
    }
    case "light": {
      themes.override("color", "#000");
      themes.override("background", "#fff");
      break;
    }
  }
}

const lightReaderTheme: IReactReaderStyle = {
  ...ReactReaderStyle,
  readerArea: {
    ...ReactReaderStyle.readerArea,
    transition: undefined,
  },
};

const darkReaderTheme: IReactReaderStyle = {
  ...ReactReaderStyle,
  arrow: {
    ...ReactReaderStyle.arrow,
    color: "white",
  },
  arrowHover: {
    ...ReactReaderStyle.arrowHover,
    color: "#ccc",
  },
  readerArea: {
    ...ReactReaderStyle.readerArea,
    backgroundColor: "#000",
    transition: undefined,
  },
  titleArea: {
    ...ReactReaderStyle.titleArea,
    color: "#ccc",
  },
  tocArea: {
    ...ReactReaderStyle.tocArea,
    background: "#111",
  },
  tocButtonExpanded: {
    ...ReactReaderStyle.tocButtonExpanded,
    background: "#222",
  },
  tocButtonBar: {
    ...ReactReaderStyle.tocButtonBar,
    background: "#fff",
  },
  tocButton: {
    ...ReactReaderStyle.tocButton,
    color: "white",
  },
};

const Epub = (props: IReactReaderProps) => {
  const rendition = useRef<Rendition | undefined>(undefined);
  const theme = useTheme();
  useEffect(() => {
    if (rendition.current) {
      updateTheme(rendition.current, theme.palette.mode);
    }
  }, [theme.palette.mode]);

  return (
    <ReactReader
      {...props}
      readerStyles={theme.palette.mode === "dark" ? darkReaderTheme : lightReaderTheme}
      getRendition={(_rendition) => {
        updateTheme(_rendition, theme.palette.mode);
        rendition.current = _rendition;
      }}
    />
  );
};

export default Epub;
