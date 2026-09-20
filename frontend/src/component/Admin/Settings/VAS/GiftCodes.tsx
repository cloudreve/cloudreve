import {
  Box,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  IconButton,
  InputLabel,
  MenuItem,
  Select,
  Stack,
  Table,
  TableBody,
  TableContainer,
  TableHead,
  TableRow,
} from "@mui/material";
import { useSnackbar } from "notistack";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useAppDispatch } from "../../../../redux/hooks.ts";
import {
  adminCreateGiftCode,
  adminDeleteGiftCode,
  adminListGiftCodes,
  getGroupList,
} from "../../../../api/api.ts";
import { GiftCode, GiftCodeListResponse, GroupEnt } from "../../../../api/dashboard.ts";
import { sizeToString } from "../../../../util/index.ts";
import {
  DenseFilledTextField,
  NoWrapCell,
  SecondaryButton,
  StyledTableContainerPaper,
} from "../../../Common/StyledComponents.tsx";
import Add from "../../../Icons/Add.tsx";
import Dismiss from "../../../Icons/Dismiss.tsx";
import TablePagination from "../../Common/TablePagination.tsx";
import SettingForm from "../../../Pages/Setting/SettingForm.tsx";
import { NoMarginHelperText } from "../Settings.tsx";

interface PaginationParams {
  page: number;
  perPage: number;
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

const GiftCodes = () => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const { enqueueSnackbar } = useSnackbar();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [result, setResult] = useState<GiftCodeListResponse | undefined>(undefined);
  const [groups, setGroups] = useState<GroupEnt[]>([]);
  const [pagination, setPagination] = useState<PaginationParams>({ page: 1, perPage: 10 });

  // Generate form state
  const [genType, setGenType] = useState<"points" | "storage" | "group" | "traffic">("points");
  const [genAmount, setGenAmount] = useState(100);
  const [genGroup, setGenGroup] = useState(0);
  const [genDuration, setGenDuration] = useState(0);
  const [genQty, setGenQty] = useState(1);
  const [genDes, setGenDes] = useState("");
  const [creating, setCreating] = useState(false);

