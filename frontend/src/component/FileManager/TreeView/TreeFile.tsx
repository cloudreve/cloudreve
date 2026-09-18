import React, { useCallback, useContext, useMemo, useState } from "react";

import { Fade, IconButton, Skeleton, styled, Tooltip } from "@mui/material";
import { TreeItem, treeItemClasses, TreeItemContentProps, TreeItemProps, useTreeItem } from "@mui/x-tree-view";
import clsx from "clsx";
import { useTranslation } from "react-i18next";
import { FileResponse } from "../../../api/explorer.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { openFileContextMenu } from "../../../redux/thunks/file.ts";
import { loadChild, navigateToPath } from "../../../redux/thunks/filemanager.ts";
import { unPinFromSidebar } from "../../../redux/thunks/settings.ts";
import { mergeRefs } from "../../../util";
import FacebookCircularProgress from "../../Common/CircularProgress.tsx";
import { NoWrapTypography } from "../../Common/StyledComponents.tsx";
import NavIconTransition from "../../Frame/NavBar/NavIconTransition.tsx";
import { SideNavItemBase } from "../../Frame/NavBar/SideNavItem.tsx";
import CaretDown from "../../Icons/CaretDown.tsx";
import Dismiss from "../../Icons/Dismiss.tsx";
import { useFileDrag } from "../Dnd/DndWrappedFile.tsx";
import FileIcon from "../Explorer/FileIcon.tsx";
import { FmIndexContext } from "../FmIndexContext.tsx";
import { StartIcon } from "../TopBar/BreadcrumbButton.tsx";
import { pinedPrefix } from "./TreeFiles.tsx";

const CustomContentRoot = styled(SideNavItemBase)<{
  isDragging?: boolean;
  isDropOver?: boolean;
}>(({ theme, isDragging, isDropOver }) => ({
  "& .MuiTreeItem-iconContainer": {
    marginLeft: theme.spacing(1),
  },
  opacity: isDragging ? 0.5 : 1,
  transition: "all 250ms cubic-bezier(0.4, 0, 0.2, 1) 0ms",
  transitionProperty: "opacity,box-shadow,background-color",
  boxShadow: isDropOver ? `inset 0 0 0 2px ${theme.palette.primary.light}` : "none",
  height: "32px",
}));

const StyledTreeItemRoot = styled(TreeItem)(() => ({
  [`& .${treeItemClasses.group}`]: {
    marginLeft: 0,
    [`& .${treeItemClasses.content}`]: {
      marginLeft: 0,
    },
  },
})) as unknown as typeof TreeItem;

export const CaretDownIcon = styled(CaretDown)<{ expanded: boolean }>(({ theme, expanded }) => ({
  fontSize: "12px!important",
  transform: `rotate(${expanded ? 0 : -90}deg)`,
  transition: theme.transitions.create("transform", {
    duration: theme.transitions.duration.shortest,
    easing: theme.transitions.easing.easeInOut,
  }),
}));

export interface CustomContentProps extends TreeItemContentProps {
  parent?: string;
  level?: number;
  notLoaded?: boolean;
  file?: FileResponse;
  fileIcon?: StartIcon;
  loading?: boolean;
  pinned?: boolean;
  canDrop?: boolean;
}

export interface TreeFileProps extends TreeItemProps {
  parent?: string;
  level?: number;
  notLoaded?: boolean;
  file?: FileResponse;
  fileIcon?: StartIcon;
  loading?: boolean;
  pinned?: boolean;
  canDrop?: boolean;
}

const SmallIconButton = styled(IconButton)(() => ({
  fontSize: "0.8rem",
}));

interface UnpinButton {
  show: boolean;
  uri: string;
}
const UnpinButton = (props: UnpinButton) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const [loading, setLoading] = useState(false);
  const onClick = useCallback(
    async (e: React.MouseEvent<HTMLButtonElement>) => {
      e.stopPropagation();
      setLoading(true);
      try {
        await dispatch(unPinFromSidebar(props.uri));
      } catch (e) {
        setLoading(false);
      }
    },
    [setLoading],
  );

  return (
    <Fade in={props.show} unmountOnExit>
      <Tooltip title={t("application:fileManager.unpin")}>
        <SmallIconButton disabled={loading} onMouseDown={(e) => e.stopPropagation()} onClick={onClick} size="small">
          <Dismiss fontSize={"inherit"} />
        </SmallIconButton>
      </Tooltip>
    </Fade>
  );
};

