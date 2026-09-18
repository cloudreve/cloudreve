import { alpha, lighten, useTheme } from "@mui/material";
import * as React from "react";
import { useCallback, useContext, useEffect, useRef } from "react";
import { FileManagerIndex } from "../component/FileManager/FileManager.tsx";
import { FmIndexContext } from "../component/FileManager/FmIndexContext.tsx";
import { clearSelected } from "../redux/fileManagerSlice.ts";
import { useAppDispatch, useAppSelector } from "../redux/hooks.ts";
import { selectionFromDragBox } from "../redux/thunks/file.ts";

const dataRectId = "data-rect-id";

interface Coordinates {
  x: number;
  y: number;
}

interface Candidates extends Coordinates {
  index: string;
  bottom: number;
  right: number;
}

interface DrawnAreaRef {
  start: Coordinates | null;
  end: Coordinates | null;
  ctrlKey: boolean;
  metaKey: boolean;
}

// Smallest value >= target
function binarySearchTop(list: Candidates[][], target: number) {
  let start = 0;
  let end = list.length - 1;
  while (start <= end) {
    const mid = Math.floor((start + end) / 2);
    if (list[mid][0].y < target) start = mid + 1;
    else end = mid - 1;
  }
  return end;
}

// Largest value <= target
function binarySearchBottom(list: Candidates[][], target: number) {
  let start = 0;
  let end = list.length - 1;
  while (start <= end) {
    const mid = Math.floor((start + end) / 2);
    if (list[mid][0].y <= target) start = mid + 1;
    else end = mid - 1;
  }
  return end;
}

const boxNode = document.createElement("div");
boxNode.style.position = "fixed";
boxNode.style.borderRadius = "2px";
boxNode.style.pointerEvents = "none";