  const load = useCallback(() => {
    dispatch(adminListGiftCodes(pagination.page, pagination.perPage)).then((res) => setResult(res));
  }, [pagination.page, pagination.perPage]);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    if (dialogOpen && groups.length === 0) {
      dispatch(
        getGroupList({ page: 1, page_size: 200, order_by: "id", order_direction: "asc" }),
      ).then((res) => setGroups(res.groups));
    }
  }, [dialogOpen]);

  const onDelete = (id: number) => {
    dispatch(adminDeleteGiftCode(id))
      .then(() => {
        enqueueSnackbar(t("giftCodes.giftCodeDeleted"), { variant: "success" });
        load();
      })
      .catch(() => enqueueSnackbar(t("giftCodes.giftCodeDeleteFailed"), { variant: "error" }));
  };

  const onCreate = () => {
    setCreating(true);
    dispatch(
      adminCreateGiftCode({
        type: genType,
        amount: genType === "group" ? genGroup : genAmount,
        duration: genDuration > 0 ? genDuration : undefined,
        qty: genQty,
        des: genDes || undefined,
      }),
    )
      .then(() => {
        enqueueSnackbar(t("giftCodes.giftCodesGenerated"), { variant: "success" });
        setDialogOpen(false);
        load();
      })
      .catch(() => enqueueSnackbar(t("giftCodes.giftCodeCreateFailed"), { variant: "error" }))
      .finally(() => setCreating(false));
  };

  const amountLabel = (gc: GiftCode) => {
    switch (gc.type) {
      case "points":
        return `${gc.amount}`;
      case "storage":
      case "traffic":
        return sizeToString(gc.amount);
      case "group":
        return `#${gc.amount}`;
    }
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
                <NoWrapCell>{t("giftCodes.giftCodeType")}</NoWrapCell>
                <NoWrapCell>{t("giftCodes.giftCodeAmount")}</NoWrapCell>
                <NoWrapCell>{t("giftCodes.giftCode")}</NoWrapCell>
                <NoWrapCell>{t("giftCodes.giftCodeStatus")}</NoWrapCell>
                <NoWrapCell>{t("giftCodes.giftCodeUsedBy")}</NoWrapCell>
                <NoWrapCell align="right"></NoWrapCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {result?.codes.map((gc) => (
                <TableRow key={gc.id}>
                  <NoWrapCell>{gc.id}</NoWrapCell>
                  <NoWrapCell>{t(`giftCodes.giftCodeType${gc.type.charAt(0).toUpperCase() + gc.type.slice(1)}`)}</NoWrapCell>
                  <NoWrapCell>{amountLabel(gc)}</NoWrapCell>
                  <NoWrapCell sx={{ fontFamily: "monospace" }}>{gc.code}</NoWrapCell>
                  <NoWrapCell>
                    <GiftCodeStatusChip used={!!gc.used_by_id} />
                  </NoWrapCell>
                  <NoWrapCell>{gc.edges?.redeemer?.email ?? "-"}</NoWrapCell>
                  <NoWrapCell align="right">
                    {!gc.used_by_id && (
                      <IconButton size="small" onClick={() => onDelete(gc.id)}>
                        <Dismiss fontSize="small" />
                      </IconButton>
                    )}
                  </NoWrapCell>
                </TableRow>
              ))}
              {(!result || result.codes.length === 0) && (
                <TableRow>
                  <NoWrapCell colSpan={7} align="center">
                    {t("giftCodes.noGiftCodes")}
                  </NoWrapCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </TableContainer>
        {result && result.total > 0 && (
          <Box sx={{ px: 1 }}>
            <TablePagination
              totalItems={result.total}
              page={pagination.page}
              rowsPerPage={pagination.perPage}
              rowsPerPageOptions={[10, 25, 50, 100]}
              onRowsPerPageChange={(size) => setPagination({ page: 1, perPage: size })}
              onChange={(_e, page) => setPagination({ ...pagination, page })}
            />
          </Box>
        )}
      </StyledTableContainerPaper>

      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} maxWidth="xs" fullWidth>
        <DialogTitle>{t("giftCodes.generateGiftCodes")}</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 1 }}>
            <SettingForm title={t("giftCodes.giftCodeProductType")}>
              <FormControl fullWidth size="small">
                <InputLabel>{t("giftCodes.giftCodeProductType")}</InputLabel>
                <Select
                  value={genType}
                  label={t("giftCodes.giftCodeProductType")}
                  onChange={(e) => setGenType(e.target.value as typeof genType)}
                >
                  <MenuItem value="points">{t("giftCodes.giftCodeTypePoints")}</MenuItem>
                  <MenuItem value="storage">{t("giftCodes.giftCodeTypeStorage")}</MenuItem>
                  <MenuItem value="group">{t("giftCodes.giftCodeTypeGroup")}</MenuItem>
                  <MenuItem value="traffic">{t("giftCodes.giftCodeTypeTraffic")}</MenuItem>
                </Select>
              </FormControl>
            </SettingForm>

            {genType !== "group" ? (
              <SettingForm title={genType === "points" ? t("giftCodes.giftCodePointsAmount") : t("giftCodes.giftCodeAmount")}>
                <DenseFilledTextField
                  fullWidth
                  type="number"
                  value={genAmount}
                  onChange={(e) => setGenAmount(parseInt(e.target.value) || 0)}
                />
                <NoMarginHelperText>
                  {genType === "points"
                    ? t("giftCodes.giftCodePointsAmountHelp")
                    : genType === "traffic"
                      ? t("vas.trafficSizeDes")
                      : t("vas.packSizeDes")}
                </NoMarginHelperText>
              </SettingForm>
            ) : (
              <SettingForm title={t("vas.purchasableGroups")}>
                <FormControl fullWidth size="small">
                  <InputLabel>{t("vas.purchasableGroups")}</InputLabel>
                  <Select
                    value={genGroup}
                    label={t("vas.purchasableGroups")}
                    onChange={(e) => setGenGroup(e.target.value as number)}
                  >
                    {groups.map((g) => (
                      <MenuItem key={g.id} value={g.id}>
                        {g.name}
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>
              </SettingForm>
            )}

            {(genType === "storage" || genType === "group") && (
              <SettingForm title={t("giftCodes.duratonTimes")}>
                <DenseFilledTextField
                  fullWidth
                  type="number"
                  value={genDuration}
                  onChange={(e) => setGenDuration(parseInt(e.target.value) || 0)}
                />
                <NoMarginHelperText>{t("giftCodes.duratonTimesDes")}</NoMarginHelperText>
              </SettingForm>
            )}

            <SettingForm title={t("giftCodes.giftCodeQuantity")}>
              <DenseFilledTextField
                fullWidth
                type="number"
                value={genQty}
                onChange={(e) => setGenQty(Math.max(1, Math.min(500, parseInt(e.target.value) || 1)))}
              />
              <NoMarginHelperText>{t("giftCodes.giftCodeQuantityHelp")}</NoMarginHelperText>
            </SettingForm>

            <SettingForm title={t("vas.description")}>
              <DenseFilledTextField fullWidth value={genDes} onChange={(e) => setGenDes(e.target.value)} />
            </SettingForm>
          </Stack>
        </DialogContent>
        <DialogActions>
          <SecondaryButton onClick={() => setDialogOpen(false)}>{t("common:cancel")}</SecondaryButton>
          <SecondaryButton
            variant="contained"
            onClick={onCreate}
            disabled={creating || (genType === "group" ? genGroup <= 0 : genAmount <= 0)}
          >
            {t("common:ok")}
          </SecondaryButton>
        </DialogActions>
      </Dialog>
    </Stack>
  );
};

export default GiftCodes;
