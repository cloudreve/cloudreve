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
import { batchDeleteFiles, getFlattenFileList } from "../../../api/api";
import { File } from "../../../api/dashboard";
import { Metadata } from "../../../api/explorer";
import { useAppDispatch } from "../../../redux/hooks";
import { confirmOperation } from "../../../redux/thunks/dialog";
import { NoWrapTableCell, SecondaryButton, StyledTableContainerPaper } from "../../Common/StyledComponents";
import ArrowImport from "../../Icons/ArrowImport";
import ArrowSync from "../../Icons/ArrowSync";
import Filter from "../../Icons/Filter";
import PageContainer from "../../Pages/PageContainer";
import PageHeader from "../../Pages/PageHeader";
import TablePagination from "../Common/TablePagination";
import { OrderByQuery, OrderDirectionQuery, PageQuery, PageSizeQuery } from "../StoragePolicy/StoragePolicySetting";
import UserDialog from "../User/UserDialog/UserDialog";
import FileDialog from "./FileDialog/FileDialog";
import FileFilterPopover from "./FileFilterPopover";
import FileRow from "./FileRow";
import { ImportFileDialog } from "./ImportFileDialog";

export const StoragePolicyQuery = "storage_policy";
export const OwnerQuery = "owner";
export const NameQuery = "name";
export const HasDirectLinkQuery = "has_direct_link";
export const SharedQuery = "shared";
export const UploadingQuery = "uploading";

