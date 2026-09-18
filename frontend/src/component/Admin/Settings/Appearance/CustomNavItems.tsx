import { Icon } from "@iconify/react/dist/iconify.js";
import {
  Box,
  IconButton,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
  useTheme,
} from "@mui/material";
import React, { useCallback, useEffect, useMemo, useState } from "react";
import { DndProvider, useDrag, useDrop } from "react-dnd";
import { HTML5Backend } from "react-dnd-html5-backend";
import { useTranslation } from "react-i18next";
import { CustomNavItem } from "../../../../api/site";
import {
  DenseFilledTextField,
  NoWrapCell,
  NoWrapTableCell,
  SecondaryButton,
  StyledTableContainerPaper,
} from "../../../Common/StyledComponents";
import Add from "../../../Icons/Add";
import ArrowDown from "../../../Icons/ArrowDown";
import Delete from "../../../Icons/Delete";

export interface CustomNavItemsProps {
  value: string;
  onChange: (value: string) => void;
}

const DND_TYPE = "custom-nav-item-row";

// 拖拽item类型
type DragItem = { index: number };

interface DraggableNavItemRowProps {
  item: CustomNavItem;
  index: number;
  moveRow: (from: number, to: number) => void;
  onDelete: (index: number) => void;
  onMoveUp: () => void;
  onMoveDown: () => void;
  isFirst: boolean;
  isLast: boolean;
  inputCache: { [key: number]: { [field: string]: string | undefined } };
  onInputChange: (index: number, field: string, value: string) => void;
  onInputBlur: (index: number, field: keyof CustomNavItem) => void;
  IconPreview: React.ComponentType<{ iconName: string }>;
  t: any;
  style?: React.CSSProperties;
}

const DraggableNavItemRow = React.memo(
  React.forwardRef<HTMLTableRowElement, DraggableNavItemRowProps>(
    (
      {
        item,
        index,
        moveRow,
        onDelete,
        onMoveUp,
        onMoveDown,
        isFirst,
        isLast,
        inputCache,
        onInputChange,
        onInputBlur,
        IconPreview,
        t,
        style,
      },
      ref,
    ): JSX.Element => {
      const [, drop] = useDrop<DragItem>({
        accept: DND_TYPE,
        hover(dragItem, monitor) {
          if (!(ref && typeof ref !== "function" && ref.current)) return;
          const dragIndex = dragItem.index;
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
          dragItem.index = hoverIndex;
        },
      });
      const [{ isDragging }, drag] = useDrag<DragItem, void, { isDragging: boolean }>({
        type: DND_TYPE,
        item: { index },
        collect: (monitor) => ({
          isDragging: monitor.isDragging(),
        }),
      });
      // 兼容ref为function和对象
      const setRowRef = (node: HTMLTableRowElement | null) => {
        if (typeof ref === "function") {
          ref(node);
        } else if (ref) {
          (ref as React.MutableRefObject<HTMLTableRowElement | null>).current = node;
        }
        drag(drop(node));
      };
      return (
        <TableRow ref={setRowRef} hover style={{ opacity: isDragging ? 0.5 : 1, cursor: "move", ...style }}>
          <TableCell>
            <IconPreview iconName={item.icon} />
          </TableCell>
          <TableCell>
            <DenseFilledTextField
              fullWidth
              required
              value={inputCache[index]?.icon ?? item.icon}
              onChange={(e) => onInputChange(index, "icon", e.target.value)}
              onBlur={() => onInputBlur(index, "icon")}
              placeholder={t("settings.iconifyNamePlaceholder")}
            />
          </TableCell>
          <TableCell>
            <DenseFilledTextField
              fullWidth
              required
              value={inputCache[index]?.name ?? item.name}
              onChange={(e) => onInputChange(index, "name", e.target.value)}
              onBlur={() => onInputBlur(index, "name")}
              placeholder={t("settings.displayNameDes")}
            />
          </TableCell>
          <TableCell>
            <DenseFilledTextField
              fullWidth
              required
              value={inputCache[index]?.url ?? item.url}
              onChange={(e) => onInputChange(index, "url", e.target.value)}
              onBlur={() => onInputBlur(index, "url")}
              placeholder="https://example.com"
            />
          </TableCell>
          <TableCell align="right">
            <IconButton size="small" onClick={() => onDelete(index)}>
              <Delete fontSize="small" />
            </IconButton>
          </TableCell>
          <TableCell>
            <IconButton size="small" onClick={onMoveUp} disabled={isFirst}>
              <ArrowDown
                sx={{
                  width: "18px",
                  height: "18px",
                  transform: "rotate(180deg)",
                }}
              />
            </IconButton>
            <IconButton size="small" onClick={onMoveDown} disabled={isLast}>
              <ArrowDown
                sx={{
                  width: "18px",
                  height: "18px",
                }}
              />
            </IconButton>
          </TableCell>
        </TableRow>
      );
    },
  ),
);

