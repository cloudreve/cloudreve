import { Excalidraw as ExcalidrawComponent } from "@excalidraw/excalidraw";
import { OrderedExcalidrawElement } from "@excalidraw/excalidraw/element/types";
import "@excalidraw/excalidraw/index.css";
import { AppState, BinaryFiles } from "@excalidraw/excalidraw/types";
import { Box } from "@mui/material";
import { useMemo } from "react";
import "./excalidraw.css";

export interface ExcalidrawProps {
  value: string;
  initialValue: string;
  darkMode?: boolean;
  onChange: (value: string) => void;
  readOnly?: boolean;
  language?: string;
  onSaveShortcut?: () => void;
}

interface ExcalidrawState {
  elements: OrderedExcalidrawElement[];
  appState: AppState;
  files: BinaryFiles;
  type: string;
  version: number;
  source: string;
}

const serializeExcalidrawState = (elements: readonly OrderedExcalidrawElement[], appState: any, file: BinaryFiles) => {
  if (!Array.isArray(appState.collaborators)) {
    appState.collaborators = [];
  }
  return JSON.stringify({
    type: "excalidraw",
    version: 2,
    source: window.location.origin,
    elements,
    appState,
    files: file,
  });
};

const Excalidraw = (props: ExcalidrawProps) => {
  const initialValue = useMemo(() => {
    try {
      return JSON.parse(props.initialValue) as ExcalidrawState;
    } catch (error) {
      return null;
    }
  }, [props.initialValue]);
  return (
    <Box
      sx={{
        width: "100%",
        height: "100%",
        minHeight: "calc(100vh - 200px)",
      }}
      onKeyDown={(e) => {
        if ((e.ctrlKey || e.metaKey) && e.key === "s") {
          e.preventDefault();
          props.onSaveShortcut?.();
        }
      }}
    >
      <ExcalidrawComponent
        isCollaborating={false}
        viewModeEnabled={props.readOnly}
        onChange={(elements, state, file) => {
          props.onChange(serializeExcalidrawState(elements, state, file));
        }}
        initialData={initialValue}
        langCode={props.language}
        theme={props.darkMode ? "dark" : "light"}
      />
    </Box>
  );
};

export default Excalidraw;
