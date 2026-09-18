import {
  Box,
  DialogContent,
  IconButton,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
} from "@mui/material";
import { useSnackbar } from "notistack";
import type { Dispatch, SetStateAction } from "react";
import React, { useCallback, useEffect, useState } from "react";
import { DndProvider, useDrag, useDrop } from "react-dnd";
import { HTML5Backend } from "react-dnd-html5-backend";
import { useTranslation } from "react-i18next";
import { setListViewColumnSettingDialog } from "../../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../../redux/hooks.ts";
import { applyListColumns } from "../../../../redux/thunks/filemanager.ts";
import AutoHeight from "../../../Common/AutoHeight.tsx";
import { DefaultCloseAction } from "../../../Common/Snackbar/snackbar.tsx";
import { StyledTableContainerPaper } from "../../../Common/StyledComponents.tsx";
import DraggableDialog from "../../../Dialogs/DraggableDialog.tsx";
import ArrowDown from "../../../Icons/ArrowDown.tsx";
import Dismiss from "../../../Icons/Dismiss.tsx";
import { FileManagerIndex } from "../../FileManager.tsx";
import AddColumn from "./AddColumn.tsx";
import { getColumnTypeDefaults, ListViewColumnSetting } from "./Column.tsx";

const DND_TYPE = "column-row";

interface DraggableColumnRowProps {
  column: ListViewColumnSetting;
  index: number;
  moveRow: (from: number, to: number) => void;
  columns: ListViewColumnSetting[];
  setColumns: Dispatch<SetStateAction<ListViewColumnSetting[]>>;
  t: (key: string) => string;
  onDelete: (idx: number) => void;
  isFirst: boolean;
  isLast: boolean;
}

const DraggableColumnRow: React.FC<DraggableColumnRowProps> = ({
  column,
  index,
  moveRow,
  columns,
  t,
  onDelete,
  isFirst,
  isLast,
}) => {
  const ref = React.useRef<HTMLTableRowElement>(null);
  const customProps = useAppSelector((state) => state.siteConfig.explorer?.config?.custom_props);
  const [, drop] = useDrop({
    accept: DND_TYPE,
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
    type: DND_TYPE,
    item: { index },
    collect: (monitor) => ({
      isDragging: monitor.isDragging(),
    }),
  });
  drag(drop(ref));
  return (
    <TableRow
      ref={ref}
      hover
      key={index}
      sx={{ "&:last-child td, &:last-child th": { border: 0 }, opacity: isDragging ? 0.5 : 1, cursor: "move" }}
    >
      <TableCell component="th" scope="row">
        {t(getColumnTypeDefaults(column, false, customProps).title)}
      </TableCell>
      <TableCell>
        <Box sx={{ display: "flex" }}>
          <IconButton size="small" onClick={() => moveRow(index, index - 1)} disabled={isFirst}>
            <ArrowDown
              sx={{
                width: "18px",
                height: "18px",
                transform: "rotate(180deg)",
              }}
            />
          </IconButton>
          <IconButton size="small" onClick={() => moveRow(index, index + 1)} disabled={isLast}>
            <ArrowDown
              sx={{
                width: "18px",
                height: "18px",
              }}
            />
          </IconButton>
          <IconButton size="small" onClick={() => onDelete(index)} disabled={columns.length <= 1}>
            <Dismiss sx={{ width: "18px", height: "18px" }} />
          </IconButton>
        </Box>
      </TableCell>
    </TableRow>
  );
};

const ColumnSetting = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const { enqueueSnackbar } = useSnackbar();
  const [columns, setColumns] = useState<ListViewColumnSetting[]>([]);

  const open = useAppSelector((state) => state.globalState.listViewColumnSettingDialogOpen);
  const listViewColumns = useAppSelector((state) => state.fileManager[FileManagerIndex.main].listViewColumns);

  useEffect(() => {
    if (open) {
      setColumns(listViewColumns ?? []);
    }
  }, [open]);

  const onClose = useCallback(() => {
    dispatch(setListViewColumnSettingDialog(false));
  }, [dispatch]);

  const onSubmitted = useCallback(() => {
    if (columns.length > 0) {
      dispatch(applyListColumns(FileManagerIndex.main, columns));
    }
    dispatch(setListViewColumnSettingDialog(false));
  }, [dispatch, columns]);

  const onColumnAdded = useCallback(
    (column: ListViewColumnSetting) => {
      const existed = columns.find((c) => c.type === column.type);
      if (
        !existed ||
        existed.props?.metadata_key != column.props?.metadata_key ||
        existed.props?.custom_props_id != column.props?.custom_props_id
      ) {
        setColumns((prev) => [...prev, column]);
      } else {
        enqueueSnackbar(t("application:fileManager.columnExisted"), {
          variant: "warning",
          action: DefaultCloseAction,
        });
      }
    },
    [columns],
  );

  return (
    <DraggableDialog
      title={t("application:fileManager.listColumnSetting")}
      onAccept={onSubmitted}
      showActions
      secondaryAction={<AddColumn onColumnAdded={onColumnAdded} />}
      dialogProps={{
        open: open ?? false,
        onClose: onClose,
        fullWidth: true,
        maxWidth: "sm",
        disableRestoreFocus: true,
      }}
    >
      <DialogContent sx={{ pb: 0 }}>
        <AutoHeight>
          <DndProvider backend={HTML5Backend}>
            <TableContainer component={StyledTableContainerPaper}>
              <Table sx={{ width: "100%" }} size="small">
                <TableHead>
                  <TableRow>
                    <TableCell width={"50%"}>{t("fileManager.column")}</TableCell>
                    <TableCell>{t("fileManager.actions")}</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {columns.map((column, index) => (
                    <DraggableColumnRow
                      key={index}
                      column={column}
                      index={index}
                      moveRow={(from, to) => {
                        if (from === to || to < 0 || to >= columns.length) return;
                        setColumns((prev) => {
                          const arr = [...prev];
                          const [moved] = arr.splice(from, 1);
                          arr.splice(to, 0, moved);
                          return arr;
                        });
                      }}
                      columns={columns}
                      setColumns={setColumns}
                      t={t}
                      onDelete={(idx) => setColumns((prev) => prev.filter((_, i) => i !== idx))}
                      isFirst={index === 0}
                      isLast={index === columns.length - 1}
                    />
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          </DndProvider>
        </AutoHeight>
      </DialogContent>
    </DraggableDialog>
  );
};
export default ColumnSetting;
