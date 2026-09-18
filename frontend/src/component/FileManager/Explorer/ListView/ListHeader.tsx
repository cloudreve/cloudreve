import Column, { ListViewColumn } from "./Column.tsx";
import { Box, Fade, IconButton, Tooltip } from "@mui/material";
import { useCallback, useContext, useMemo, useRef, useState } from "react";
import { FmIndexContext } from "../../FmIndexContext.tsx";
import { useAppDispatch, useAppSelector } from "../../../../redux/hooks.ts";
import { changeSortOption } from "../../../../redux/thunks/filemanager.ts";
import SessionManager, { UserSettings } from "../../../../session";
import { Add } from "@mui/icons-material";
import { useTranslation } from "react-i18next";
import { setListViewColumnSettingDialog } from "../../../../redux/globalStateSlice.ts";

export interface ListHeaderProps {
  columns: ListViewColumn[];
  setColumns: React.Dispatch<React.SetStateAction<ListViewColumn[]>>;
  commitColumnSetting: () => void;
}

export interface ResizeProps {
  index: number;
  startX: number;
}

const ListHeader = ({ setColumns, commitColumnSetting, columns }: ListHeaderProps) => {
  const dispatch = useAppDispatch();
  const { t } = useTranslation();
  const [showDivider, setShowDivider] = useState(false);
  const resizeProps = useRef<ResizeProps | undefined>();
  const startResizing = (props: ResizeProps) => {
    resizeProps.current = props;
    document.body.style.cursor = "col-resize";
    window.addEventListener("mousemove", onMouseMove);
    window.addEventListener("mouseup", onMouseUp);
  };

  const onMouseMove = useCallback(
    (e: MouseEvent) => {
      if (!resizeProps.current) {
        return;
      }
      const column = columns[resizeProps.current.index];
      const currentWidth = column.width ?? column.defaults.width;
      const minWidth = column.defaults.minWidth ?? 100;
      const newWidth = Math.max(minWidth, currentWidth + (e.clientX - resizeProps.current.startX));
      setColumns((prev) =>
        prev.map((c, index) => (index === resizeProps.current?.index ? { ...c, width: newWidth } : c)),
      );
    },
    [columns, setColumns],
  );

  const onMouseUp = useCallback(() => {
    document.body.style.removeProperty("cursor");
    window.removeEventListener("mousemove", onMouseMove);
    window.removeEventListener("mouseup", onMouseUp);
    commitColumnSetting();
  }, [onMouseMove, commitColumnSetting]);

  const fmIndex = useContext(FmIndexContext);
  const orderMethodOptions = useAppSelector((state) => state.fileManager[fmIndex].list?.props.order_by_options);
  const orderDirectionOption = useAppSelector(
    (state) => state.fileManager[fmIndex].list?.props.order_direction_options,
  );
  const sortBy = useAppSelector((state) => state.fileManager[fmIndex].sortBy);
  const sortDirection = useAppSelector((state) => state.fileManager[fmIndex].sortDirection);

  const allAvailableSortOptions = useMemo((): {
    [key: string]: boolean;
  } => {
    if (!orderMethodOptions || !orderDirectionOption) return {};
    const res: { [key: string]: boolean } = {};
    orderMethodOptions.forEach((method) => {
      // make sure orderDirectionOption contains both asc and desc
      if (orderDirectionOption.includes("asc") && orderDirectionOption.includes("desc")) {
        res[method] = true;
      }
    });
    return res;
  }, [orderMethodOptions, sortDirection]);

  const setSortBy = useCallback(
    (order_by: string, order_direction: string) => {
      dispatch(changeSortOption(fmIndex, order_by, order_direction));
    },
    [dispatch, fmIndex],
  );

  return (
    <Box
      onMouseEnter={() => setShowDivider(true)}
      onMouseLeave={() => setShowDivider(false)}
      sx={{
        display: "flex",
        borderBottom: (theme) => `1px solid ${theme.palette.divider}`,
      }}
    >
      {columns.map((column, index) => (
        <Column
          startResizing={startResizing}
          index={index}
          showDivider={showDivider}
          key={index}
          column={column}
          setSortBy={setSortBy}
          sortable={!!column.defaults.order_by && allAvailableSortOptions[column.defaults.order_by]}
          sortDirection={sortBy && sortBy === column.defaults.order_by ? sortDirection : undefined}
        />
      ))}
      <Fade in={showDivider}>
        <Box
          sx={{
            display: "flex",
            alignItems: "center",
            mr: 1,
          }}
        >
          <Tooltip title={t("fileManager.addColumn")}>
            <IconButton onClick={() => dispatch(setListViewColumnSettingDialog(true))} sx={{ ml: 1 }} size={"small"}>
              <Add
                sx={{
                  width: "18px",
                  height: "18px",
                }}
              />
            </IconButton>
          </Tooltip>
        </Box>
      </Fade>
    </Box>
  );
};

export default ListHeader;
