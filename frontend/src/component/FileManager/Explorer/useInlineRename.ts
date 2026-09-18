import { enqueueSnackbar } from "notistack";
import React, { useCallback, useContext, useEffect, useRef, useState } from "react";
import { useHotkeys } from "react-hotkeys-hook";
import { FileType } from "../../../api/explorer.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { fileDoubleClicked, inlineRenameSubmit } from "../../../redux/thunks/file.ts";
import { DefaultCloseAction } from "../../Common/Snackbar/snackbar.tsx";
import { getActionOpt } from "../ContextMenu/useActionDisplayOpt.ts";
import { FmIndexContext } from "../FmIndexContext.tsx";
import { FmFile } from "./GridView/GridView.tsx";

export interface UseInlineRenameOptions {
  file: FmFile;
  isSelected?: boolean;
  uploading?: boolean;
  isTouch: boolean;
}

export function useInlineRename({ file, isSelected, uploading, isTouch }: UseInlineRenameOptions) {
  const dispatch = useAppDispatch();
  const fmIndex = useContext(FmIndexContext);

  const [isEditing, setIsEditing] = useState(false);
  const [editValue, setEditValue] = useState("");
  const renameTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const isSoleSelected = useAppSelector(
    (state) =>
      !!state.fileManager[fmIndex].selected[file.path] && Object.keys(state.fileManager[fmIndex].selected).length === 1,
  );

  // Clear rename timer on unmount
  useEffect(() => {
    return () => {
      if (renameTimerRef.current) {
        clearTimeout(renameTimerRef.current);
      }
    };
  }, []);

  // Auto-focus and select text when entering edit mode
  useEffect(() => {
    if (isEditing && inputRef.current) {
      inputRef.current.focus();
      const lastDot = file.type == FileType.folder ? 0 : file.name.lastIndexOf(".");
      inputRef.current.setSelectionRange(0, lastDot > 0 ? lastDot : file.name.length);
    }
  }, [isEditing]);

  // Disable draggable on the ancestor while editing so react-dnd's
  // HTML5Backend doesn't hijack mousedown+mousemove for text selection.
  useEffect(() => {
    if (!isEditing || !inputRef.current) return;
    const draggableAncestor = inputRef.current.closest<HTMLElement>('[draggable="true"]');
    if (!draggableAncestor) return;
    draggableAncestor.setAttribute("draggable", "false");
    return () => {
      draggableAncestor.setAttribute("draggable", "true");
    };
  }, [isEditing]);

  // Exit edit mode and cancel rename timer when deselected
  useEffect(() => {
    if (!isSelected) {
      if (renameTimerRef.current) {
        clearTimeout(renameTimerRef.current);
        renameTimerRef.current = null;
      }
      if (isEditing) {
        setTimeout(() => {
          setIsEditing(false);
        }, 500);
      }
    }
  }, [isSelected, isEditing]);

  const enterEditMode = useCallback(() => {
    const displayOpt = getActionOpt([file]);
    if (!displayOpt.showRename) {
      return;
    }
    setIsEditing(true);
    setEditValue(file.name);
  }, [file]);

  const canRename = isSoleSelected && !uploading && !isEditing;
  useHotkeys("f2", enterEditMode, { enabled: canRename, preventDefault: true }, [enterEditMode, canRename]);

  const onNameClick = useCallback(
    (e: React.MouseEvent<HTMLElement>) => {
      const displayOpt = getActionOpt([file]);
      if (!displayOpt.showRename || !canRename || isTouch || e.ctrlKey || e.metaKey || e.shiftKey) {
        return;
      }
      e.stopPropagation();
      if (renameTimerRef.current) {
        clearTimeout(renameTimerRef.current);
      }
      renameTimerRef.current = setTimeout(enterEditMode, 500);
    },
    [file, canRename, isTouch, enterEditMode],
  );

  const onNameDoubleClick = useCallback(
    (e: React.MouseEvent<HTMLElement>) => {
      if (renameTimerRef.current) {
        clearTimeout(renameTimerRef.current);
        renameTimerRef.current = null;
      }
      e.stopPropagation();
      dispatch(fileDoubleClicked(fmIndex, file, e));
    },
    [dispatch, fmIndex, file],
  );

  const submitRename = useCallback(async () => {
    const newName = editValue;
    if (newName === file.name) {
      setIsEditing(false);
      return;
    }

    const error: string | undefined = await dispatch(inlineRenameSubmit(fmIndex, file, newName));
    if (error) {
      enqueueSnackbar({ message: error, variant: "warning", action: DefaultCloseAction });
      inputRef.current?.focus();
      return;
    }

    setIsEditing(false);
  }, [editValue, file, dispatch, fmIndex]);

  const onInputKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      e.stopPropagation();
      if (e.key === "Enter" && !e.nativeEvent.isComposing) {
        e.preventDefault();
        submitRename();
      } else if (e.key === "Escape") {
        e.preventDefault();
        setIsEditing(false);
      }
    },
    [submitRename],
  );

  const onInputBlur = useCallback(() => {
    submitRename();
  }, [submitRename]);

  const onInputChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setEditValue(e.target.value);
  }, []);

  const stopPropagation = useCallback((e: React.SyntheticEvent) => {
    e.stopPropagation();
  }, []);

  return {
    isEditing,
    editValue,
    inputRef,
    onNameClick,
    onNameDoubleClick,
    onInputKeyDown,
    onInputBlur,
    onInputChange,
    stopPropagation,
  };
}
