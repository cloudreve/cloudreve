import { useTheme } from "@emotion/react";
import {
  Box,
  Button,
  Container,
  Stack,
  Table,
  TableBody,
  TableContainer,
  TableHead,
  TableRow,
  TableSortLabel,
} from "@mui/material";
import { useQueryState } from "nuqs";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { getGroupList } from "../../../api/api";
import { GroupEnt } from "../../../api/dashboard";
import { useAppDispatch } from "../../../redux/hooks";
import { NoWrapTableCell, SecondaryButton, StyledTableContainerPaper } from "../../Common/StyledComponents";
import Add from "../../Icons/Add";
import ArrowSync from "../../Icons/ArrowSync";
import PageContainer from "../../Pages/PageContainer";
import PageHeader from "../../Pages/PageHeader";
import TablePagination from "../Common/TablePagination";
import { OrderByQuery, OrderDirectionQuery, PageQuery, PageSizeQuery } from "../StoragePolicy/StoragePolicySetting";
import GroupRow from "./GroupRow";
import NewGroupDialog from "./NewGroupDIalog";

const GroupSetting = () => {
  const { t } = useTranslation("dashboard");
  const theme = useTheme();
  const dispatch = useAppDispatch();
  const [loading, setLoading] = useState(true);
  const [groups, setGroups] = useState<GroupEnt[]>([]);
  const [page, setPage] = useQueryState(PageQuery, { defaultValue: "1" });
  const [pageSize, setPageSize] = useQueryState(PageSizeQuery, {
    defaultValue: "10",
  });
  const [orderBy, setOrderBy] = useQueryState(OrderByQuery, {
    defaultValue: "",
  });
  const [orderDirection, setOrderDirection] = useQueryState(OrderDirectionQuery, { defaultValue: "desc" });
  const [count, setCount] = useState(0);
  const [createNewOpen, setCreateNewOpen] = useState(false);

  const pageInt = parseInt(page) ?? 1;
  const pageSizeInt = parseInt(pageSize) ?? 11;

  useEffect(() => {
    fetchGroups();
  }, [page, pageSize, orderBy, orderDirection]);

  const fetchGroups = () => {
    setLoading(true);
    dispatch(
      getGroupList({
        page: pageInt,
        page_size: pageSizeInt,
        order_by: orderBy ?? "",
        order_direction: orderDirection ?? "desc",
      }),
    )
      .then((res) => {
        setGroups(res.groups);
        setPageSize(res.pagination.page_size.toString());
        setCount(res.pagination.total_items ?? 0);
        setLoading(false);
      })
      .finally(() => {
        setLoading(false);
      });
  };

  const orderById = orderBy === "id" || orderBy === "";
  const direction = orderDirection as "asc" | "desc";
  const onSortClick = (field: string) => () => {
    const alreadySorted = orderBy === field || (field === "id" && orderById);
    setOrderBy(field);
    setOrderDirection(alreadySorted ? (direction === "asc" ? "desc" : "asc") : "asc");
  };

  return (
    <PageContainer>
      <NewGroupDialog open={createNewOpen} onClose={() => setCreateNewOpen(false)} />
      <Container maxWidth="xl">
        <PageHeader title={t("dashboard:nav.groups")} />
        <Stack direction="row" spacing={1} sx={{ mb: 2 }}>
          <Button onClick={() => setCreateNewOpen(true)} variant={"contained"} startIcon={<Add />}>
            {t("group.create")}
          </Button>
          <SecondaryButton onClick={fetchGroups} disabled={loading} variant={"contained"} startIcon={<ArrowSync />}>
            {t("node.refresh")}
          </SecondaryButton>
        </Stack>
        <TableContainer component={StyledTableContainerPaper} sx={{ mt: 2 }}>
          <Table size="small" stickyHeader sx={{ width: "100%", tableLayout: "fixed" }}>
            <TableHead>
              <TableRow>
                <NoWrapTableCell width={50} sortDirection={orderById ? direction : false}>
                  <TableSortLabel active={orderById} direction={direction} onClick={onSortClick("id")}>
                    {t("group.#")}
                  </TableSortLabel>
                </NoWrapTableCell>
                <NoWrapTableCell width={150}>
                  <TableSortLabel active={orderBy === "name"} direction={direction} onClick={onSortClick("name")}>
                    {t("group.name")}
                  </TableSortLabel>
                </NoWrapTableCell>
                <NoWrapTableCell width={250}>{t("group.type")}</NoWrapTableCell>
                <NoWrapTableCell width={150}>{t("group.count")}</NoWrapTableCell>
                <NoWrapTableCell width={150}>
                  <TableSortLabel
                    active={orderBy === "max_storage"}
                    direction={direction}
                    onClick={onSortClick("max_storage")}
                  >
                    {t("group.size")}
                  </TableSortLabel>
                </NoWrapTableCell>
                <NoWrapTableCell width={100} align="right"></NoWrapTableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {!loading && groups.map((group) => <GroupRow key={group.id} group={group} onDelete={fetchGroups} />)}
              {loading &&
                groups.length > 0 &&
                groups.map((group) => <GroupRow key={`loading-${group.id}`} loading={true} />)}
              {loading &&
                groups.length === 0 &&
                Array.from(Array(5)).map((_, index) => <GroupRow key={`loading-${index}`} loading={true} />)}
            </TableBody>
          </Table>
        </TableContainer>
        {count > 0 && (
          <Box sx={{ mt: 1 }}>
            <TablePagination
              page={pageInt}
              totalItems={count}
              rowsPerPage={pageSizeInt}
              rowsPerPageOptions={[10, 25, 50, 100, 200, 500, 1000]}
              onRowsPerPageChange={(value) => setPageSize(value.toString())}
              onChange={(_, value) => setPage(value.toString())}
            />
          </Box>
        )}
      </Container>
    </PageContainer>
  );
};

export default GroupSetting;
