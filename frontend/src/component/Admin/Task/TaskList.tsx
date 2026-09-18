import { Delete } from "@mui/icons-material";
import {
  Badge,
  Box,
  Button,
  Checkbox,
  Container,
  Divider,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TableSortLabel,
  useMediaQuery,
  useTheme,
} from "@mui/material";
import { bindPopover, bindTrigger, usePopupState } from "material-ui-popup-state/hooks";
import { useQueryState } from "nuqs";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { batchDeleteTasks, getTaskList } from "../../../api/api";
import { AdminListService, Task } from "../../../api/dashboard";
import { useAppDispatch } from "../../../redux/hooks";
import { confirmOperation } from "../../../redux/thunks/dialog";
import { NoWrapTableCell, SecondaryButton, StyledTableContainerPaper } from "../../Common/StyledComponents";
import ArrowSync from "../../Icons/ArrowSync";
import Broom from "../../Icons/Broom";
import Filter from "../../Icons/Filter";
import PageContainer from "../../Pages/PageContainer";
import PageHeader from "../../Pages/PageHeader";
import TablePagination from "../Common/TablePagination";
import EntityDialog from "../Entity/EntityDialog/EntityDialog";
import FileDialog from "../File/FileDialog/FileDialog";
import { OrderByQuery, OrderDirectionQuery, PageQuery, PageSizeQuery } from "../StoragePolicy/StoragePolicySetting";
import UserDialog from "../User/UserDialog/UserDialog";
import TaskCleanupDialog from "./TaskCleanupDialog";
import TaskDialog from "./TaskDialog/TaskDialog";
import TaskFilterPopover from "./TaskFilterPopover";
import TaskRow from "./TaskRow";

export const UserQuery = "user";
export const TypeQuery = "type";
export const StatusQuery = "status";
export const CorrelationIDQuery = "correlation_id";