const CustomContent = React.memo(
  React.forwardRef(function CustomContent(props: CustomContentProps, ref) {
    const dispatch = useAppDispatch();
    const fmIndex = useContext(FmIndexContext);
    const [loading, setLoading] = useState(false);
    const [showDelete, setShowDelete] = useState(false);
    const { classes, className, label, nodeId, icon: iconProp, expansionIcon, displayIcon, file, fileIcon } = props;

    const { disabled, expanded, selected, focused, handleExpansion, handleSelection, preventSelection } =
      useTreeItem(nodeId);

    const uri = useMemo(() => {
      // Trim 'pinedPrefix' if exist in prefix
      if (nodeId.startsWith(pinedPrefix)) {
        return nodeId.substring(pinedPrefix.length);
      }

      return nodeId;
    }, [nodeId]);

    const icon = iconProp || expansionIcon || displayIcon;

    const handleMouseDown = (event: React.MouseEvent<HTMLDivElement, MouseEvent>) => {
      preventSelection(event);
    };

    const handleExpansionClick = useCallback(
      async (event: React.MouseEvent<HTMLDivElement, MouseEvent>) => {
        event.stopPropagation();
        let timeOutID: NodeJS.Timeout | undefined;
        handleExpansion(event);
        if (!expanded) {
          try {
            await dispatch(loadChild(fmIndex, uri, () => (timeOutID = setTimeout(() => setLoading(true), 300))));
          } finally {
            if (timeOutID) {
              clearTimeout(timeOutID);
            }
            setLoading(false);
          }
        }
      },
      [handleExpansion, setLoading, dispatch, uri],
    );

    const handleSelectionClick = useCallback(
      (event: React.MouseEvent<HTMLDivElement, MouseEvent>) => {
        event.stopPropagation();
        handleSelection(event);
        dispatch(navigateToPath(fmIndex, uri, file));
      },
      [dispatch, handleSelection, fmIndex, uri],
    );

    const FileItemIcon = useMemo(() => {
      if (props.loading) {
        return <Skeleton sx={{ mr: "14px" }} variant={"rounded"} width={20} height={20} />;
      }
      if (fileIcon && fileIcon.Icons) {
        return (
          <NavIconTransition
            sx={{ px: 0, py: 0, pr: "14px", height: "20px" }}
            iconProps={{ fontSize: "small", color: "action" }}
            fileIcon={fileIcon.Icons}
            active={selected}
          />
        );
      }
      if (fileIcon && fileIcon.Element) {
        return <fileIcon.Element sx={{ height: 20, width: 20, mr: "14px" }} />;
      }
      return (
        <FileIcon
          variant={"small"}
          file={file}
          sx={{ px: 0, py: 0, mr: "14px", height: "20px" }}
          fontSize={"small"}
          iconProps={{ fontSize: "small", sx: { minWidth: "20px" } }}
          notLoaded={props.notLoaded}
        />
      );
    }, [file, fileIcon, selected, props.notLoaded, props.loading]);

    const onContextMenu = useCallback(
      (e: React.MouseEvent<HTMLElement>) => {
        if (!file || file.name === "") {
          return;
        }
        dispatch(openFileContextMenu(fmIndex, file, true, e));
      },
      [file, dispatch, fmIndex],
    );

    const fileName = useMemo(
      () => (
        <Tooltip title={label} disableInteractive>
          <NoWrapTypography
            onClick={handleSelectionClick}
            component="div"
            sx={{
              flexGrow: 1,
              textAlign: "left",
            }}
            variant={"body2"}
          >
            {!props.loading && label}
            {props.loading && <Skeleton variant={"text"} width={75} />}
          </NoWrapTypography>
        </Tooltip>
      ),
      [label, handleSelectionClick, props.loading],
    );

    const onMouseEnter = useCallback(() => {
      if (props.pinned) setShowDelete(true);
    }, [setShowDelete, props.pinned]);

    const onMouseLeave = useCallback(() => {
      if (props.pinned) setShowDelete(false);
    }, [setShowDelete, props.pinned]);

    const [drag, drop, isOver, isDragging] = useFileDrag({
      file,
      dropUri: props.canDrop ? nodeId : undefined,
    });

    const mergedRef = useCallback(
      (val: any) => {
        mergeRefs(ref as React.Ref<HTMLDivElement>, drop, drag)(val);
      },
      [ref, drop, drag],
    );

    return (
      // eslint-disable-next-line jsx-a11y/no-static-element-interactions
      <CustomContentRoot
        onContextMenu={onContextMenu}
        onMouseEnter={onMouseEnter}
        onMouseLeave={onMouseLeave}
        isDragging={isDragging}
        isDropOver={isOver && !isDragging}
        className={clsx(className, classes.root, {
          [classes.expanded]: expanded,
          [classes.selected]: selected,
          [classes.focused]: selected,
          [classes.disabled]: disabled,
        })}
        active={selected}
        sx={{
          py: "4px!important",
          pl: (theme) => `${theme.spacing((props.level ?? 0) * 2)}!important`,
        }}
        onClick={handleSelectionClick}
        ref={mergedRef}
      >
        {/* eslint-disable-next-line jsx-a11y/click-events-have-key-events,jsx-a11y/no-static-element-interactions */}
        <div onClick={handleExpansionClick} className={classes.iconContainer}>
          {icon && !loading && <CaretDownIcon expanded={expanded} />}
          {icon && loading && <FacebookCircularProgress size={15} sx={{ pt: 0.5 }} />}
        </div>
        {FileItemIcon}
        {fileName}
        <UnpinButton show={showDelete} uri={uri} />
      </CustomContentRoot>
    );
  }),
);

const TreeFile = React.memo(
  React.forwardRef(function CustomTreeItem(props: TreeFileProps, ref: React.Ref<HTMLLIElement>) {
    const contentProps = useMemo(() => {
      const { level, file, notLoaded, fileIcon, loading, pinned, canDrop } = props;
      return { level, file, notLoaded, fileIcon, loading, pinned, canDrop };
    }, [props.level, props.file, props.notLoaded, props.fileIcon, props.loading, props.canDrop, props.pinned]);
    return (
      <StyledTreeItemRoot
        ContentComponent={CustomContent}
        // @ts-ignore
        ContentProps={contentProps}
        {...props}
        ref={ref}
      />
    );
  }),
);

export default TreeFile;
