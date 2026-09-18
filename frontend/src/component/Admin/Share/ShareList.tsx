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
import { batchDeleteShares, getShareList } from "../../../api/api";
import { AdminListService, Share } from "../../../api/dashboard";
import { useAppDispatch } from "../../../redux/hooks";
import { confirmOperation } from "../../../redux/thunks/dialog";
import { NoWrapTableCell, SecondaryButton, StyledTableContainerPaper } from "../../Common/StyledComponents";
import ArrowSync from "../../Icons/ArrowSync";
import Filter from "../../Icons/Filter";
import PageContainer from "../../Pages/PageContainer";
import PageHeader from "../../Pages/PageHeader";
import TablePagination from "../Common/TablePagination";
import FileDialog from "../File/FileDialog/FileDialog";
import { OrderByQuery, OrderDirectionQuery, PageQuery, PageSizeQuery } from "../StoragePolicy/StoragePolicySetting";
import UserDialog from "../User/UserDialog/UserDialog";
import ShareDialog from "./ShareDialog/ShareDialog";
import ShareFilterPopover from "./ShareFilterPopover";
import ShareRow from "./ShareRow";

export const UserQuery = "user";
export const FileQuery = "file";

const ShareList = () => {
  const { t } = useTranslation("dashboard");
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const dispatch = useAppDispatch();
  const [loading, setLoading] = useState(true);
  const [shares, setShares] = useState<Share[]>([]);
  const [page, setPage] = useQueryState(PageQuery, { defaultValue: "1" });
  const [pageSize, setPageSize] = useQueryState(PageSizeQuery, {
    defaultValue: "10",
  });
  const [orderBy, setOrderBy] = useQueryState(OrderByQuery, {
    defaultValue: "",
  });
  const [orderDirection, setOrderDirection] = useQueryState(OrderDirectionQuery, { defaultValue: "desc" });
  const [user, setUser] = useQueryState(UserQuery, { defaultValue: "" });
  const [file, setFile] = useQueryState(FileQuery, { defaultValue: "" });

  const [count, setCount] = useState(0);
  const [selected, setSelected] = useState<readonly number[]>([]);
  const filterPopupState = usePopupState({
    variant: "popover",
    popupId: "shareFilterPopover",
  });

  const [userDialogOpen, setUserDialogOpen] = useState(false);
  const [userDialogID, setUserDialogID] = useState<number | undefined>(undefined);
  const [openFile, setOpenFile] = useState<number | undefined>(undefined);
  const [openFileDialogOpen, setOpenFileDialogOpen] = useState(false);
  const [openShare, setOpenShare] = useState<number | undefined>(undefined);
  const [openShareDialogOpen, setOpenShareDialogOpen] = useState(false);
  const [deleteLoading, setDeleteLoading] = useState(false);

  const pageInt = parseInt(page) ?? 1;
  const pageSizeInt = parseInt(pageSize) ?? 10;

  const clearFilters = useCallback(() => {
    setUser("");
    setFile("");
  }, [setUser, setFile]);

  useEffect(() => {
    fetchShares();
  }, [page, pageSize, orderBy, orderDirection, user, file]);

  const fetchShares = () => {
    setLoading(true);
    setSelected([]);

    const params: AdminListService = {
      page: pageInt,
      page_size: pageSizeInt,
      order_by: orderBy ?? "",
      order_direction: orderDirection ?? "desc",
      conditions: {
        share_file_id: file,
        share_user_id: user,
      },
    };

    dispatch(getShareList(params))
      .then((res) => {
        setShares(res.shares);
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
    dispatch(confirmOperation(t("share.confirmBatchDelete", { num: selected.length })))
      .then(() => {
        dispatch(batchDeleteShares({ ids: Array.from(selected) }))
          .then(() => {
            fetchShares();
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
      const newSelected = shares.map((n) => n.id);
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
    return !!(user || file);
  }, [user, file]);

  const handleUserDialogOpen = (id: number) => {
    setUserDialogID(id);
    setUserDialogOpen(true);
  };

  const handleOpenFile = (fileID: number) => {
    setOpenFile(fileID);
    setOpenFileDialogOpen(true);
  };

  const handleOpenShare = (shareID: number) => {
    setOpenShare(shareID);
    setOpenShareDialogOpen(true);
  };

  return (
    <PageContainer>
      <UserDialog open={userDialogOpen} onClose={() => setUserDialogOpen(false)} userID={userDialogID} />
      <FileDialog open={openFileDialogOpen} onClose={() => setOpenFileDialogOpen(false)} fileID={openFile} />
      <ShareDialog open={openShareDialogOpen} onClose={() => setOpenShareDialogOpen(false)} shareID={openShare} />
      <Container maxWidth="xl">
        <PageHeader title={t("dashboard:nav.shares")} />
        <Stack direction="row" spacing={1} sx={{ mb: 2 }}>
          <ShareFilterPopover
            {...bindPopover(filterPopupState)}
            user={user}
            setUser={setUser}
            file={file}
            setFile={setFile}
            clearFilters={clearFilters}
          />

          <SecondaryButton onClick={fetchShares} disabled={loading} variant={"contained"} startIcon={<ArrowSync />}>
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
                {t("share.deleteXShares", { num: selected.length })}
              </Button>
            </>
          )}
        </Stack>
        {isMobile && selected.length > 0 && (
          <Stack direction="row" spacing={1} sx={{ mb: 2 }}>
            <Button startIcon={<Delete />} variant="contained" color="error" onClick={handleDelete}>
              {t("share.deleteXShares", { num: selected.length })}
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
                    indeterminate={selected.length > 0 && selected.length < shares.length}
                    checked={shares.length > 0 && selected.length === shares.length}
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
                <NoWrapTableCell width={400}>{t("share.srcFileName")}</NoWrapTableCell>
                <NoWrapTableCell width={100}>
                  <TableSortLabel
                    active={orderBy === "views"}
                    direction={orderBy === "views" ? direction : "asc"}
                    onClick={onSortClick("views")}
                  >
                    {t("share.views")}
                  </TableSortLabel>
                </NoWrapTableCell>
                <NoWrapTableCell width={100}>
                  <TableSortLabel
                    active={orderBy === "downloads"}
                    direction={orderBy === "downloads" ? direction : "asc"}
                    onClick={onSortClick("downloads")}
                  >
                    {t("share.downloads")}
                  </TableSortLabel>
                </NoWrapTableCell>
                <NoWrapTableCell width={100}>
                  <TableSortLabel
                    active={orderBy === "price"}
                    direction={orderBy === "price" ? direction : "asc"}
                    onClick={onSortClick("price")}
                  >
                    {t("share.price")}
                  </TableSortLabel>
                </NoWrapTableCell>
                <NoWrapTableCell width={150}>{t("share.autoExpire")}</NoWrapTableCell>
                <NoWrapTableCell width={150}>{t("share.owner")}</NoWrapTableCell>
                <NoWrapTableCell width={180}>
                  <TableSortLabel
                    active={orderBy === "created_at"}
                    direction={orderBy === "created_at" ? direction : "asc"}
                    onClick={onSortClick("created_at")}
                  >
                    {t("share.createdAt")}
                  </TableSortLabel>
                </NoWrapTableCell>
                <NoWrapTableCell width={80}></NoWrapTableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {!loading &&
                shares.map((share) => (
                  <ShareRow
                    deleting={deleteLoading}
                    key={share.id}
                    share={share}
                    onDelete={fetchShares}
                    selected={selected.indexOf(share.id) !== -1}
                    onSelect={handleSelect}
                    openUserDialog={handleUserDialogOpen}
                    openFileDialog={handleOpenFile}
                    onDetails={handleOpenShare}
                  />
                ))}
              {loading &&
                shares.length > 0 &&
                shares.slice(0, 10).map((share) => <ShareRow key={`loading-${share.id}`} loading={true} />)}
              {loading &&
                shares.length === 0 &&
                Array.from(Array(10)).map((_, index) => <ShareRow key={`loading-${index}`} loading={true} />)}
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

export default ShareList;
