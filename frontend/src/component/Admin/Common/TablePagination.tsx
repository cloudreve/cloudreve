import {
  Box,
  ListItemText,
  Pagination,
  PaginationProps,
  SelectChangeEvent,
  useMediaQuery,
  useTheme,
} from "@mui/material";
import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { DenseSelect } from "../../Common/StyledComponents";
import { SquareMenuItem } from "../../FileManager/ContextMenu/ContextMenu";

export interface TablePaginationProps extends PaginationProps {
  rowsPerPageOptions?: number[];
  rowsPerPage?: number;
  onRowsPerPageChange?: (pageSize: number) => void;
  page: number;
  totalItems: number;
  onChange: (event: React.ChangeEvent<unknown>, value: number) => void;
}

export const TablePagination = ({
  rowsPerPageOptions = [5, 10, 25, 50],
  rowsPerPage = 5,
  onRowsPerPageChange,
  page,
  onChange,
  totalItems,
  ...props
}: TablePaginationProps) => {
  const { t } = useTranslation("dashboard");
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));

  const onDenseSelectChange = (e: SelectChangeEvent<unknown>) => {
    onRowsPerPageChange?.(e.target.value as number);
  };

  useEffect(() => {
    if ((page - 1) * rowsPerPage >= totalItems) {
      onChange({} as React.ChangeEvent<unknown>, Math.ceil(totalItems / rowsPerPage));
    }
  }, [rowsPerPage, totalItems]);

  return (
    <Box
      sx={{
        py: 1,
        display: "flex",
        flexDirection: isMobile ? "column" : "row",
        justifyContent: "space-between",
        alignItems: isMobile ? "start" : "center",
        gap: 1,
      }}
    >
      <Pagination
        size={isMobile ? "small" : "medium"}
        count={Math.ceil(totalItems / rowsPerPage)}
        page={page}
        onChange={onChange}
        {...props}
      />
      <DenseSelect
        variant="filled"
        value={rowsPerPage}
        onChange={onDenseSelectChange}
        renderValue={(value) => (
          <ListItemText primary={t("settings.perPage", { num: value })} slotProps={{ primary: { variant: "body2" } }} />
        )}
      >
        {rowsPerPageOptions.map((option) => (
          <SquareMenuItem key={option} value={option}>
            <ListItemText
              primary={t("settings.perPage", { num: option })}
              slotProps={{ primary: { variant: "body2" } }}
            />
          </SquareMenuItem>
        ))}
      </DenseSelect>
    </Box>
  );
};

export default TablePagination;
