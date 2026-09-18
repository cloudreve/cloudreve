import { useDragLayer, XYCoord } from "react-dnd";
import { FileResponse } from "../../../api/explorer.ts";
import { Badge, Box, Paper, PaperProps } from "@mui/material";
import { useAppSelector } from "../../../redux/hooks.ts";
import { useEffect, useMemo, useState } from "react";
import { DragItem } from "./DndWrappedFile.tsx";
import DisableDropDelay from "./DisableDropDelay.tsx";
import { FileNameText, Header } from "../Explorer/GridView/GridFile.tsx";
import FileSmallIcon from "../Explorer/FileSmallIcon.tsx";

interface DragPreviewProps extends PaperProps {
  files: FileResponse[];
  pointerOffset: XYCoord | null;
}

const DragPreview = ({ pointerOffset, files, ...rest }: DragPreviewProps) => {
  const [size, setSize] = useState([0, 0]);
  useEffect(() => {
    setSize([220, 48]);
  }, []);
  if (!files || files.length == 0) {
    return undefined;
  }
  return (
    <Badge
      badgeContent={files.length <= 1 ? undefined : files.length}
      color="primary"
      sx={{
        "& .MuiBadge-badge": { zIndex: 1612 },

        transform: `translate(${pointerOffset?.x}px, ${pointerOffset?.y}px)`,
      }}
    >
      <Paper
        elevation={3}
        sx={{
          width: size[0],
          height: size[1],
          zIndex: 1610,
          transition: (theme) => theme.transitions.create(["width", "height"]),
        }}
        {...rest}
      >
        <Header>
          <FileSmallIcon ignoreHovered selected={false} file={files[0]} />
          <FileNameText variant="body2">{files[0]?.name}</FileNameText>
        </Header>
      </Paper>
      {[...Array(Math.min(2, files.length - 1)).keys()].map((i) => (
        <Paper
          sx={{
            position: "absolute",
            width: size[0],
            height: size[1],
            zIndex: 1610 - i - 1,
            top: (i + 1) * 4,
            left: (i + 1) * 4,
            transition: (theme) => theme.transitions.create(["width", "height"]),
          }}
          elevation={3}
        />
      ))}
    </Badge>
  );
};

const DragLayer = () => {
  DisableDropDelay();

  const { itemType, isDragging, item, pointerOffset } = useDragLayer((monitor) => ({
    item: monitor.getItem(),
    itemType: monitor.getItemType(),
    pointerOffset: monitor.getClientOffset(),
    isDragging: monitor.isDragging(),
  }));

  const selected = useAppSelector((state) => state.fileManager[0].selected);
  const draggingFiles = useMemo(() => {
    if (item && (item as DragItem) && item.target) {
      const selectedList = item.includeSelected
        ? Object.keys(selected)
            .map((key) => selected[key])
            .filter((x) => x.path != item.target.path)
        : [];
      return [item.target, ...selectedList];
    }

    return [];
  }, [selected, item]);

  if (!isDragging) {
    return null;
  }

  return (
    <Box
      sx={{
        position: "fixed",
        pointerEvents: "none",
        zIndex: 1600,
        left: 0,
        top: 0,
        width: "100%",
        height: "100%",
      }}
    >
      <DragPreview files={draggingFiles} pointerOffset={pointerOffset} />
    </Box>
  );
};

export default DragLayer;
