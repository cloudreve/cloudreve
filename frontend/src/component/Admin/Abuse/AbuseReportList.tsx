import {
  Box,
  Button,
  Chip,
  Collapse,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  IconButton,
  Stack,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TextField,
  useMediaQuery,
  useTheme,
} from "@mui/material";
import { KeyboardArrowDown, KeyboardArrowUp } from "@mui/icons-material";
import { enqueueSnackbar } from "notistack";
import { useQueryState } from "nuqs";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { adminListAbuseReports, adminUpdateAbuseReport } from "../../../api/api";
import { AbuseReport, AbuseReportStatus } from "../../../api/dashboard";
import { useAppDispatch } from "../../../redux/hooks";
import { DefaultCloseAction } from "../../Common/Snackbar/snackbar";
import { DenseSelect, StyledTableContainerPaper } from "../../Common/StyledComponents";
import { SquareMenuItem } from "../../FileManager/ContextMenu/ContextMenu";
import PageContainer from "../../Pages/PageContainer";
import PageHeader from "../../Pages/PageHeader";
import TablePagination from "../Common/TablePagination";
import { PageQuery, PageSizeQuery } from "../StoragePolicy/StoragePolicySetting";

export const StatusQuery = "status";

const statusColor = (status: AbuseReportStatus): "warning" | "success" | "default" => {
  switch (status) {
    case "open":
      return "warning";
    case "resolved":
      return "success";
    default:
      return "default";
  }
};