const FileSetting = () => {
  const { t } = useTranslation("dashboard");
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const dispatch = useAppDispatch();
  const [loading, setLoading] = useState(true);
  const [files, setFiles] = useState<File[]>([]);
  const [page, setPage] = useQueryState(PageQuery, { defaultValue: "1" });
  const [pageSize, setPageSize] = useQueryState(PageSizeQuery, {
    defaultValue: "10",
  });
  const [orderBy, setOrderBy] = useQueryState(OrderByQuery, {
    defaultValue: "",
  });
  const [orderDirection, setOrderDirection] = useQueryState(OrderDirectionQuery, { defaultValue: "desc" });
  const [storagePolicy, setStoragePolicy] = useQueryState(StoragePolicyQuery, { defaultValue: "" });
  const [owner, setOwner] = useQueryState(OwnerQuery, { defaultValue: "" });
  const [name, setName] = useQueryState(NameQuery, { defaultValue: "" });
  const [hasDirectLink, setHasDirectLink] = useQueryState(HasDirectLinkQuery, { defaultValue: "" });
  const [shared, setShared] = useQueryState(SharedQuery, { defaultValue: "" });
  const [uploading, setUploading] = useQueryState(UploadingQuery, { defaultValue: "" });
  const [count, setCount] = useState(0);
  const [selected, setSelected] = useState<readonly number[]>([]);
  const [createNewOpen, setCreateNewOpen] = useState(false);
  const [importFileDialogOpen, setImportFileDialogOpen] = useState(false);
  const filterPopupState = usePopupState({
    variant: "popover",
    popupId: "userFilterPopover",
  });

  const [userDialogOpen, setUserDialogOpen] = useState(false);
  const [userDialogID, setUserDialogID] = useState<number | undefined>(undefined);
  const [fileDialogOpen, setFileDialogOpen] = useState(false);
  const [fileDialogID, setFileDialogID] = useState<number | undefined>(undefined);
  const [deleteLoading, setDeleteLoading] = useState(false);

  const pageInt = parseInt(page) ?? 1;
  const pageSizeInt = parseInt(pageSize) ?? 11;

  const clearFilters = useCallback(() => {
    setStoragePolicy("");
    setOwner("");
    setName("");
    setHasDirectLink("");
    setShared("");
    setUploading("");
  }, [setStoragePolicy, setOwner, setName, setHasDirectLink, setShared, setUploading]);

  useEffect(() => {
    fetchFiles();
  }, [page, pageSize, orderBy, orderDirection, storagePolicy, owner, name, hasDirectLink, shared, uploading]);

  const fetchFiles = () => {
    setLoading(true);
    setSelected([]);
    dispatch(
      getFlattenFileList({
        page: pageInt,
        page_size: pageSizeInt,
        order_by: orderBy ?? "",
        order_direction: orderDirection ?? "desc",
        conditions: {
          file_policy: storagePolicy,
          file_user: owner,
          file_name: name,
          file_direct_link: hasDirectLink === "true" ? "true" : "",
          file_shared: shared === "true" ? "true" : "",
          file_metadata: uploading === "true" ? Metadata.upload_session_id : "",
        },
      }),
    )
      .then((res) => {
        setFiles(res.files);
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
    dispatch(confirmOperation(t("file.confirmBatchDelete", { num: selected.length })))
      .then(() => {
        dispatch(batchDeleteFiles({ ids: Array.from(selected) }))
          .then(() => {
            fetchFiles();
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
      const newSelected = files.map((n) => n.id);
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
    return !!(storagePolicy || owner || name || hasDirectLink || shared || uploading);
  }, [storagePolicy, owner, name, hasDirectLink, shared, uploading]);

  const handleFileDialogOpen = (id: number) => {
    setFileDialogID(id);
    setFileDialogOpen(true);
  };

  const handleUserDialogOpen = (id: number) => {
    setUserDialogID(id);
    setUserDialogOpen(true);
  };

  return (
    <PageContainer>
      {/* <NewUserDialog
        open={createNewOpen}
        onClose={() => setCreateNewOpen(false)}
        onCreated={(user) => {
          setUserDialogID(user.id);
          setUserDialogOpen(true);
        }}
      /> */}
      <FileDialog
        open={fileDialogOpen}
        onClose={() => setFileDialogOpen(false)}
        fileID={fileDialogID}
        onUpdated={(file) => {
          setFileDialogID(file.id);
          setFileDialogOpen(true);
        }}
      />
      <UserDialog open={userDialogOpen} onClose={() => setUserDialogOpen(false)} userID={userDialogID} />
      <ImportFileDialog open={importFileDialogOpen} onClose={() => setImportFileDialogOpen(false)} />
      <Container maxWidth="xl">
        <PageHeader title={t("dashboard:nav.files")} />
        <Stack direction="row" spacing={1} sx={{ mb: 2 }}>
          <Button onClick={() => setImportFileDialogOpen(true)} variant={"contained"} startIcon={<ArrowImport />}>
            {t("file.import")}
          </Button>

          <FileFilterPopover
            {...bindPopover(filterPopupState)}
            storagePolicy={storagePolicy}
            setStoragePolicy={setStoragePolicy}
            owner={owner}
            setOwner={setOwner}
            name={name}
            setName={setName}
            clearFilters={clearFilters}
            hasDirectLink={hasDirectLink === "true"}
            setHasDirectLink={(value: boolean) => setHasDirectLink(value ? "true" : "")}
            hasShareLink={shared === "true"}
            setHasShareLink={(value: boolean) => setShared(value ? "true" : "")}
            isUploading={uploading === "true"}
            setIsUploading={(value: boolean) => setUploading(value ? "true" : "")}
          />

          <SecondaryButton onClick={fetchFiles} disabled={loading} variant={"contained"} startIcon={<ArrowSync />}>
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
                {t("file.deleteXFiles", { num: selected.length })}
              </Button>
            </>
          )}
        </Stack>
        {isMobile && selected.length > 0 && (
          <Stack direction="row" spacing={1} sx={{ mb: 2 }}>
            <Button startIcon={<Delete />} variant="contained" color="error" onClick={handleDelete}>
              {t("file.deleteXFiles", { num: selected.length })}
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
                    indeterminate={selected.length > 0 && selected.length < files.length}
                    checked={files.length > 0 && selected.length === files.length}
                    onChange={handleSelectAllClick}
                  />
                </TableCell>
                <NoWrapTableCell width={60} sortDirection={orderById ? direction : false}>
                  <TableSortLabel active={orderById} direction={direction} onClick={onSortClick("id")}>
                    {t("group.#")}
                  </TableSortLabel>
                </NoWrapTableCell>
                <NoWrapTableCell width={300}>
                  <TableSortLabel active={orderBy === "name"} direction={direction} onClick={onSortClick("name")}>
                    {t("file.name")}
                  </TableSortLabel>
                </NoWrapTableCell>
                <NoWrapTableCell width={70}>
                  <TableSortLabel active={orderBy === "size"} direction={direction} onClick={onSortClick("size")}>
                    {t("file.size")}
                  </TableSortLabel>
                </NoWrapTableCell>
                <NoWrapTableCell width={70}>{t("file.sizeUsed")}</NoWrapTableCell>
                <NoWrapTableCell width={150}>{t("file.uploader")}</NoWrapTableCell>
                <NoWrapTableCell width={150}>
                  <TableSortLabel
                    active={orderBy === "created_at"}
                    direction={direction}
                    onClick={onSortClick("created_at")}
                  >
                    {t("file.createdAt")}
                  </TableSortLabel>
                </NoWrapTableCell>
                <NoWrapTableCell width={100} align="right"></NoWrapTableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {!loading &&
                files.map((file) => (
                  <FileRow
                    deleting={deleteLoading}
                    key={file.id}
                    file={file}
                    onDelete={fetchFiles}
                    selected={selected.includes(file.id)}
                    onSelect={handleSelect}
                    onDetails={handleFileDialogOpen}
                    openUserDialog={handleUserDialogOpen}
                  />
                ))}
              {loading &&
                files.length > 0 &&
                files.slice(0, 10).map((file) => <FileRow key={`loading-${file.id}`} loading={true} />)}
              {loading &&
                files.length === 0 &&
                Array.from(Array(10)).map((_, index) => <FileRow key={`loading-${index}`} loading={true} />)}
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

export default FileSetting;