const CustomNavItems = ({ value, onChange }: CustomNavItemsProps) => {
  const { t } = useTranslation("dashboard");
  const theme = useTheme();
  const [items, setItems] = useState<CustomNavItem[]>([]);
  const [inputCache, setInputCache] = useState<{
    [key: number]: { [field: string]: string | undefined };
  }>({});

  useEffect(() => {
    try {
      const parsedItems = JSON.parse(value);
      setItems(Array.isArray(parsedItems) ? parsedItems : []);
    } catch (e) {
      setItems([]);
    }
  }, [value]);

  const handleSave = useCallback(
    (newItems: CustomNavItem[]) => {
      onChange(JSON.stringify(newItems));
    },
    [onChange],
  );

  const handleDelete = useCallback(
    (index: number) => {
      const newItems = items.filter((_, i) => i !== index);
      handleSave(newItems);
    },
    [items, handleSave],
  );

  const handleAdd = useCallback(() => {
    const newItems = [
      ...items,
      {
        name: "",
        url: "",
        icon: "fluent:home-24-regular",
      },
    ];
    handleSave(newItems);
  }, [items, handleSave]);

  const handleFieldChange = useCallback(
    (index: number, field: keyof CustomNavItem, value: string) => {
      const newItems = [...items];
      newItems[index] = {
        ...newItems[index],
        [field]: value,
      };
      handleSave(newItems);
    },
    [items, handleSave],
  );

  const handleInputChange = useCallback((index: number, field: string, value: string) => {
    setInputCache((prev) => ({
      ...prev,
      [index]: {
        ...prev[index],
        [field]: value,
      },
    }));
  }, []);

  const handleInputBlur = useCallback(
    (index: number, field: keyof CustomNavItem) => {
      const cachedValue = inputCache[index]?.[field];
      if (cachedValue !== undefined) {
        handleFieldChange(index, field, cachedValue);
        setInputCache((prev) => ({
          ...prev,
          [index]: {
            ...prev[index],
            [field]: undefined,
          },
        }));
      }
    },
    [inputCache, handleFieldChange],
  );

  // 拖拽排序逻辑
  const moveRow = useCallback(
    (from: number, to: number) => {
      if (from === to) return;
      const updated = [...items];
      const [moved] = updated.splice(from, 1);
      updated.splice(to, 0, moved);
      setItems(updated);
      handleSave(updated);
    },
    [items, handleSave],
  );

  const handleMoveUp = (idx: number) => {
    if (idx <= 0) return;
    moveRow(idx, idx - 1);
  };
  const handleMoveDown = (idx: number) => {
    if (idx >= items.length - 1) return;
    moveRow(idx, idx + 1);
  };

  const IconPreview = useMemo(
    () =>
      ({ iconName }: { iconName: string }) => {
        if (!iconName) {
          return (
            <Box
              sx={{
                width: 24,
                height: 24,
                bgcolor: "grey.300",
                borderRadius: "4px",
              }}
            />
          );
        }

        return (
          <Icon
            icon={iconName}
            width={24}
            height={24}
            style={{
              color: theme.palette.action.active,
            }}
          />
        );
      },
    [],
  );

  return (
    <Box>
      <Typography variant="h6" gutterBottom>
        {t("settings.customNavItems")}
      </Typography>
      <Typography variant="body2" color="text.secondary" paragraph>
        {t("settings.customNavItemsDes")}
      </Typography>

      <TableContainer component={StyledTableContainerPaper} sx={{ mt: 2 }}>
        <DndProvider backend={HTML5Backend}>
          <Table size="small" stickyHeader sx={{ width: "100%", tableLayout: "fixed" }}>
            <TableHead>
              <TableRow>
                <NoWrapTableCell width={60}>{t("settings.icon")}</NoWrapTableCell>
                <NoWrapTableCell width={200}>{t("settings.iconifyName")}</NoWrapTableCell>
                <NoWrapTableCell width={200}>{t("settings.displayName")}</NoWrapTableCell>
                <NoWrapTableCell width={250}>{t("settings.navItemUrl")}</NoWrapTableCell>
                <NoWrapTableCell width={80} align="right"></NoWrapTableCell>
                <NoWrapTableCell width={80}></NoWrapTableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {items.map((item, index) => {
                const rowRef = React.createRef<HTMLTableRowElement>();
                return (
                  <DraggableNavItemRow
                    key={index}
                    ref={rowRef}
                    item={item}
                    index={index}
                    moveRow={moveRow}
                    onDelete={handleDelete}
                    onMoveUp={() => handleMoveUp(index)}
                    onMoveDown={() => handleMoveDown(index)}
                    isFirst={index === 0}
                    isLast={index === items.length - 1}
                    inputCache={inputCache}
                    onInputChange={handleInputChange}
                    onInputBlur={handleInputBlur}
                    IconPreview={IconPreview}
                    t={t}
                  />
                );
              })}
              {items.length === 0 && (
                <TableRow>
                  <NoWrapCell colSpan={6} align="center">
                    <Typography variant="caption" color="text.secondary">
                      {t("application:setting.listEmpty")}
                    </Typography>
                  </NoWrapCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </DndProvider>
      </TableContainer>

      <SecondaryButton variant="contained" startIcon={<Add />} onClick={handleAdd} sx={{ mt: 2 }}>
        {t("settings.addNavItem")}
      </SecondaryButton>
    </Box>
  );
};

export default CustomNavItems;
