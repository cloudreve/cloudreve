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
import { getEntityList } from "../../../api/api";
import { AdminListService, Entity } from "../../../api/dashboard";
import { useAppDispatch } from "../../../redux/hooks";
import { NoWrapTableCell, SecondaryButton, StyledTableContainerPaper } from "../../Common/StyledComponents";
import ArrowSync from "../../Icons/ArrowSync";
import Filter from "../../Icons/Filter";
import PageContainer from "../../Pages/PageContainer";
import PageHeader from "../../Pages/PageHeader";
import TablePagination from "../Common/TablePagination";
import { OrderByQuery, OrderDirectionQuery, PageQuery, PageSizeQuery } from "../StoragePolicy/StoragePolicySetting";
import UserDialog from "../User/UserDialog/UserDialog";
import EntityDeleteDialog from "./EntityDeleteDialog";
import EntityDialog from "./EntityDialog/EntityDialog";
import EntityFilterPopover from "./EntityFilterPopover";
import EntityRow from "./EntityRow";
export const StoragePolicyQuery = "storage_policy";
export const UserQuery = "user";
export const TypeQuery = "type";

const EntitySetting = () => {
  const { t } = useTranslation("dashboard");
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const dispatch = useAppDispatch();
  const [loading, setLoading] = useState(true);
  const [entities, setEntities] = useState<Entity[]>([]);
  const [page, setPage] = useQueryState(PageQuery, { defaultValue: "1" });
  const [pageSize, setPageSize] = useQueryState(PageSizeQuery, {
    defaultValue: "10",
  });
  const [orderBy, setOrderBy] = useQueryState(OrderByQuery, {
    defaultValue: "",
  });
  const [orderDirection, setOrderDirection] = useQueryState(OrderDirectionQuery, { defaultValue: "desc" });
  const [storagePolicy, setStoragePolicy] = useQueryState(StoragePolicyQuery, { defaultValue: "" });
  const [user, setUser] = useQueryState(UserQuery, { defaultValue: "" });
  const [type, setType] = useQueryState(TypeQuery, { defaultValue: "" });
  const [count, setCount] = useState(0);
  const [selected, setSelected] = useState<readonly number[]>([]);
  const filterPopupState = usePopupState({
    variant: "popover",
    popupId: "entityFilterPopover",
  });

  const [userDialogOpen, setUserDialogOpen] = useState(false);
  const [userDialogID, setUserDialogID] = useState<number | undefined>(undefined);
  const [entityDialogOpen, setEntityDialogOpen] = useState(false);
  const [entityDialogID, setEntityDialogID] = useState<number | undefined>(undefined);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deleteDialogID, setDeleteDialogID] = useState<number[] | undefined>(undefined);

  const pageInt = parseInt(page) ?? 1;
  const pageSizeInt = parseInt(pageSize) ?? 10;

  const clearFilters = useCallback(() => {
    setStoragePolicy("");
    setUser("");
    setType("");
  }, [setStoragePolicy, setUser, setType]);

  useEffect(() => {
    fetchEntities();
  }, [page, pageSize, orderBy, orderDirection, storagePolicy, user, type]);

  const fetchEntities = () => {
    setLoading(true);
    setSelected([]);

    const params: AdminListService = {
      page: pageInt,
      page_size: pageSizeInt,
      order_by: orderBy ?? "",
      order_direction: orderDirection ?? "desc",
      conditions: {},
    };

    if (storagePolicy) {
      params.conditions!.entity_policy = storagePolicy;
    }

    if (user) {
      params.conditions!.entity_user = user;
    }

    if (type) {
      params.conditions!.entity_type = type;
    }

    dispatch(getEntityList(params))
      .then((res) => {
        setEntities(res.entities);
        setPageSize(res.pagination.page_size.toString());
        setCount(res.pagination.total_items ?? 0);
        setLoading(false);
      })
      .finally(() => {
        setLoading(false);
      });
  };

  const handleDelete = () => {
    setDeleteDialogOpen(true);
    setDeleteDialogID(Array.from(selected));
  };

  const handleSelectAllClick = (event: React.ChangeEvent<HTMLInputElement>) => {
    if (event.target.checked) {
      const newSelected = entities.map((n) => n.id);
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
    return !!(storagePolicy || user || type);
  }, [storagePolicy, user, type]);

  const handleUserDialogOpen = (id: number) => {
    setUserDialogID(id);
    setUserDialogOpen(true);
  };

  const handleEntityDialogOpen = (id: number) => {
    setEntityDialogID(id);
    setEntityDialogOpen(true);
  };

  const handleSingleDelete = (id: number) => {
    setDeleteDialogOpen(true);
    setDeleteDialogID([id]);
  };

  return (
    <PageContainer>
      <EntityDialog open={entityDialogOpen} onClose={() => setEntityDialogOpen(false)} entityID={entityDialogID} />
      <UserDialog open={userDialogOpen} onClose={() => setUserDialogOpen(false)} userID={userDialogID} />
      <EntityDeleteDialog
        open={deleteDialogOpen}
        onClose={() => setDeleteDialogOpen(false)}
        entityID={deleteDialogID}
        onDelete={fetchEntities}
      />
      <Container maxWidth="xl">
        <PageHeader title={t("dashboard:nav.entities")} />
        <Stack direction="row" spacing={1} sx={{ mb: 2 }}>
          <EntityFilterPopover
            {...bindPopover(filterPopupState)}
            storagePolicy={storagePolicy}
            setStoragePolicy={setStoragePolicy}
            owner={user}
            setOwner={setUser}
            type={type !== "" ? parseInt(type) : undefined}
            setType={(type) => setType(type !== undefined ? type.toString() : "")}
            clearFilters={clearFilters}
          />

          <SecondaryButton onClick={fetchEntities} disabled={loading} variant={"contained"} startIcon={<ArrowSync />}>
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
                {t("entity.deleteXEntities", { num: selected.length })}
              </Button>
            </>
          )}
        </Stack>
        {isMobile && selected.length > 0 && (
          <Stack direction="row" spacing={1} sx={{ mb: 2 }}>
            <Button startIcon={<Delete />} variant="contained" color="error" onClick={handleDelete}>
              {t("entity.deleteXEntities", { num: selected.length })}
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
                    indeterminate={selected.length > 0 && selected.length < entities.length}
                    checked={entities.length > 0 && selected.length === entities.length}
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
                <NoWrapTableCell width={100}>{t("file.blobType")}</NoWrapTableCell>
                <NoWrapTableCell width={250}>{t("file.source")}</NoWrapTableCell>
                <NoWrapTableCell width={100}>
                  <TableSortLabel
                    active={orderBy === "size"}
                    direction={orderBy === "size" ? direction : "asc"}
                    onClick={onSortClick("size")}
                  >
                    {t("file.size")}
                  </TableSortLabel>
                </NoWrapTableCell>
                <NoWrapTableCell width={150}>{t("file.storagePolicy")}</NoWrapTableCell>
                <NoWrapTableCell width={100}>
                  <TableSortLabel
                    active={orderBy === "reference_count"}
                    direction={orderBy === "reference_count" ? direction : "asc"}
                    onClick={onSortClick("reference_count")}
                  >
                    {t("entity.refenenceCount")}
                  </TableSortLabel>
                </NoWrapTableCell>
                <NoWrapTableCell width={180}>
                  <TableSortLabel
                    active={orderBy === "created_at"}
                    direction={orderBy === "created_at" ? direction : "asc"}
                    onClick={onSortClick("created_at")}
                  >
                    {t("file.createdAt")}
                  </TableSortLabel>
                </NoWrapTableCell>
                <NoWrapTableCell width={180}>{t("file.creator")}</NoWrapTableCell>
                <NoWrapTableCell width={100}></NoWrapTableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {!loading &&
                entities.map((entity) => (
                  <EntityRow
                    key={entity.id}
                    entity={entity}
                    onDelete={handleSingleDelete}
                    selected={selected.indexOf(entity.id) !== -1}
                    onSelect={handleSelect}
                    openUserDialog={handleUserDialogOpen}
                    openEntityDialog={handleEntityDialogOpen}
                  />
                ))}
              {loading &&
                entities.length > 0 &&
                entities.slice(0, 10).map((entity) => <EntityRow key={`loading-${entity.id}`} loading={true} />)}
              {loading &&
                entities.length === 0 &&
                Array.from(Array(10)).map((_, index) => <EntityRow key={`loading-${index}`} loading={true} />)}
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

export default EntitySetting;
