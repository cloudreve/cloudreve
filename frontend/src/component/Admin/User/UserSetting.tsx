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
import { batchDeleteUser, getUserList } from "../../../api/api";
import { User } from "../../../api/dashboard";
import { useAppDispatch } from "../../../redux/hooks";
import { confirmOperation } from "../../../redux/thunks/dialog";
import { NoWrapTableCell, SecondaryButton, StyledTableContainerPaper } from "../../Common/StyledComponents";
import Add from "../../Icons/Add";
import ArrowSync from "../../Icons/ArrowSync";
import Filter from "../../Icons/Filter";
import PageContainer from "../../Pages/PageContainer";
import PageHeader from "../../Pages/PageHeader";
import TablePagination from "../Common/TablePagination";
import { OrderByQuery, OrderDirectionQuery, PageQuery, PageSizeQuery } from "../StoragePolicy/StoragePolicySetting";
import NewUserDialog from "./NewUserDialog";
import UserDialog from "./UserDialog/UserDialog";
import UserFilterPopover from "./UserFilterPopover";
import UserRow from "./UserRow";
export const EmailQuery = "email";
export const NickQuery = "nick";
export const GroupQuery = "group";
export const StatusQuery = "status";

const UserSetting = () => {
  const { t } = useTranslation("dashboard");
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const dispatch = useAppDispatch();
  const [loading, setLoading] = useState(true);
  const [users, setUsers] = useState<User[]>([]);
  const [page, setPage] = useQueryState(PageQuery, { defaultValue: "1" });
  const [pageSize, setPageSize] = useQueryState(PageSizeQuery, {
    defaultValue: "10",
  });
  const [orderBy, setOrderBy] = useQueryState(OrderByQuery, {
    defaultValue: "",
  });
  const [orderDirection, setOrderDirection] = useQueryState(OrderDirectionQuery, { defaultValue: "desc" });
  const [email, setEmail] = useQueryState(EmailQuery, { defaultValue: "" });
  const [nick, setNick] = useQueryState(NickQuery, { defaultValue: "" });
  const [group, setGroup] = useQueryState(GroupQuery, { defaultValue: "" });
  const [status, setStatus] = useQueryState(StatusQuery, { defaultValue: "" });
  const [count, setCount] = useState(0);
  const [selected, setSelected] = useState<readonly number[]>([]);
  const [createNewOpen, setCreateNewOpen] = useState(false);
  const filterPopupState = usePopupState({
    variant: "popover",
    popupId: "userFilterPopover",
  });

  const [userDialogOpen, setUserDialogOpen] = useState(false);
  const [userDialogID, setUserDialogID] = useState<number | undefined>(undefined);
  const [deleteLoading, setDeleteLoading] = useState(false);

  const pageInt = parseInt(page) ?? 1;
  const pageSizeInt = parseInt(pageSize) ?? 11;

  const clearFilters = useCallback(() => {
    setEmail("");
    setNick("");
    setGroup("");
    setStatus("");
  }, [setEmail, setNick, setGroup, setStatus]);

  useEffect(() => {
    fetchUsers();
  }, [page, pageSize, orderBy, orderDirection, email, nick, group, status]);

  const fetchUsers = () => {
    setLoading(true);
    setSelected([]);
    dispatch(
      getUserList({
        page: pageInt,
        page_size: pageSizeInt,
        order_by: orderBy ?? "",
        order_direction: orderDirection ?? "desc",
        conditions: {
          user_email: email,
          user_nick: nick,
          user_group: group,
          user_status: status,
        },
      }),
    )
      .then((res) => {
        setUsers(res.users);
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
    dispatch(confirmOperation(t("user.confirmBatchDelete", { num: selected.length })))
      .then(() => {
        dispatch(batchDeleteUser({ ids: Array.from(selected) }))
          .then(() => {
            fetchUsers();
          })
          .finally(() => {
            setDeleteLoading(false);
          });
      })
      .finally(() => {
        setDeleteLoading(false);
      });
  };

  const handleSelectAllClick = (event: React.ChangeEvent<HTMLInputElement>) => {
    if (event.target.checked) {
      const newSelected = users.map((n) => n.id);
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
    return !!(email || nick || group || status);
  }, [email, nick, group, status]);

  const handleUserDialogOpen = (id: number) => {
    setUserDialogID(id);
    setUserDialogOpen(true);
  };

  return (
    <PageContainer>
      <NewUserDialog
        open={createNewOpen}
        onClose={() => setCreateNewOpen(false)}
        onCreated={(user) => {
          setUserDialogID(user.id);
          setUserDialogOpen(true);
        }}
      />
      <UserDialog
        open={userDialogOpen}
        onClose={() => setUserDialogOpen(false)}
        userID={userDialogID}
        onUpdated={() => fetchUsers()}
      />
      <Container maxWidth="xl">
        <PageHeader title={t("dashboard:nav.users")} />
        <Stack direction="row" spacing={1} sx={{ mb: 2 }}>
          <Button onClick={() => setCreateNewOpen(true)} variant={"contained"} startIcon={<Add />}>
            {t("group.create")}
          </Button>

          <UserFilterPopover
            {...bindPopover(filterPopupState)}
            email={email}
            setEmail={setEmail}
            nick={nick}
            setNick={setNick}
            group={group}
            setGroup={setGroup}
            status={status}
            setStatus={setStatus}
            clearFilters={clearFilters}
          />

          <SecondaryButton onClick={fetchUsers} disabled={loading} variant={"contained"} startIcon={<ArrowSync />}>
            {t("node.refresh")}
          </SecondaryButton>

          <Badge color="primary" variant="dot" invisible={!hasActiveFilters}>
            <SecondaryButton startIcon={<Filter />} variant="contained" {...bindTrigger(filterPopupState)}>
              {t("user.filter")}
            </SecondaryButton>
          </Badge>

          {selected.length > 0 && !isMobile && (
            <>
              <Divider orientation="vertical" flexItem />
              <Button startIcon={<Delete />} variant="contained" color="error" onClick={handleDelete}>
                {t("user.deleteXUsers", { num: selected.length })}
              </Button>
            </>
          )}
        </Stack>
        {isMobile && selected.length > 0 && (
          <Stack direction="row" spacing={1} sx={{ mb: 2 }}>
            <Button startIcon={<Delete />} variant="contained" color="error" onClick={handleDelete}>
              {t("user.deleteXUsers", { num: selected.length })}
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
                    disableRipple
                    color="primary"
                    indeterminate={selected.length > 0 && selected.length < users.length}
                    checked={users.length > 0 && selected.length === users.length}
                    onChange={handleSelectAllClick}
                  />
                </TableCell>
                <NoWrapTableCell width={60} sortDirection={orderById ? direction : false}>
                  <TableSortLabel active={orderById} direction={direction} onClick={onSortClick("id")}>
                    {t("group.#")}
                  </TableSortLabel>
                </NoWrapTableCell>
                <NoWrapTableCell width={250}>
                  <TableSortLabel active={orderBy === "nick"} direction={direction} onClick={onSortClick("nick")}>
                    {t("user.nick")}
                  </TableSortLabel>
                </NoWrapTableCell>
                <NoWrapTableCell width={250}>
                  <TableSortLabel active={orderBy === "email"} direction={direction} onClick={onSortClick("email")}>
                    {t("user.email")}
                  </TableSortLabel>
                </NoWrapTableCell>
                <NoWrapTableCell width={150}>{t("user.group")}</NoWrapTableCell>
                <NoWrapTableCell width={100}>
                  <TableSortLabel active={orderBy === "storage"} direction={direction} onClick={onSortClick("storage")}>
                    {t("user.usedStorage")}
                  </TableSortLabel>
                </NoWrapTableCell>
                <NoWrapTableCell width={100} align="right"></NoWrapTableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {!loading &&
                users.map((user) => (
                  <UserRow
                    deleting={deleteLoading}
                    key={user.id}
                    user={user}
                    onDelete={fetchUsers}
                    selected={selected.includes(user.id)}
                    onSelect={handleSelect}
                    onDetails={handleUserDialogOpen}
                  />
                ))}
              {loading &&
                users.length > 0 &&
                users.slice(0, 10).map((user) => <UserRow key={`loading-${user.id}`} loading={true} />)}
              {loading &&
                users.length === 0 &&
                Array.from(Array(5)).map((_, index) => <UserRow key={`loading-${index}`} loading={true} />)}
            </TableBody>
          </Table>
        </TableContainer>
        {count > 0 && (
          <Box sx={{ mt: 1 }}>
            <TablePagination
              page={pageInt}
              totalItems={count}
              rowsPerPage={pageSizeInt}
              rowsPerPageOptions={[10, 25, 50, 100, 200]}
              onRowsPerPageChange={(value) => setPageSize(value.toString())}
              onChange={(_, value) => setPage(value.toString())}
            />
          </Box>
        )}
      </Container>
    </PageContainer>
  );
};

export default UserSetting;
