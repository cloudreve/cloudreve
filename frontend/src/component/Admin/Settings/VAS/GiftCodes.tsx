import { Box, Chip, Stack, Table, TableBody, TableContainer, TableHead, TableRow } from "@mui/material";
import { useSnackbar } from "notistack";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useDispatch } from "react-redux";
import { AnyAction } from "redux";
import { ThunkDispatch } from "redux-thunk";
import { NoWrapCell, SecondaryButton, StyledTableContainerPaper } from "../../../Common/StyledComponents.tsx";
import Add from "../../../Icons/Add.tsx";
import TablePagination from "../../Common/TablePagination.tsx";

interface GiftCodesProps {
  storageProductsConfig: string;
  groupProductsConfig: string;
}

// Simplified GiftCode interface for our component use
interface GiftCode {
  id: number;
  code: string;
  used: boolean;
  qyt: number;
}

// Pagination params
interface PaginationParams {
  page: number;
  perPage: number;
  total: number;
}

const GiftCodeStatusChip = ({ used }: { used: boolean }) => {
  const { t } = useTranslation("dashboard");

  return (
    <Chip
      color={used ? "default" : "success"}
      label={used ? t("giftCodes.giftCodeUsed") : t("giftCodes.giftCodeUnused")}
      size="small"
    />
  );
};

const GiftCodes = ({ storageProductsConfig, groupProductsConfig }: GiftCodesProps) => {
  const { t } = useTranslation("dashboard");
  const dispatch = useDispatch<ThunkDispatch<any, any, AnyAction>>();
  const { enqueueSnackbar } = useSnackbar();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [giftCodes, setGiftCodes] = useState<GiftCode[]>([]);
  const [loading, setLoading] = useState(false);

  // Pagination state
  const [pagination, setPagination] = useState<PaginationParams>({
    page: 1,
    perPage: 10,
    total: 0,
  });

  const handleChangeRowsPerPage = (pageSize: number) => {
    setPagination({
      page: 1,
      perPage: pageSize,
      total: pagination.total,
    });
  };

  return (
    <Stack spacing={2}>
      <Box>
        <SecondaryButton variant="contained" startIcon={<Add />} onClick={() => setDialogOpen(true)}>
          {t("giftCodes.generateGiftCodes")}
        </SecondaryButton>
      </Box>

      <StyledTableContainerPaper>
        <TableContainer>
          <Table size="small">
            <TableHead>
              <TableRow>
                <NoWrapCell>#</NoWrapCell>
                <NoWrapCell>{t("giftCodes.giftCodeProduct")}</NoWrapCell>
                <NoWrapCell>{t("giftCodes.giftCodeAmount")}</NoWrapCell>
                <NoWrapCell>{t("giftCodes.giftCode")}</NoWrapCell>
                <NoWrapCell>{t("giftCodes.giftCodeStatus")}</NoWrapCell>
                <NoWrapCell>{t("giftCodes.giftCodeUsedBy")}</NoWrapCell>
                <NoWrapCell align="right"></NoWrapCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {giftCodes.length === 0 && !loading && (
                <TableRow>
                  <NoWrapCell colSpan={6} align="center">
                    {t("giftCodes.noGiftCodes")}
                  </NoWrapCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </TableContainer>
        {pagination?.total > 0 && (
          <Box sx={{ px: 1 }}>
            <TablePagination
              totalItems={pagination.total}
              page={pagination.page}
              rowsPerPage={pagination.perPage}
              rowsPerPageOptions={[10, 25, 50, 100]}
              onRowsPerPageChange={handleChangeRowsPerPage}
              onChange={(_, page) => setPagination({ ...pagination, page })}
            />
          </Box>
        )}
      </StyledTableContainerPaper>
    </Stack>
  );
};

export default GiftCodes;