export function useAreaSelection(container: React.RefObject<HTMLElement>, explorerIndex: number, enabled: boolean) {
  const theme = useTheme();
  const fmIndex = useContext(FmIndexContext);
  const dispatch = useAppDispatch();
  const boxRef = useRef<HTMLDivElement>(boxNode);
  const fileList = useAppSelector((state) => state.fileManager[explorerIndex]?.list?.files);

  const [mouseDown, setMouseDown] = React.useState<boolean>(false);
  const mouseMoving = useRef(false);
  const selectCandidates = useRef<Candidates[][]>([]);
  const elementsCacheRef = useRef<string | null>(null);
  const rafIdRef = useRef<number | null>(null);
  const drawAreaRef = useRef<DrawnAreaRef>({
    start: null,
    end: null,
    ctrlKey: false,
    metaKey: false,
  });

  const getPosition = useCallback((containerElement: HTMLElement, clientX: number, clientY: number): Coordinates => {
    const containerBox = containerElement.getBoundingClientRect();
    const y = containerElement.scrollTop + clientY - containerBox.top;
    const x = containerElement.scrollLeft + clientX - containerBox.left;
    return { x, y };
  }, []);

  const getDrawPosition = useCallback(
    (containerElement: HTMLElement, containerBox: DOMRect, cord: Coordinates): Coordinates => {
      const y = Math.min(
        Math.max(cord.y - containerElement.scrollTop + containerBox.top, containerBox.top),
        containerBox.bottom,
      );
      const x = Math.min(
        Math.max(cord.x - containerElement.scrollLeft + containerBox.left, containerBox.left),
        containerBox.right,
      );
      return { x, y };
    },
    [],
  );

  const updateCandidate = useCallback((containerElement: HTMLElement) => {
    const rows: Candidates[][] = [];
    const containerBox = containerElement.getBoundingClientRect();
    const scrollTop = containerElement.scrollTop;
    let currentY = -Infinity;
    let currentRow: Candidates[] = [];

    const elements = containerElement.querySelectorAll("[data-rect-id]");
    for (let i = 0; i < elements.length; i++) {
      const el = elements[i];
      if (el instanceof HTMLElement) {
        const rectId = el.getAttribute(dataRectId);
        if (rectId) {
          const rect = el.getBoundingClientRect();
          const y = scrollTop + rect.y - containerBox.y;
          if (y > currentY) {
            currentY = y;
            if (currentRow.length > 0) {
              rows.push(currentRow);
            }
            currentRow = [];
          }
          currentRow.push({
            index: rectId,
            x: rect.x - containerBox.x,
            y,
            bottom: scrollTop + rect.bottom - containerBox.y,
            right: rect.right - containerBox.x,
          });
        }
      }
    }
    if (currentRow.length > 0) {
      rows.push(currentRow);
    }
    selectCandidates.current = rows;
  }, []);

  const processSelection = useCallback(() => {
    const { start, end, ctrlKey, metaKey } = drawAreaRef.current;
    const containerElement = container.current;
    const boxElement = boxRef.current;

    if (!start || !end || !boxElement || !containerElement) {
      return;
    }

    const containerBox = containerElement.getBoundingClientRect();
    drawSelectionBox(
      boxElement,
      getDrawPosition(containerElement, containerBox, start),
      getDrawPosition(containerElement, containerBox, end),
    );

    const startX = Math.min(start.x, end.x);
    const startY = Math.min(start.y, end.y);
    const endX = Math.max(start.x, end.x);
    const endY = Math.max(start.y, end.y);

    const candidates = selectCandidates.current;
    if (candidates.length === 0) {
      return;
    }

    // Use binary search to find interest area in candidates
    const top = Math.max(0, binarySearchTop(candidates, startY));
    const bottom = binarySearchBottom(candidates, endY);

    // Collect matching elements without creating intermediate arrays
    const activeElements: string[] = [];
    for (let i = top; i <= bottom; i++) {
      const row = candidates[i];
      for (let j = 0; j < row.length; j++) {
        const el = row[j];
        if (!(el.x > endX || el.right < startX || el.y > endY || el.bottom < startY)) {
          activeElements.push(el.index);
        }
      }
    }

    // Use string key for fast comparison
    const cacheKey = activeElements.join(",");
    if (elementsCacheRef.current === cacheKey) {
      return;
    }
    elementsCacheRef.current = cacheKey;

    dispatch(selectionFromDragBox(0, activeElements, ctrlKey, metaKey));
  }, [container, dispatch, getDrawPosition]);

  const scheduleSelectionUpdate = useCallback(() => {
    if (rafIdRef.current !== null) {
      return;
    }
    rafIdRef.current = requestAnimationFrame(() => {
      rafIdRef.current = null;
      if (mouseMoving.current) {
        processSelection();
      }
    });
  }, [processSelection]);

  const handleMouseMove = useCallback(
    (e: React.MouseEvent<HTMLElement>) => {
      if (!mouseMoving.current || fmIndex === FileManagerIndex.selector || !enabled) {
        return;
      }

      document.body.style.userSelect = "none";
      const containerElement = container.current;
      if (!containerElement) {
        return;
      }

      const pos = getPosition(containerElement, e.clientX, e.clientY);
      drawAreaRef.current.end = pos;
      drawAreaRef.current.ctrlKey = e.ctrlKey;
      drawAreaRef.current.metaKey = e.metaKey;

      // Auto-scroll when near edges
      const containerBox = containerElement.getBoundingClientRect();
      const containerHeight = containerBox.height;
      const scrollMargin = containerHeight * 0.1;
      const distanceFromBottom = containerBox.bottom - e.clientY;
      const distanceFromTop = e.clientY - containerBox.top;

      if (distanceFromBottom < scrollMargin) {
        containerElement.scrollTop += 10;
      } else if (distanceFromTop < scrollMargin) {
        containerElement.scrollTop -= 10;
      }

      scheduleSelectionUpdate();
    },
    [container, enabled, fmIndex, getPosition, scheduleSelectionUpdate],
  );

  const handleMouseDown = useCallback(
    (e: React.MouseEvent<HTMLElement>) => {
      e.preventDefault();
      if (e.button !== 0 || !enabled) {
        return;
      }

      if (fmIndex === FileManagerIndex.selector) {
        return dispatch(clearSelected({ index: fmIndex, value: undefined }));
      }

      const containerElement = container.current;
      if (!containerElement || !containerElement.contains(e.target as HTMLElement)) {
        return;
      }

      setMouseDown(true);
      mouseMoving.current = true;
      elementsCacheRef.current = null;

      const pos = getPosition(containerElement, e.clientX, e.clientY);
      updateCandidate(containerElement);
      drawAreaRef.current = {
        start: pos,
        end: pos,
        ctrlKey: e.ctrlKey,
        metaKey: e.metaKey,
      };

      // Process selection immediately on mouse down to handle click-to-clear
      processSelection();
    },
    [container, dispatch, enabled, fmIndex, getPosition, updateCandidate, processSelection],
  );

  const handleMouseUp = useCallback(() => {
    document.body.style.userSelect = "initial";
    setMouseDown(false);
    mouseMoving.current = false;
    if (rafIdRef.current !== null) {
      cancelAnimationFrame(rafIdRef.current);
      rafIdRef.current = null;
    }
  }, []);

  // Update candidates when file list changes during drag
  useEffect(() => {
    if (mouseMoving.current && container.current) {
      updateCandidate(container.current);
      scheduleSelectionUpdate();
    }
  }, [fileList, container, updateCandidate, scheduleSelectionUpdate]);

  // Handle box element attachment/detachment
  useEffect(() => {
    const containerElement = container.current;
    const selectionBoxElement = boxRef.current;
    if (!containerElement || !selectionBoxElement) {
      return;
    }

    if (mouseDown) {
      if (!document.body.contains(selectionBoxElement)) {
        containerElement.appendChild(selectionBoxElement);
      }
    } else {
      if (containerElement.contains(selectionBoxElement)) {
        containerElement.removeChild(selectionBoxElement);
      }
    }
  }, [mouseDown, container]);

  // Update box styling when theme changes
  useEffect(() => {
    const selectionBoxElement = boxRef.current;
    if (selectionBoxElement) {
      if (theme.palette.mode === "dark") {
        selectionBoxElement.style.background = alpha(lighten(theme.palette.primary.main, 0.3), 0.2);
      } else {
        selectionBoxElement.style.background = alpha(theme.palette.primary.main, 0.3);
      }
      selectionBoxElement.style.boxShadow = `inset 0 0 0 2px ${theme.palette.primary.light}`;
    }
  }, [theme]);

  // Cleanup RAF on unmount
  useEffect(() => {
    return () => {
      if (rafIdRef.current !== null) {
        cancelAnimationFrame(rafIdRef.current);
      }
    };
  }, []);

  return [handleMouseDown, handleMouseUp, handleMouseMove] as const;
}

function drawSelectionBox(boxElement: HTMLElement, start: Coordinates, end: Coordinates): void {
  const b = boxElement;
  if (end.x > start.x) {
    b.style.left = start.x + "px";
    b.style.width = end.x - start.x + "px";
  } else {
    b.style.left = end.x + "px";
    b.style.width = start.x - end.x + "px";
  }

  if (end.y > start.y) {
    b.style.top = start.y + "px";
    b.style.height = end.y - start.y + "px";
  } else {
    b.style.top = end.y + "px";
    b.style.height = start.y - end.y + "px";
  }
}