const TaskList = () => {
  const { t } = useTranslation("dashboard");
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const dispatch = useAppDispatch();
  const [loading, setLoading] = useState(true);
  const [tasks, setTasks] = useState<Task[]>([]);
  const [page, setPage] = useQueryState(PageQuery, { defaultValue: "1" });
  const [pageSize, setPageSize] = useQueryState(PageSizeQuery, {
    defaultValue: "10",
  });
  const [orderBy, setOrderBy] = useQueryState(OrderByQuery, {
    defaultValue: "",
  });
  const [orderDirection, setOrderDirection] = useQueryState(OrderDirectionQuery, { defaultValue: "desc" });
  const [user, setUser] = useQueryState(UserQuery, { defaultValue: "" });
  const [type, setType] = useQueryState(TypeQuery, { defaultValue: "" });
  const [status, setStatus] = useQueryState(StatusQuery, { defaultValue: "" });
  const [correlationID, setCorrelationID] = useQueryState(CorrelationIDQuery, { defaultValue: "" });

  const [count, setCount] = useState(0);
  const [selected, setSelected] = useState<readonly number[]>([]);
  const filterPopupState = usePopupState({
    variant: "popover",
    popupId: "taskFilterPopover",
  });

  const [userDialogOpen, setUserDialogOpen] = useState(false);
  const [userDialogID, setUserDialogID] = useState<number | undefined>(undefined);
  const [openEntity, setOpenEntity] = useState<number | undefined>(undefined);
  const [openEntityDialogOpen, setOpenEntityDialogOpen] = useState(false);
  const [openFileID, setOpenFileID] = useState<number | undefined>(undefined);
  const [openFileDialogOpen, setOpenFileDialogOpen] = useState(false);
  const [openTask, setOpenTask] = useState<number | undefined>(undefined);
  const [openTaskDialogOpen, setOpenTaskDialogOpen] = useState(false);
  const [deleteLoading, setDeleteLoading] = useState(false);
  const [cleanupDialogOpen, setCleanupDialogOpen] = useState(false);

  const pageInt = parseInt(page) ?? 1;
  const pageSizeInt = parseInt(pageSize) ?? 10;

  const clearFilters = useCallback(() => {
    setUser("");
    setType("");
    setStatus("");
    setCorrelationID("");
  }, [setUser, setType, setStatus, setCorrelationID]);

  useEffect(() => {
    fetchTasks();
  }, [page, pageSize, orderBy, orderDirection, user, type, status, correlationID]);

  const fetchTasks = () => {
    setLoading(true);
    setSelected([]);

    const params: AdminListService = {
      page: pageInt,
      page_size: pageSizeInt,
      order_by: orderBy ?? "",
      order_direction: orderDirection ?? "desc",
      conditions: {
        task_status: status,
        task_user_id: user,
        task_type: type,
        task_correlation_id: correlationID,
      },
    };

    dispatch(getTaskList(params))
      .then((res) => {
        setTasks(res.tasks);
        setPageSize(res.pagination.page_size.toString());
        setCount(res.pagination.total_items ?? 0);
        setLoading(false);
      })
      .finally(() => {
        setLoading(false);
      });
  };

  const handleDelete = () => {
    setDeleteLoading(true);
    dispatch(confirmOperation(t("task.confirmBatchDelete", { num: selected.length })))
      .then(() => {
        dispatch(batchDeleteTasks({ ids: Array.from(selected) }))
          .then(() => {
            fetchTasks();
          })
          .finally(() => {
            setDeleteLoading(false);
          });
        setDeleteLoading(false);
      })
      .finally(() => {
        setDeleteLoading(false);
      });
  };

  const handleSelectAllClick = (event: React.ChangeEvent<HTMLInputElement>) => {
    if (event.target.checked) {
      const newSelected = tasks.map((n) => n.id);
      setSelected(newSelected);
      return;
    }
    setSelected([]);
  };

  const handleSelect = useCallback(
    (id: number) => {
      const selectedIndex = selected.indexOf(id);
      let newSelected: readonly number[] = [];

      if (selectedIndex === -1) {
        newSelected = newSelected.concat(selected, id);
      } else if (selectedIndex === 0) {
        newSelected = newSelected.concat(selected.slice(1));
      } else if (selectedIndex === selected.length - 1) {
        newSelected = newSelected.concat(selected.slice(0, -1));
      } else if (selectedIndex > 0) {
        newSelected = newSelected.concat(selected.slice(0, selectedIndex), selected.slice(selectedIndex + 1));
      }
      setSelected(newSelected);
    },
    [selected],
  );

  const orderById = orderBy === "id" || orderBy === "";
  const direction = orderDirection as "asc" | "desc";
  const onSortClick = (field: string) => () => {
    const alreadySorted = orderBy === field || (field === "id" && orderById);
    setOrderBy(field);
    setOrderDirection(alreadySorted ? (direction === "asc" ? "desc" : "asc") : "asc");
  };

  const hasActiveFilters = useMemo(() => {
    return !!(status || user || type);
  }, [status, user, type]);

  const handleUserDialogOpen = (id: number) => {
    setUserDialogID(id);
    setUserDialogOpen(true);
  };

  const handleOpenEntity = (entityID: number) => {
    setOpenEntity(entityID);
    setOpenEntityDialogOpen(true);
  };

  const handleOpenFile = (fileID: number) => {
    setOpenFileID(fileID);
    setOpenFileDialogOpen(true);
  };

  const handleOpenTask = (taskID: number) => {
    setOpenTask(taskID);
    setOpenTaskDialogOpen(true);
  };

  return (
    <PageContainer>
      <UserDialog open={userDialogOpen} onClose={() => setUserDialogOpen(false)} userID={userDialogID} />
      <EntityDialog open={openEntityDialogOpen} onClose={() => setOpenEntityDialogOpen(false)} entityID={openEntity} />
      <FileDialog open={openFileDialogOpen} onClose={() => setOpenFileDialogOpen(false)} fileID={openFileID} />
      <TaskDialog open={openTaskDialogOpen} onClose={() => setOpenTaskDialogOpen(false)} taskID={openTask} />
      <TaskCleanupDialog
        open={cleanupDialogOpen}
        onClose={() => setCleanupDialogOpen(false)}
        onCleanupComplete={fetchTasks}
      />
      <Container maxWidth="xl">
        <PageHeader title={t("dashboard:nav.tasks")} />
        <Stack direction="row" spacing={1} sx={{ mb: 2 }}>
          <TaskFilterPopover
            {...bindPopover(filterPopupState)}
            status={status}
            setStatus={setStatus}
            user={user}
            setUser={setUser}
            correlationID={correlationID}
            setCorrelationID={setCorrelationID}
            type={type}
            setType={setType}
            clearFilters={clearFilters}
          />

          <SecondaryButton onClick={fetchTasks} disabled={loading} variant={"contained"} startIcon={<ArrowSync />}>
            {t("node.refresh")}
          </SecondaryButton>

          <Badge color="primary" variant="dot" invisible={!hasActiveFilters}>
            <SecondaryButton startIcon={<Filter />} variant="contained" {...bindTrigger(filterPopupState)}>
              {t("user.filter")}
            </SecondaryButton>
          </Badge>

          <SecondaryButton startIcon={<Broom />} variant="contained" onClick={() => setCleanupDialogOpen(true)}>
            {t("event.cleanup")}
          </SecondaryButton>

          {selected.length > 0 && !isMobile && (
            <>
              <Divider orientation="vertical" flexItem />
              <Button startIcon={<Delete />} variant="contained" color="error" onClick={handleDelete}>
                {t("task.deleteXTasks", { num: selected.length })}
              </Button>
            </>
          )}
        </Stack>
        {isMobile && selected.length > 0 && (
          <Stack direction="row" spacing={1} sx={{ mb: 2 }}>
            <Button startIcon={<Delete />} variant="contained" color="error" onClick={handleDelete}>
              {t("task.deleteXTasks", { num: selected.length })}
            </Button>
          </Stack>
        )}
        <TableContainer component={StyledTableContainerPaper} sx={{ mt: 2 }}>
          <Table size="small" stickyHeader sx={{ width: "100%", tableLayout: "fixed" }}>
            <TableHead>
              <TableRow>
                <TableCell padding="checkbox" sx={{ width: "36px!important" }} width={50}>
                  <Checkbox
                    size="small"
                    indeterminate={selected.length > 0 && selected.length < tasks.length}
                    checked={tasks.length > 0 && selected.length === tasks.length}
                    onChange={handleSelectAllClick}
                  />
                </TableCell>
                <NoWrapTableCell width={80}>
                  <TableSortLabel
                    active={orderById}
                    direction={orderById ? direction : "asc"}
                    onClick={onSortClick("id")}
                  >
                    {t("group.#")}
                  </TableSortLabel>
                </NoWrapTableCell>
                <NoWrapTableCell width={400}>{t("task.content")}</NoWrapTableCell>
                <NoWrapTableCell width={120}>{t("task.status")}</NoWrapTableCell>
                <NoWrapTableCell width={150}>{t("file.creator")}</NoWrapTableCell>
                <NoWrapTableCell width={100}>{t("task.node")}</NoWrapTableCell>
                <NoWrapTableCell width={180}>
                  <TableSortLabel
                    active={orderBy === "created_at"}
                    direction={orderBy === "created_at" ? direction : "asc"}
                    onClick={onSortClick("created_at")}
                  >
                    {t("file.createdAt")}
                  </TableSortLabel>
                </NoWrapTableCell>
                <NoWrapTableCell width={50}></NoWrapTableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {!loading &&
                tasks.map((task) => (
                  <TaskRow
                    deleting={deleteLoading}
                    key={task.id}
                    task={task}
                    onDelete={fetchTasks}
                    selected={selected.indexOf(task.id) !== -1}
                    onSelect={handleSelect}
                    openUserDialog={handleUserDialogOpen}
                    openEntity={handleOpenEntity}
                    openFile={handleOpenFile}
                    onDetails={handleOpenTask}
                  />
                ))}
              {loading &&
                tasks.length > 0 &&
                tasks.slice(0, 10).map((task) => <TaskRow key={`loading-${task.id}`} loading={true} />)}
              {loading &&
                tasks.length === 0 &&
                Array.from(Array(10)).map((_, index) => <TaskRow key={`loading-${index}`} loading={true} />)}
            </TableBody>
          </Table>
        </TableContainer>
        {count > 0 && (
          <Box sx={{ mt: 1 }}>
            <TablePagination
              page={pageInt}
              totalItems={count}
              rowsPerPage={pageSizeInt}
              rowsPerPageOptions={[10, 25, 50, 100, 200, 500]}
              onRowsPerPageChange={(value) => setPageSize(value.toString())}
              onChange={(_, value) => setPage(value.toString())}
            />
          </Box>
        )}
      </Container>
    </PageContainer>
  );
};

export default TaskList;
