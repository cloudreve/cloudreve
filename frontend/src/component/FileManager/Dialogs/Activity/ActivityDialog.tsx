import {
  DialogContent,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Typography,
} from "@mui/material";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { getFileActivity } from "../../../../api/api.ts";
import { ActivityEvent } from "../../../../api/dashboard.ts";
import { closeActivityDialog } from "../../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../../redux/hooks.ts";
import AutoHeight from "../../../Common/AutoHeight.tsx";
import { StyledTableContainerPaper } from "../../../Common/StyledComponents.tsx";
import DraggableDialog from "../../../Dialogs/DraggableDialog.tsx";
import { getEventName } from "../../../Admin/Settings/Event/Events.tsx";
import TablePagination from "../../../Admin/Common/TablePagination.tsx";

const ActivityDialog = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();

  const open = useAppSelector((state) => state.globalState.activityDialogOpen);
  const target = useAppSelector((state) => state.globalState.activityDialogFile);

  const [events, setEvents] = useState<ActivityEvent[] | undefined>(undefined);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const pageSize = 20;

  const uri = target?.path;

  useEffect(() => {
    if (!open || !uri) {
      return;
    }
    setEvents(undefined);
    dispatch(getFileActivity(uri, page, pageSize)).then((res) => {
      setEvents(res.events);
      setTotal(res.total);
    });
  }, [open, uri, page]);

  useEffect(() => {
    if (open) {
      setPage(1);
    }
  }, [open]);

  return (
    <DraggableDialog
      title={t("application:fileManager.activity")}
      loading={events === undefined}
      dialogProps={{
        open: open ?? false,
        onClose: () => dispatch(closeActivityDialog()),
        fullWidth: true,
        maxWidth: "md",
      }}
    >
      <DialogContent>
        <AutoHeight>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            {t("application:fileManager.activityDes", { name: target?.name })}
          </Typography>
          <StyledTableContainerPaper>
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>{t("dashboard:event.event")}</TableCell>
                  <TableCell>{t("dashboard:event.initiator")}</TableCell>
                  <TableCell>{t("dashboard:event.ip")}</TableCell>
                  <TableCell>{t("dashboard:event.datetime")}</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {events?.map((event) => (
                  <TableRow key={event.id} hover>
                    <TableCell>{t(`dashboard:settings.event.${getEventName(event.type)}`, getEventName(event.type))}</TableCell>
                    <TableCell>{event.actor_name || event.actor_id || "-"}</TableCell>
                    <TableCell>{event.ip || "-"}</TableCell>
                    <TableCell>{new Date(event.created_at * 1000).toLocaleString()}</TableCell>
                  </TableRow>
                ))}
                {events?.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={4} align="center">
                      {t("dashboard:event.noEvents")}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </StyledTableContainerPaper>
          {total > pageSize && (
            <TablePagination
              page={page}
              totalItems={total}
              rowsPerPage={pageSize}
              onChange={(_e, value) => setPage(value)}
            />
          )}
        </AutoHeight>
      </DialogContent>
    </DraggableDialog>
  );
};

export default ActivityDialog;
