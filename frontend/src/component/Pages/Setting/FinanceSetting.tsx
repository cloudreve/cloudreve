import {
  Box,
  Chip,
  Paper,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from "@mui/material";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { getCredit, getCreditTxns } from "../../../api/api.ts";
import { CreditInfo, CreditTxnList } from "../../../api/user.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { sizeToString } from "../../../util/index.ts";
import FacebookCircularProgress from "../../Common/CircularProgress.tsx";
import RedeemCodeInput from "../../Common/Form/RedeemCodeInput.tsx";
import TablePagination from "../../Admin/Common/TablePagination.tsx";
import { NoMarginHelperText, SettingSection, SettingSectionContent } from "../../Admin/Settings/Settings.tsx";
import SettingForm from "./SettingForm.tsx";

const FinanceSetting = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const [info, setInfo] = useState<CreditInfo | undefined>(undefined);
  const [txns, setTxns] = useState<CreditTxnList | undefined>(undefined);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  const loadInfo = () => {
    dispatch(getCredit()).then((res) => setInfo(res));
  };

  useEffect(() => {
    loadInfo();
  }, []);

  useEffect(() => {
    dispatch(getCreditTxns(page, pageSize)).then((res) => setTxns(res));
  }, [page, pageSize]);

  const txnReason = (type: string) => t(`setting.txnType.${type}`, { defaultValue: type });

  if (!info) {
    return (
      <Box sx={{ pt: 20, display: "flex", justifyContent: "center" }}>
        <FacebookCircularProgress />
      </Box>
    );
  }

  return (
    <Stack spacing={5}>
      <SettingSection>
        <Typography variant="h6" gutterBottom>
          {t("setting.finance")}
        </Typography>
        <SettingSectionContent>
          <Stack direction={{ xs: "column", sm: "row" }} spacing={3}>
            <Paper variant="outlined" sx={{ p: 2, minWidth: 200 }}>
              <Typography variant="subtitle2" color="text.secondary">
                {t("setting.creditBalance")}
              </Typography>
              <Typography variant="h4">{info.credits}</Typography>
            </Paper>
            {info.storage_bonus > 0 && (
              <Paper variant="outlined" sx={{ p: 2, minWidth: 200 }}>
                <Typography variant="subtitle2" color="text.secondary">
                  {t("setting.storageBonus")}
                </Typography>
                <Typography variant="h4">{sizeToString(info.storage_bonus)}</Typography>
              </Paper>
            )}
            <Paper variant="outlined" sx={{ p: 2, minWidth: 200 }}>
              <Typography variant="subtitle2" color="text.secondary">
                {t("setting.dlTraffic")}
              </Typography>
              <Typography variant="h4">
                {info.dl_traffic < 0 ? t("setting.dlTrafficUnlimited") : sizeToString(info.dl_traffic)}
              </Typography>
            </Paper>
          </Stack>

          {info.grants.length > 0 && (
            <Box sx={{ mt: 2 }}>
              <Typography variant="subtitle2" color="text.secondary" gutterBottom>
                {t("setting.activeGrants")}
              </Typography>
              <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
                {info.grants.map((g) => (
                  <Chip
                    key={g.id}
                    size="small"
                    label={
                      g.type === "storage"
                        ? `${t("setting.grantStorage")} +${sizeToString(g.amount)}`
                        : `${t("setting.grantGroup")} #${g.amount}`
                    }
                  />
                ))}
              </Stack>
            </Box>
          )}

          <SettingForm title={t("setting.redeemGiftCode")} lgWidth={5}>
            <RedeemCodeInput
              onRedeemed={() => {
                loadInfo();
                dispatch(getCreditTxns(page, pageSize)).then((res) => setTxns(res));
              }}
            />
            <NoMarginHelperText>{t("setting.redeemGiftCodeDes")}</NoMarginHelperText>
          </SettingForm>
        </SettingSectionContent>
      </SettingSection>

      <SettingSection>
        <Typography variant="h6" gutterBottom>
          {t("setting.creditLedger")}
        </Typography>
        <SettingSectionContent>
          <TableContainer component={Paper} variant="outlined">
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>{t("setting.txnChange")}</TableCell>
                  <TableCell>{t("setting.txnTime")}</TableCell>
                  <TableCell>{t("setting.txnReason")}</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {txns?.txns.map((txn) => (
                  <TableRow key={txn.id}>
                    <TableCell sx={{ color: txn.amount >= 0 ? "success.main" : "error.main" }}>
                      {txn.amount >= 0 ? "+" : ""}
                      {txn.amount}
                    </TableCell>
                    <TableCell>{new Date(txn.created_at).toLocaleString()}</TableCell>
                    <TableCell>
                      {txnReason(txn.type)}
                      {txn.des ? ` — ${txn.des}` : ""}
                    </TableCell>
                  </TableRow>
                ))}
                {txns && txns.txns.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={3} align="center">
                      {t("setting.noTxns")}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </TableContainer>
          {txns && txns.total > 0 && (
            <TablePagination
              totalItems={txns.total}
              page={page}
              rowsPerPage={pageSize}
              rowsPerPageOptions={[10, 25, 50]}
              onRowsPerPageChange={(size) => {
                setPage(1);
                setPageSize(size);
              }}
              onChange={(_e, p) => setPage(p)}
            />
          )}
        </SettingSectionContent>
      </SettingSection>
    </Stack>
  );
};

export default FinanceSetting;
