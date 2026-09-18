import { Box, ClickAwayListener, Menu, styled, TextField, useMediaQuery, useTheme } from "@mui/material";
import React, { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useIsOverflow } from "../../../hooks/useOverflow.tsx";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { navigateToPath } from "../../../redux/thunks/filemanager.ts";
import { mergeRefs } from "../../../util";
import CrUri from "../../../util/uri.ts";
import ChevronRight from "../../Icons/ChevronRight.tsx";
import MoreHorizontal from "../../Icons/MoreHorizontal.tsx";
import { NoOpDropUri, useFileDrag } from "../Dnd/DndWrappedFile.tsx";
import { FmIndexContext } from "../FmIndexContext.tsx";
import BreadcrumbButton, { BreadcrumbButtonBase, BreadcrumbButtonProps } from "./BreadcrumbButton.tsx";
import BreadcrumbHiddenItem from "./BreadcrumbHiddenItem.tsx";

const PathTextField = styled(TextField)(() => ({
  "& .MuiOutlinedInput-notchedOutline": {
    border: "none",
  },
  "& .MuiOutlinedInput-input": {
    padding: "6px 4px",
    fontSize: "0.8125rem",
    lineHeight: "1.5",
    fontFamily: "monospace",
  },
  height: "32px",
  overflow: "hidden",
  verticalAlign: "middle",
}));

const RightIcon = styled(ChevronRight)(({ theme }) => ({
  fontSize: 15,
  mx: 0.5,
  verticalAlign: "middle",
  color: theme.palette.text.disabled,
}));

interface pathElements extends BreadcrumbButtonProps {}

const useBreadcrumb = (targetPath?: string) => {
  if (targetPath) {
    const uri = new CrUri(targetPath);
    const elements = uri.elements();
    return [targetPath, elements, uri.base(true)] as const;
  }

  const index = useContext(FmIndexContext);

  const base = useAppSelector((s) => s.fileManager[index].path_root_with_category);
  const path = useAppSelector((s) => s.fileManager[index].path);
  const elements = useAppSelector((s) => s.fileManager[index].path_elements);

  return [path, elements, base] as const;
};

export interface BreadcrumbProps {
  targetPath?: string;
  displayOnly?: boolean;
}

const Breadcrumb = (props: BreadcrumbProps) => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const dispatch = useAppDispatch();
  const fmIndex = useContext(FmIndexContext);
  const previousElements = useRef(0);
  const [editing, setEditing] = useState(false);
  const [editedPath, setEditedPath] = useState("");
  const [maxHiding, setMaxHiding] = useState(0);
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const hiddenOpen = Boolean(anchorEl);
  const hiddenExpandButtonRef = useRef<HTMLButtonElement>();
  const chainRef = useRef(null);

  const openHiddenMenu = (e: React.MouseEvent<HTMLElement>) => {
    e.stopPropagation();
    setAnchorEl(e.currentTarget);
  };

  const [path, elements, base] = useBreadcrumb(props.targetPath);

  const onEdit = useCallback(() => {
    if (props.displayOnly) {
      return;
    }
    if (path) {
      const uri = new CrUri(path);
      setEditedPath(uri.path());
    }
    setEditing(true);
  }, [path, props.displayOnly]);

  const buttons = useMemo(() => {
    if (!base || !elements) {
      return [];
    }

    const res: pathElements[] = [];

    // First, inject root button
    res.push({
      path: base,
      is_root: true,
      displayOnly: props.displayOnly,
      is_latest: elements.length === 0,
    });

    const currentBase = new CrUri(base);
    res.push(
      ...elements.map((e, i) => {
        return {
          path: currentBase.join(e).toString(),
          name: e,
          displayOnly: props.displayOnly,
          is_latest: i === elements.length - 1,
        };
      }),
    );

    return res;
  }, [elements, base, props.displayOnly]);

  const hidedButtons = useMemo(() => {
    return buttons.slice(0, maxHiding);
  }, [buttons, maxHiding]);

  const displayedButtons = useMemo(() => {
    return buttons.slice(maxHiding);
  }, [buttons, maxHiding]);

  const isOverflow = useIsOverflow(chainRef, (_isOverflow) => {});

  useEffect(() => {
    if (isOverflow && !isMobile) {
      setMaxHiding(buttons.length - 1);
    }
  }, [isOverflow, isMobile]);

  // Cancel collapse when elements are less than previous
  useEffect(() => {
    const current = elements?.length ?? 0;
    if (previousElements.current > current) {
      setTimeout(() => {
        setMaxHiding(0);
      }, theme.transitions.duration.standard);
    }
    previousElements.current = current;
  }, [elements]);

  const submitNewPath = useCallback(() => {
    setEditing(false);
    if (!path || !editedPath) {
      return;
    }
    const uri = new CrUri(path);
    if (uri.path() == editedPath || props.displayOnly) {
      // No change
      return;
    }

    // Apply new path and navigate
    dispatch(navigateToPath(fmIndex, uri.setPath(editedPath).toString()));
  }, [path, editedPath, props.displayOnly, fmIndex, dispatch]);

  const [drag, drop, isOver, isDragging] = useFileDrag({
    dropUri: NoOpDropUri,
  });

  useEffect(() => {
    if (isOver && hiddenExpandButtonRef.current) {
      hiddenExpandButtonRef.current?.click();
    }
  }, [isOver]);

  return (
    <>
      <Box
        ref={props.displayOnly ? undefined : chainRef}
        sx={{
          flexGrow: 1,
          whiteSpace: props.displayOnly ? "initial" : "nowrap",
          cursor: "text",
          overflow: "auto",
        }}
        onClick={onEdit}
      >
        {editing && (
          <ClickAwayListener mouseEvent={"onMouseDown"} onClickAway={() => submitNewPath()}>
            <PathTextField
              autoFocus
              onFocus={(e) => {
                e.target.select();
              }}
              fullWidth
              onClick={(e) => e.stopPropagation()}
              InputProps={{
                readOnly: props.displayOnly,
              }}
              onKeyPress={(e) => {
                if (e.key === "Enter") {
                  submitNewPath();
                }
              }}
              value={editedPath}
              onChange={(e) => setEditedPath(e.target.value)}
            />
          </ClickAwayListener>
        )}
        {!editing && (
          <>
            {hidedButtons.length > 0 && (
              <BreadcrumbButtonBase
                ref={mergeRefs(drop, hiddenExpandButtonRef)}
                size={"small"}
                isDropOver={isOver}
                onClick={openHiddenMenu}
              >
                <MoreHorizontal />
              </BreadcrumbButtonBase>
            )}
            {...displayedButtons.map((b, i) => (
              <>
                {i == 0 && hidedButtons.length > 0 && <RightIcon />}
                <BreadcrumbButton {...b} />
                {i != displayedButtons.length - 1 && <RightIcon />}
              </>
            ))}
          </>
        )}
      </Box>
      <Menu
        anchorEl={anchorEl}
        open={hiddenOpen}
        onClose={() => setAnchorEl(null)}
        slotProps={{
          list: {
            dense: true,
          },
        }}
      >
        {hidedButtons &&
          hidedButtons.map((b) => <BreadcrumbHiddenItem key={b.path} onClose={() => setAnchorEl(null)} {...b} />)}
      </Menu>
    </>
  );
};

export default Breadcrumb;
