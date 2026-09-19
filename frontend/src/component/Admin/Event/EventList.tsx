import {
  Box,
  Chip,
  Collapse,
  IconButton,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  useMediaQuery,
  useTheme,
} from "@mui/material";
import { KeyboardArrowDown, KeyboardArrowUp } from "@mui/icons-material";
import { useQueryState } from "nuqs";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { adminListEvents } from "../../../api/api";
import { ActivityEvent } from "../../../api/dashboard";
import { AuditLogType } from "../../../api/explorer";
import { useAppDispatch } from "../../../redux/hooks";
import { DenseSelect } from "../../Common/StyledComponents";
import { SquareMenuItem } from "../../FileManager/ContextMenu/ContextMenu";
import PageContainer from "../../Pages/PageContainer";
import PageHeader from "../../Pages/PageHeader";
import TablePagination from "../Common/TablePagination";
import { PageQuery, PageSizeQuery } from "../StoragePolicy/StoragePolicySetting";
import { getEventName } from "../Settings/Event/Events";
import { StyledTableContainerPaper } from "../../Common/StyledComponents";

export const TypeQuery = "type";

const EventRow = ({ event }: { event: ActivityEvent }) => {
  const { t } = useTranslation("dashboard");
  const [open, setOpen] = useState(false);
  const hasExtra = event.extra && Object.keys(event.extra).length > 0;

  return (
    <>
      <TableRow hover>
        <TableCell>
          {hasExtra && (
            <IconButton size="small" onClick={() => setOpen(!open)}>
              {open ? <KeyboardArrowUp /> : <KeyboardArrowDown />}
            </IconButton>
          )}
        </TableCell>
        <TableCell>
          <Chip size="small" label={t(`settings.event.${getEventName(event.type)}`, getEventName(event.type))} />
        </TableCell>
        <TableCell>{event.actor_name || event.actor_id || "-"}</TableCell>
        <TableCell>{event.ip || "-"}</TableCell>
        <TableCell>{event.file_id || "-"}</TableCell>
        <TableCell>{event.share_id || "-"}</TableCell>
        <TableCell>{new Date(event.created_at * 1000).toLocaleString()}</TableCell>
      </TableRow>
      {hasExtra && (
        <TableRow>
          <TableCell colSpan={7} sx={{ py: 0, borderBottom: open ? undefined : 0 }}>
            <Collapse in={open}>
              <Box
                component="pre"
                sx={{
                  m: 1,
                  p: 1,
                  fontSize: 12,
                  bgcolor: "action.hover",
                  borderRadius: 1,
                  overflow: "auto",
                  maxHeight: 240,
                }}
              >
                {JSON.stringify(event.extra, null, 2)}
              </Box>
            </Collapse>
          </TableCell>
        </TableRow>
      )}
    </>
  );
};

const EventList = () => {
  const { t } = useTranslation("dashboard");
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const dispatch = useAppDispatch();
  const [loading, setLoading] = useState(true);
  const [events, setEvents] = useState<ActivityEvent[]>([]);
  const [count, setCount] = useState(0);
  const [page, setPage] = useQueryState(PageQuery, { defaultValue: "1" });
  const [pageSize, setPageSize] = useQueryState(PageSizeQuery, { defaultValue: "25" });
  const [type, setType] = useQueryState(TypeQuery, { defaultValue: "" });

  const pageInt = parseInt(page) ?? 1;
  const pageSizeInt = parseInt(pageSize) ?? 25;
  const typeInt = parseInt(type) || 0;

  useEffect(() => {
    setLoading(true);
    dispatch(adminListEvents({ page: pageInt, pageSize: pageSizeInt, type: typeInt }))
      .then((res) => {
        setEvents(res.events);
        setCount(res.total);
      })
      .finally(() => setLoading(false));
  }, [page, pageSize, type]);

  return (
    <PageContainer>
      <PageHeader title={t("nav.events")} />
      <Stack direction="row" spacing={1} sx={{ mb: 1 }}>
        <DenseSelect
          size="small"
          value={type}
          displayEmpty
          onChange={(e) => {
            setType(e.target.value as string);
            setPage("1");
          }}
          sx={{ minWidth: 220 }}
        >
          <SquareMenuItem value="">{t("event.allEventTypes")}</SquareMenuItem>
          {Object.entries(AuditLogType).map(([name, id]) => (
            <SquareMenuItem key={id} value={String(id)}>
              {t(`settings.event.${name}`, name)}
            </SquareMenuItem>
          ))}
        </DenseSelect>
      </Stack>
      <StyledTableContainerPaper>
        <Table size={isMobile ? "small" : "medium"}>
          <TableHead>
            <TableRow>
              <TableCell width={40} />
              <TableCell>{t("event.event")}</TableCell>
              <TableCell>{t("event.initiator")}</TableCell>
              <TableCell>{t("event.ip")}</TableCell>
              <TableCell>{t("event.linkedFile")}</TableCell>
              <TableCell>{t("event.linkedShare")}</TableCell>
              <TableCell>{t("event.datetime")}</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {events.map((event) => (
              <EventRow key={event.id} event={event} />
            ))}
            {!loading && events.length === 0 && (
              <TableRow>
                <TableCell colSpan={7} align="center">
                  {t("event.noEvents")}
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
    </PageContainer>
  );
};

export default EventList;
