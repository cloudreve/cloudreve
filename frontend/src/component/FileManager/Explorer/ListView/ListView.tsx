import { Box, useMediaQuery, useTheme } from "@mui/material";
import React, { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useAppDispatch, useAppSelector } from "../../../../redux/hooks.ts";
import { applyListColumns } from "../../../../redux/thunks/filemanager.ts";
import { FmIndexContext } from "../../FmIndexContext.tsx";
import { SearchLimitReached } from "../EmptyFileList.tsx";
import { getColumnTypeDefaults, ListViewColumn, ListViewColumnSetting } from "./Column.tsx";
import ListBody from "./ListBody.tsx";
import ListHeader from "./ListHeader.tsx";

const ListView = React.forwardRef(
  (
    {
      ...rest
    }: {
      [key: string]: any;
    },
    ref,
  ) => {
    const { t } = useTranslation("application");
    const theme = useTheme();
    const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
    const dispatch = useAppDispatch();
    const fmIndex = useContext(FmIndexContext);
    const recursion_limit_reached = useAppSelector((state) => state.fileManager[fmIndex].list?.recursion_limit_reached);
    const columnSetting = useAppSelector((state) => state.fileManager[fmIndex].listViewColumns);
    const customProps = useAppSelector((state) => state.siteConfig.explorer?.config?.custom_props);

    const [columns, setColumns] = useState<ListViewColumn[]>(
      columnSetting.map(
        (c): ListViewColumn => ({
          type: c.type,
          width: c.width,
          props: c.props,
          defaults: getColumnTypeDefaults(c, isMobile, customProps),
        }),
      ),
    );

    useEffect(() => {
      setColumns(
        columnSetting.map(
          (c): ListViewColumn => ({
            type: c.type,
            width: c.width,
            props: c.props,
            defaults: getColumnTypeDefaults(c, isMobile, customProps),
          }),
        ),
      );
    }, [columnSetting, customProps]);

    const totalWidth = useMemo(() => {
      return columns.reduce((acc, column) => acc + (column.width ?? column.defaults.width), 0);
    }, [columns]);

    const commitColumnSetting = useCallback(() => {
      let settings: ListViewColumnSetting[] = [];
      setColumns((prev) => {
        settings = [
          ...prev.map((c) => ({
            type: c.type,
            width: c.width,
            props: c.props,
          })),
        ];
        return prev;
      });
      if (settings.length > 0) {
        dispatch(applyListColumns(fmIndex, settings));
      }
    }, [dispatch, setColumns]);

    return (
      <Box
        ref={ref}
        {...rest}
        sx={{
          minWidth: totalWidth + 44,
          height: "100%",
          display: "flex",
          flexDirection: "column",
        }}
      >
        <ListHeader commitColumnSetting={commitColumnSetting} setColumns={setColumns} columns={columns} />
        <ListBody columns={columns} />
        {recursion_limit_reached && (
          <Box sx={{ px: 1, py: 1 }}>
            <SearchLimitReached />
          </Box>
        )}
      </Box>
    );
  },
);

export default ListView;
