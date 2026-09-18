import { ExpandMoreRounded } from "@mui/icons-material";
import {
  AccordionDetails,
  Box,
  Link,
  ListItemIcon,
  Menu,
  Table,
  TableBody,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from "@mui/material";
import { bindMenu, bindTrigger, usePopupState } from "material-ui-popup-state/hooks";
import * as React from "react";
import { memo, useCallback, useMemo, useState } from "react";
import { DndProvider, useDrag, useDrop } from "react-dnd";
import { HTML5Backend } from "react-dnd-html5-backend";
import { useTranslation } from "react-i18next";
import { Viewer, ViewerGroup, ViewerType } from "../../../../api/explorer.ts";
import { uuidv4 } from "../../../../util";
import { NoWrapTableCell, SecondaryButton } from "../../../Common/StyledComponents.tsx";
import { SquareMenuItem } from "../../../FileManager/ContextMenu/ContextMenu.tsx";
import Add from "../../../Icons/Add.tsx";
import DesktopFlow from "../../../Icons/DesktopFlow.tsx";
import DocumentDataLink from "../../../Icons/DocumentDataLink.tsx";
import { AccordionSummary, StyledAccordion } from "../../Settings/UserSession/SSOSettings.tsx";
import FileViewerEditDialog from "./FileViewerEditDialog.tsx";
import FileViewerRow from "./FileViewerRow.tsx";
import ImportWopiDialog from "./ImportWopiDialog.tsx";

interface ViewerGroupProps {
  group: ViewerGroup;
  index: number;
  onDelete: (e: React.MouseEvent<HTMLElement>) => void;
  onGroupChange: (g: ViewerGroup) => void;
  dndType: string;
}

const DND_TYPE = "viewer-row";

const DraggableViewerRow = memo(function DraggableViewerRow({
  viewer,
  index,
  moveRow,
  onChange,
  onDelete,
  onMoveUp,
  onMoveDown,
  isLast,
  isFirst,
  dndType,
}: any) {
  const ref = React.useRef<HTMLTableRowElement>(null);
  const [, drop] = useDrop({
    accept: dndType,
    hover(item: any, monitor) {
      if (!ref.current) return;

      const dragIndex = item.index;
      const hoverIndex = index;
      if (dragIndex === hoverIndex) return;

      const hoverBoundingRect = ref.current.getBoundingClientRect();
      const hoverMiddleY = (hoverBoundingRect.bottom - hoverBoundingRect.top) / 2;
      const clientOffset = monitor.getClientOffset();
      if (!clientOffset) return;

      const hoverClientY = clientOffset.y - hoverBoundingRect.top;

      if (dragIndex < hoverIndex && hoverClientY < hoverMiddleY) return;
      if (dragIndex > hoverIndex && hoverClientY > hoverMiddleY) return;

      moveRow(dragIndex, hoverIndex);
      item.index = hoverIndex;
    },
  });
  const [{ isDragging }, drag] = useDrag({
    type: dndType,
    item: { index },
    collect: (monitor) => ({
      isDragging: monitor.isDragging(),
    }),
  });
  drag(drop(ref));
  return (
    <FileViewerRow
      ref={ref}
      viewer={viewer}
      onChange={onChange}
      onDelete={onDelete}
      onMoveUp={onMoveUp}
      onMoveDown={onMoveDown}
      isLast={isLast}
      isFirst={isFirst}
      style={{ opacity: isDragging ? 0.5 : 1, cursor: "move" }}
    />
  );
});

const ViewerGroupRow = memo(({ group, index, onDelete, onGroupChange, dndType }: ViewerGroupProps) => {
  const { t } = useTranslation("dashboard");

  const onViewerChange = useMemo(() => {
    return group.viewers.map((_, index) => (vChanged: Viewer) => {
      onGroupChange({
        viewers: group.viewers.map((v, i) => (i == index ? vChanged : v)),
      });
    });
  }, [group.viewers]);

  const onViewerDeleted = useMemo(() => {
    return group.viewers.map((_, index) => (e: React.MouseEvent<HTMLElement>) => {
      onGroupChange({
        viewers: group.viewers.filter((_, i) => i != index),
      });
      e.preventDefault();
      e.stopPropagation();
    });
  }, [group.viewers]);

  const [viewers, setViewers] = useState(group.viewers);
  React.useEffect(() => {
    setViewers(group.viewers);
  }, [group.viewers]);
  const moveRow = useCallback(
    (from: number, to: number) => {
      if (from === to) return;
      const updated = [...viewers];
      const [moved] = updated.splice(from, 1);
      updated.splice(to, 0, moved);
      setViewers(updated);
      onGroupChange({ viewers: updated });
    },
    [viewers, onGroupChange],
  );
  const handleMoveUp = (idx: number) => {
    if (idx <= 0) return;
    moveRow(idx, idx - 1);
  };
  const handleMoveDown = (idx: number) => {
    if (idx >= viewers.length - 1) return;
    moveRow(idx, idx + 1);
  };

  return (
    <StyledAccordion
      defaultExpanded={index == 0}
      disableGutters
      slotProps={{
        transition: {
          unmountOnExit: true,
        },
      }}
    >
      <AccordionSummary expandIcon={<ExpandMoreRounded />}>
        <Box
          sx={{
            display: "flex",
            justifyContent: "space-between",
            width: "100%",
            alignItems: "center",
          }}
        >
          <Typography>{t("settings.viewerGroupTitle", { index: index + 1 })}</Typography>
          {index > 0 && (
            <Link href={"#"} onClick={onDelete}>
              {t("policy.delete")}
            </Link>
          )}
        </Box>
      </AccordionSummary>
      <AccordionDetails sx={{ display: "block" }}>
        <DndProvider backend={HTML5Backend}>
          <TableContainer sx={{ mt: 1, maxHeight: 440 }}>
            <Table stickyHeader sx={{ width: "100%", tableLayout: "fixed" }} size="small">
              <TableHead>
                <TableRow>
                  <NoWrapTableCell width={64}>{t("settings.icon")}</NoWrapTableCell>
                  <NoWrapTableCell width={100}>{t("settings.viewerType")}</NoWrapTableCell>
                  <NoWrapTableCell width={200}>{t("settings.displayName")}</NoWrapTableCell>
                  <NoWrapTableCell width={250}>{t("settings.exts")}</NoWrapTableCell>
                  <NoWrapTableCell width={150}>{t("settings.viewerPlatform")}</NoWrapTableCell>
                  <NoWrapTableCell width={100}>{t("settings.newFileAction")}</NoWrapTableCell>
                  <NoWrapTableCell width={64}>{t("settings.viewerEnabled")}</NoWrapTableCell>
                  <NoWrapTableCell width={100}>{t("settings.actions")}</NoWrapTableCell>
                  <NoWrapTableCell width={100}></NoWrapTableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {viewers.map((viewer, idx) => (
                  <DraggableViewerRow
                    key={viewer.id}
                    viewer={viewer}
                    index={idx}
                    moveRow={moveRow}
                    onChange={onViewerChange[idx]}
                    onDelete={onViewerDeleted[idx]}
                    onMoveUp={() => handleMoveUp(idx)}
                    onMoveDown={() => handleMoveDown(idx)}
                    isFirst={idx === 0}
                    isLast={idx === viewers.length - 1}
                    dndType={dndType}
                  />
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </DndProvider>
      </AccordionDetails>
    </StyledAccordion>
  );
});

export interface FileViewerListProps {
  config: string;
  onChange: (value: string) => void;
}

const FileViewerList = memo(({ config, onChange }: FileViewerListProps) => {
  const { t } = useTranslation("dashboard");
  const addNewPopupState = usePopupState({
    variant: "popover",
    popupId: "addNewViewer",
  });
  const [createNewOpen, setCreateNewOpen] = useState(false);
  const [newViewer, setNewViewer] = useState<Viewer | undefined>(undefined);
  const [importOpen, setImportOpen] = useState(false);

  const configParsed = useMemo((): ViewerGroup[] => JSON.parse(config), [config]);

  const onNewViewerChange = useCallback(
    (v: Viewer) => {
      setNewViewer(v);
      const newViewerSetting = [...configParsed];
      newViewerSetting[0].viewers.push(v);
      onChange(JSON.stringify(newViewerSetting));
    },
    [configParsed],
  );

  const onGroupDelete = useMemo(() => {
    return configParsed.map((_, index) => (e: React.MouseEvent<HTMLElement>) => {
      onChange(JSON.stringify([...configParsed].filter((_, i) => i != index)));
      e.preventDefault();
      e.stopPropagation();
    });
  }, [configParsed]);

  const onGroupChange = useMemo(() => {
    return configParsed.map((_, index) => (g: ViewerGroup) => {
      onChange(JSON.stringify([...configParsed].map((item, i) => (i == index ? g : item))));
    });
  }, [configParsed]);

  const { onClose, ...menuProps } = bindMenu(addNewPopupState);

  const onCreateNewClosed = useCallback(() => {
    setCreateNewOpen(false);
  }, []);

  const openCreateNew = useCallback(() => {
    setNewViewer({
      id: uuidv4(),
      icon: "",
      type: ViewerType.custom,
      display_name: "",
      exts: [],
    });
    setCreateNewOpen(true);
    onClose();
  }, [onClose, setNewViewer]);

  const openImportNew = useCallback(() => {
    setImportOpen(true);
    onClose();
  }, [onClose, setImportOpen]);

  const onImportedNew = useCallback(
    (v: ViewerGroup) => {
      const newViewerSetting = [...configParsed];
      newViewerSetting.push(v);
      onChange(JSON.stringify(newViewerSetting));
    },
    [configParsed],
  );

  return (
    <Box>
      <SecondaryButton variant={"contained"} {...bindTrigger(addNewPopupState)} startIcon={<Add />} sx={{ mb: 1 }}>
        {t("settings.addViewer")}
      </SecondaryButton>
      {configParsed?.length > 0 &&
        configParsed.map((item: ViewerGroup, index) => (
          <ViewerGroupRow
            group={item}
            index={index}
            key={index}
            onDelete={onGroupDelete[index]}
            onGroupChange={onGroupChange[index]}
            dndType={`viewer-row-${index}`}
          />
        ))}
      <Menu
        onClose={onClose}
        anchorOrigin={{
          vertical: "top",
          horizontal: "left",
        }}
        transformOrigin={{
          vertical: "bottom",
          horizontal: "left",
        }}
        {...menuProps}
      >
        <SquareMenuItem dense onClick={openCreateNew}>
          <ListItemIcon>
            <DocumentDataLink />
          </ListItemIcon>
          {t("settings.embeddedWebpageViewer")}
        </SquareMenuItem>
        <SquareMenuItem dense onClick={openImportNew}>
          <ListItemIcon>
            <DesktopFlow />
          </ListItemIcon>
          {t("settings.wopiViewer")}
        </SquareMenuItem>
      </Menu>
      {newViewer && (
        <FileViewerEditDialog
          viewer={newViewer}
          onChange={onNewViewerChange}
          open={createNewOpen}
          onClose={onCreateNewClosed}
        />
      )}
      <ImportWopiDialog onImported={onImportedNew} onClose={() => setImportOpen(false)} open={importOpen} />
    </Box>
  );
});

export default FileViewerList;
