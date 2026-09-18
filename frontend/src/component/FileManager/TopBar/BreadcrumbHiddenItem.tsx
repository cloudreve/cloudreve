import { BreadcrumbButtonProps, useBreadcrumbButtons } from "./BreadcrumbButton.tsx";
import { ListItemIcon, ListItemText, MenuItem, Skeleton, styled } from "@mui/material";
import { useContext, useMemo } from "react";
import { useAppSelector } from "../../../redux/hooks.ts";
import FileIcon from "../Explorer/FileIcon.tsx";
import { useFileDrag } from "../Dnd/DndWrappedFile.tsx";

import { FmIndexContext } from "../FmIndexContext.tsx";

export interface BreadcrumbHiddenItem extends BreadcrumbButtonProps {
  onClose: () => void;
}

export const StyledMenuItem = styled(MenuItem)<{ isDropOver?: boolean }>(({ theme, isDropOver }) => ({
  transition: "all 250ms cubic-bezier(0.4, 0, 0.2, 1) 0ms !important",
  transitionProperty: "background-color,opacity,box-shadow",
  boxShadow: isDropOver ? `inset 0 0 0 2px ${theme.palette.primary.light}` : "none",
}));

const BreadcrumbHiddenItem = ({ name, is_root, is_latest, path, onClose, ...rest }: BreadcrumbHiddenItem) => {
  const [loading, displayName, startIcon, onClick] = useBreadcrumbButtons({
    name,
    is_latest,
    path,
    count_share_views: true,
  });
  const onItemClick = async () => {
    onClose();
    onClick && onClick();
  };

  const fmIndex = useContext(FmIndexContext);
  const file = useAppSelector((s) => s.fileManager[fmIndex].tree[path]?.file);

  const StartIcon = useMemo(() => {
    if (loading) {
      return <Skeleton width={18} height={18} variant={"rounded"} />;
    }
    if (startIcon?.Icons?.[0]) {
      const Icon = startIcon?.Icons?.[0];
      return <Icon fontSize={"small"} />;
    }
    if (startIcon?.Element) {
      return startIcon.Element({ sx: { width: 20, height: 20 } });
    }
  }, [startIcon, loading]);

  const [drag, drop, isOver, isDragging] = useFileDrag({
    dropUri: path,
  });

  return (
    <StyledMenuItem isDropOver={isOver} ref={drop} {...rest} onClick={onItemClick}>
      <ListItemIcon>
        {StartIcon ? (
          StartIcon
        ) : (
          <FileIcon
            sx={{ px: 0, py: 0, height: "20px" }}
            file={file}
            variant={"small"}
            name={name ?? ""}
            iconProps={{
              fontSize: "small",
            }}
          />
        )}
      </ListItemIcon>
      <ListItemText>{loading ? <Skeleton variant={"text"} width={75} /> : displayName}</ListItemText>
    </StyledMenuItem>
  );
};

export default BreadcrumbHiddenItem;
