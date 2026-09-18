import { Icon } from "@iconify/react/dist/iconify.js";
import { Box, IconButton, SvgIconProps, TableRow, useTheme } from "@mui/material";
import React from "react";
import { useDrag, useDrop } from "react-dnd";
import { CustomProps, CustomPropsType } from "../../../../api/explorer";
import { NoWrapCell } from "../../../Common/StyledComponents";
import { getPropsContent } from "../../../FileManager/Sidebar/CustomProps/CustomPropsItem";
import ArrowDown from "../../../Icons/ArrowDown";
import CheckboxChecked from "../../../Icons/CheckboxChecked";
import DataBarVerticalStar from "../../../Icons/DataBarVerticalStar";
import Dismiss from "../../../Icons/Dismiss";
import Edit from "../../../Icons/Edit";
import LinkOutlined from "../../../Icons/LinkOutlined";
import Numbers from "../../../Icons/Numbers";
import PersonOutlined from "../../../Icons/PersonOutlined";
import SlideText from "../../../Icons/SlideText";
import TaskListRegular from "../../../Icons/TaskListRegular";
import TextIndentIncrease from "../../../Icons/TextIndentIncrease";

const DND_TYPE = "storage-product-row";

// 拖拽item类型
type DragItem = { index: number };

export interface DraggableCustomPropsRowProps {
  customProps: CustomProps;
  index: number;
  moveRow: (from: number, to: number) => void;
  onEdit: (customProps: CustomProps) => void;
  onDelete: (id: string) => void;
  onMoveUp: () => void;
  onMoveDown: () => void;
  isFirst: boolean;
  isLast: boolean;
  t: any;
  style?: React.CSSProperties;
}

export interface FieldTypeProps {
  title: string;
  icon: (props: SvgIconProps) => JSX.Element;
  minTitle?: string;
  minDes?: string;
  maxTitle?: string;
  maxDes?: string;
  maxRequired?: boolean;
  showOptions?: boolean;
  pro?: boolean;
}

export const FieldTypes: Record<CustomPropsType, FieldTypeProps> = {
  [CustomPropsType.text]: {
    title: "customProps.text",
    icon: SlideText,
    minTitle: "customProps.minLength",
    minDes: "customProps.emptyLimit",
    maxTitle: "customProps.maxLength",
    maxDes: "customProps.emptyLimit",
  },
  [CustomPropsType.number]: {
    title: "customProps.number",
    icon: Numbers,
    minTitle: "customProps.minValue",
    minDes: "customProps.emptyLimit",
    maxTitle: "customProps.maxValue",
    maxDes: "customProps.emptyLimit",
  },
  [CustomPropsType.boolean]: {
    title: "customProps.boolean",
    icon: CheckboxChecked,
  },
  [CustomPropsType.select]: {
    title: "customProps.select",
    icon: TextIndentIncrease,
    showOptions: true,
  },
  [CustomPropsType.multi_select]: {
    title: "customProps.multiSelect",
    icon: TaskListRegular,
    showOptions: true,
  },
  [CustomPropsType.user]: {
    title: "customProps.user",
    icon: PersonOutlined,
    pro: true,
  },
  [CustomPropsType.link]: {
    title: "customProps.link",
    icon: LinkOutlined,
    minTitle: "customProps.minLength",
    minDes: "customProps.emptyLimit",
    maxTitle: "customProps.maxLength",
    maxDes: "customProps.emptyLimit",
  },
  [CustomPropsType.rating]: {
    title: "customProps.rating",
    icon: DataBarVerticalStar,
    maxRequired: true,
    maxTitle: "customProps.maxValue",
  },
};

const DraggableCustomPropsRow = React.memo(
  React.forwardRef<HTMLTableRowElement, DraggableCustomPropsRowProps>(
    (
      { customProps, index, moveRow, onEdit, onDelete, onMoveUp, onMoveDown, isFirst, isLast, t, style },
      ref,
    ): JSX.Element => {
      const theme = useTheme();
      const [, drop] = useDrop<DragItem>({
        accept: DND_TYPE,
        hover(item, monitor) {
          if (!(ref && typeof ref !== "function" && ref.current)) return;
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

      const fieldType = FieldTypes[customProps.type];
      const TypeIcon = fieldType.icon;

      return (
        <TableRow ref={setRowRef} hover style={{ opacity: isDragging ? 0.5 : 1, cursor: "move", ...style }}>
          <NoWrapCell>
            <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
              {customProps.icon && (
                <Icon icon={customProps.icon} width={20} height={20} color={theme.palette.action.active} />
              )}
              {t(customProps.name, { ns: "application" })}
            </Box>
          </NoWrapCell>
          <NoWrapCell>
            <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
              <TypeIcon width={20} height={20} sx={{ color: theme.palette.action.active }} />
              {t(fieldType.title)}
            </Box>
          </NoWrapCell>
          <NoWrapCell>
            {getPropsContent(
              {
                props: customProps,
                id: customProps.id,
                value: customProps.default ?? "",
              },
              () => {},
              false,
              true,
            )}
          </NoWrapCell>
          <NoWrapCell>
            <IconButton size="small" onClick={() => onEdit(customProps)}>
              <Edit fontSize="small" />
            </IconButton>
            <IconButton size="small" onClick={() => onDelete(customProps.id)}>
              <Dismiss fontSize="small" />
            </IconButton>
          </NoWrapCell>
          <NoWrapCell>
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
          </NoWrapCell>
        </TableRow>
      );
    },
  ),
);

export default DraggableCustomPropsRow;
