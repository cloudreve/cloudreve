import { memo, useCallback, useContext, useEffect } from "react";
import { useDrag, useDrop } from "react-dnd";
import { getEmptyImage } from "react-dnd-html5-backend";
import { FileResponse, FileType } from "../../../api/explorer.ts";
import { setDragging } from "../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { processDnd } from "../../../redux/thunks/file.ts";
import { getFileLinkedUri, mergeRefs } from "../../../util";

import { useTheme } from "@mui/material/styles";
import useMediaQuery from "@mui/material/useMediaQuery";
import CrUri, { Filesystem } from "../../../util/uri.ts";
import { FileBlockProps } from "../Explorer/Explorer.tsx";
import { FileManagerIndex } from "../FileManager.tsx";
import { FmIndexContext } from "../FmIndexContext.tsx";

export interface DragItem {
  target: FileResponse;
  includeSelected?: boolean;
}

export interface DropResult {
  dropEffect: string;
  uri?: string;
}

export const DropEffect = {
  copy: "copy",
  move: "move",
};

export interface UseFileDragProps {
  file?: FileResponse;
  includeSelected?: boolean;
  dropUri?: string;
}

export const NoOpDropUri = "noop";
export const useFileDrag = ({ file, includeSelected, dropUri }: UseFileDragProps) => {
  const dispatch = useAppDispatch();
  const theme = useTheme();
  const isTablet = useMediaQuery(theme.breakpoints.down("md"));
  const fmIndex = useContext(FmIndexContext);
  // const { addEventListenerForWindow, removeEventListenerForWindow } =
  //   useDragScrolling(["#" + MainExplorerContainerID]);

  // @ts-ignore
  const [{ isDragging }, drag, preview] = useDrag({
    type: "file",
    item: {
      target: file,
      includeSelected,
    },
    end: (item, monitor) => {
      // Ignore NoOpDropUri
      const target = monitor.getDropResult<DropResult>();
      if (!item || !target || !target.uri || target.uri == NoOpDropUri) {
        return;
      }

      dispatch(processDnd(0, item as DragItem, target));
    },
    canDrag: () => {
      if (!file || fmIndex == FileManagerIndex.selector || isTablet) {
        return false;
      }

      const crUri = new CrUri(file.path);
      return file.owned && crUri.fs() != Filesystem.share;
    },
    collect: (monitor) => ({
      isDragging: monitor.isDragging(),
    }),
  });

  const [{ canDrop, isOver }, drop] = useDrop({
    accept: "file",
    drop: () => (file ? { uri: getFileLinkedUri(file) } : { uri: dropUri }),
    collect: (monitor) => ({
      isOver: monitor.isOver(),
      canDrop: monitor.canDrop(),
    }),
    canDrop: (item, _monitor) => {
      const dropExist = !!dropUri || (!!file && file.type == FileType.folder);
      if (!dropExist || fmIndex == FileManagerIndex.selector) {
        return false;
      }

      if (!item) {
        return false;
      }

      return true;
    },
  });
  const isActive = canDrop && isOver;

  useEffect(() => {
    preview(getEmptyImage(), { captureDraggingState: true });
    // eslint-disable-next-line
  }, []);

  useEffect(() => {
    if (isDragging) {
      // addEventListenerForWindow();
    }
    dispatch(
      setDragging({
        dragging: isDragging,
        draggingWithSelected: !!includeSelected,
      }),
    );
  }, [isDragging]);

  return [drag, drop, isActive, isDragging] as const;
};

export interface DndWrappedFileProps extends FileBlockProps {
  component: React.MemoExoticComponent<(props: FileBlockProps) => JSX.Element>;
}

const DndWrappedFile = memo((props: DndWrappedFileProps) => {
  const fmIndex = useContext(FmIndexContext);
  const globalDragging = useAppSelector((state) => state.globalState.dndState);
  const isSelected = useAppSelector((state) => state.fileManager[fmIndex].selected[props.file.path]);

  const [drag, drop, isOver, isDragging] = useFileDrag({
    file: props.file.placeholder ? undefined : props.file,
    includeSelected: true,
  });

  const mergedRef = useCallback(
    (val: any) => {
      mergeRefs(drop, drag)(val);
    },
    [drop, drag],
  );

  const Component = props.component;

  return (
    <Component
      dragRef={mergedRef}
      isDropOver={isOver}
      isDragging={isDragging || (!!globalDragging.dragging && !!isSelected && globalDragging.draggingWithSelected)}
      {...props}
    />
  );
});

export default DndWrappedFile;