const ReviewDialog = ({
  report,
  status,
  onClose,
  onDone,
}: {
  report?: AbuseReport;
  status: AbuseReportStatus;
  onClose: () => void;
  onDone: () => void;
}) => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const [note, setNote] = useState("");
  const [blockShare, setBlockShare] = useState(false);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setNote("");
    setBlockShare(false);
  }, [report, status]);

  const onSubmit = () => {
    if (!report) {
      return;
    }
    setLoading(true);
    dispatch(
      adminUpdateAbuseReport(report.id, {
        status,
        admin_note: note || undefined,
        block_share: blockShare,
      }),
    )
      .then(() => {
        enqueueSnackbar({
          message: t("abuseReport.reviewSuccess"),
          variant: "success",
          action: DefaultCloseAction,
        });
        onDone();
      })
      .finally(() => setLoading(false));
  };

  return (
    <Dialog open={!!report} onClose={onClose} fullWidth maxWidth="sm">
      <DialogTitle>{t(`abuseReport.markAs.${status}`)}</DialogTitle>
      <DialogContent>
        <TextField
          fullWidth
          multiline
          minRows={2}
          label={t("abuseReport.adminNote")}
          value={note}
          onChange={(e) => setNote(e.target.value)}
          inputProps={{ maxLength: 2000 }}
          sx={{ mt: 1 }}
        />
        {report?.target_type === "share" && (
          <FormControlLabel
            sx={{ mt: 1 }}
            control={<Switch checked={blockShare} onChange={(e) => setBlockShare(e.target.checked)} />}
            label={t("abuseReport.blockShare")}
          />
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t("common:cancel")}</Button>
        <Button variant="contained" onClick={onSubmit} disabled={loading}>
          {t("common:ok")}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

const ReportRow = ({ report, onReview }: { report: AbuseReport; onReview: (status: AbuseReportStatus) => void }) => {
  const { t } = useTranslation("dashboard");
  const [open, setOpen] = useState(false);
  const reasonOptions = t("application:vas.reportReasonOptions", { returnObjects: true }) as string[];

  return (
    <>
      <TableRow hover>
        <TableCell>
          <IconButton size="small" onClick={() => setOpen(!open)}>
            {open ? <KeyboardArrowUp /> : <KeyboardArrowDown />}
          </IconButton>
        </TableCell>
        <TableCell>
          {report.target_type === "share"
            ? `${t("abuseReport.shareID")} #${report.target_id}`
            : `${t("abuseReport.reportedUserID")} #${report.target_id}`}
        </TableCell>
        <TableCell>{reasonOptions[report.reason] ?? report.reason}</TableCell>
        <TableCell>{report.reporter_email || report.reporter_id || "-"}</TableCell>
        <TableCell>
          <Chip size="small" color={statusColor(report.status)} label={t(`abuseReport.status.${report.status}`)} />
        </TableCell>
        <TableCell>{new Date(report.created_at * 1000).toLocaleString()}</TableCell>
        <TableCell align="right">
          {report.status === "open" && (
            <Stack direction="row" spacing={1} justifyContent="flex-end">
              <Button size="small" onClick={() => onReview("resolved")}>
                {t("abuseReport.resolve")}
              </Button>
              <Button size="small" onClick={() => onReview("dismissed")}>
                {t("abuseReport.dismiss")}
              </Button>
            </Stack>
          )}
        </TableCell>
      </TableRow>
      <TableRow>
        <TableCell colSpan={7} sx={{ py: 0, borderBottom: open ? undefined : 0 }}>
          <Collapse in={open}>
            <Box sx={{ m: 1 }}>
              {report.description && (
                <Box sx={{ mb: 1, whiteSpace: "pre-wrap" }}>{report.description}</Box>
              )}
              {report.admin_note && (
                <Box sx={{ color: "text.secondary", fontSize: 13 }}>
                  {t("abuseReport.adminNote")}: {report.admin_note}
                </Box>
              )}
            </Box>
          </Collapse>
        </TableCell>
      </TableRow>
    </>
  );
};

const AbuseReportList = () => {
  const { t } = useTranslation("dashboard");
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const dispatch = useAppDispatch();
  const [loading, setLoading] = useState(true);
  const [reports, setReports] = useState<AbuseReport[]>([]);
  const [count, setCount] = useState(0);
  const [page, setPage] = useQueryState(PageQuery, { defaultValue: "1" });
  const [pageSize, setPageSize] = useQueryState(PageSizeQuery, { defaultValue: "25" });
  const [status, setStatus] = useQueryState(StatusQuery, { defaultValue: "open" });
  const [review, setReview] = useState<{ report: AbuseReport; status: AbuseReportStatus } | undefined>(undefined);

  const pageInt = parseInt(page) ?? 1;
  const pageSizeInt = parseInt(pageSize) ?? 25;

  const load = () => {
    setLoading(true);
    dispatch(adminListAbuseReports({ page: pageInt, pageSize: pageSizeInt, status: status || undefined }))
      .then((res) => {
        setReports(res.reports);
        setCount(res.total);
      })
      .finally(() => setLoading(false));
  };

  useEffect(load, [page, pageSize, status]);

  return (
    <PageContainer>
      <PageHeader title={t("nav.abuseReport")} />
      <Stack direction="row" spacing={1} sx={{ mb: 1 }}>
        <DenseSelect
          size="small"
          value={status}
          displayEmpty
          onChange={(e) => {
            setStatus(e.target.value as string);
            setPage("1");
          }}
          sx={{ minWidth: 180 }}
        >
          <SquareMenuItem value="open">{t("abuseReport.status.open")}</SquareMenuItem>
          <SquareMenuItem value="resolved">{t("abuseReport.status.resolved")}</SquareMenuItem>
          <SquareMenuItem value="dismissed">{t("abuseReport.status.dismissed")}</SquareMenuItem>
          <SquareMenuItem value="">{t("abuseReport.status.all")}</SquareMenuItem>
        </DenseSelect>
      </Stack>
      <StyledTableContainerPaper>
        <Table size={isMobile ? "small" : "medium"}>
          <TableHead>
            <TableRow>
              <TableCell width={40} />
              <TableCell>{t("abuseReport.target")}</TableCell>
              <TableCell>{t("abuseReport.reason")}</TableCell>
              <TableCell>{t("abuseReport.reporter")}</TableCell>
              <TableCell>{t("abuseReport.statusLabel")}</TableCell>
              <TableCell>{t("abuseReport.reportedAt")}</TableCell>
              <TableCell align="right">{t("abuseReport.actions")}</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {reports.map((report) => (
              <ReportRow
                key={report.id}
                report={report}
                onReview={(s) => setReview({ report, status: s })}
              />
            ))}
            {!loading && reports.length === 0 && (
              <TableRow>
                <TableCell colSpan={7} align="center">
                  {t("abuseReport.noReports")}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </StyledTableContainerPaper>
      <TablePagination
        page={pageInt}
        totalItems={count}
        rowsPerPage={pageSizeInt}
        onRowsPerPageChange={(size) => setPageSize(size.toString())}
        onChange={(_e, value) => setPage(value.toString())}
      />
      <ReviewDialog
        report={review?.report}
        status={review?.status ?? "resolved"}
        onClose={() => setReview(undefined)}
        onDone={() => {
          setReview(undefined);
          load();
        }}
      />
    </PageContainer>
  );
};

export default AbuseReportList;
