import { Add } from "@mui/icons-material";
import { Box, Container, Grid2 as Grid, IconButton, Stack, Typography } from "@mui/material";
import { useQueryState } from "nuqs";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { getNodeList } from "../../../api/api";
import { Node } from "../../../api/dashboard";
import { useAppDispatch } from "../../../redux/hooks";
import { SecondaryButton } from "../../Common/StyledComponents";
import ArrowSync from "../../Icons/ArrowSync";
import QuestionCircle from "../../Icons/QuestionCircle";
import PageContainer from "../../Pages/PageContainer";
import PageHeader from "../../Pages/PageHeader";
import { BorderedCardClickable } from "../Common/AdminCard";
import TablePagination from "../Common/TablePagination";
import { OrderByQuery, OrderDirectionQuery, PageQuery, PageSizeQuery } from "../StoragePolicy/StoragePolicySetting";
import { NewNodeDialog } from "./NewNode/NewNodeDialog";
import NodeCard from "./NodeCard";

const NodeSetting = () => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const [loading, setLoading] = useState(true);
  const [nodes, setNodes] = useState<Node[]>([]);
  const [page, setPage] = useQueryState(PageQuery, { defaultValue: "1" });
  const [pageSize, setPageSize] = useQueryState(PageSizeQuery, {
    defaultValue: "11",
  });
  const [orderBy, setOrderBy] = useQueryState(OrderByQuery, {
    defaultValue: "",
  });
  const [orderDirection, setOrderDirection] = useQueryState(OrderDirectionQuery, { defaultValue: "desc" });
  const [count, setCount] = useState(0);
  const [selectProviderOpen, setSelectProviderOpen] = useState(false);
  const [createNewOpen, setCreateNewOpen] = useState(false);

  const pageInt = parseInt(page) ?? 1;
  const pageSizeInt = parseInt(pageSize) ?? 11;

  useEffect(() => {
    fetchNodes();
  }, [page, pageSize, orderBy, orderDirection]);

  const fetchNodes = () => {
    setLoading(true);
    dispatch(
      getNodeList({
        page: pageInt,
        page_size: pageSizeInt,
        order_by: orderBy ?? "",
        order_direction: orderDirection ?? "desc",
        conditions: {},
      }),
    )
      .then((res) => {
        setNodes(res.nodes);
        setPage((res.pagination.page + 1).toString());
        setPageSize(res.pagination.page_size.toString());
        setCount(res.pagination.total_items ?? 0);
        setLoading(false);
      })
      .finally(() => {
        setLoading(false);
      });
  };

  return (
    <PageContainer>
      <NewNodeDialog open={createNewOpen} onClose={() => setCreateNewOpen(false)} />
      <Container maxWidth="xl">
        <PageHeader
          title={t("dashboard:nav.nodes")}
          secondaryAction={
            <IconButton onClick={() => window.open("https://docs.cloudreve.org/usage/slave-node", "_blank")}>
              <QuestionCircle />
            </IconButton>
          }
        />
        <Stack direction="row" spacing={1} sx={{ mb: 2 }}>
          <SecondaryButton onClick={fetchNodes} disabled={loading} variant={"contained"} startIcon={<ArrowSync />}>
            {t("node.refresh")}
          </SecondaryButton>
        </Stack>
        <Grid container spacing={2}>
          <Grid
            size={{
              xs: 12,
              md: 6,
              lg: 4,
            }}
          >
            <BorderedCardClickable
              onClick={() => setCreateNewOpen(true)}
              sx={{
                height: "100%",
                borderStyle: "dashed",
                display: "flex",
                alignItems: "center",
                gap: 1,
                justifyContent: "center",
                color: (t) => t.palette.text.secondary,
              }}
            >
              <Add />
              <Typography variant="h6">{t("node.addNewNode")}</Typography>
            </BorderedCardClickable>
          </Grid>
          {!loading && nodes.map((n) => <NodeCard key={n.id} node={n} onRefresh={fetchNodes} />)}
          {loading && nodes.length > 0 && nodes.map((n) => <NodeCard key={`loading-${n.id}`} loading={true} />)}
          {loading &&
            nodes.length === 0 &&
            Array.from(Array(5)).map((_, index) => <NodeCard key={`loading-placeholder-${index}`} loading={true} />)}
        </Grid>
        {count > 0 && (
          <Box sx={{ mt: 1 }}>
            <TablePagination
              page={pageInt}
              totalItems={count}
              rowsPerPage={pageSizeInt}
              rowsPerPageOptions={[11, 25, 50, 100, 200, 500, 1000]}
              onRowsPerPageChange={(value) => setPageSize(value.toString())}
              onChange={(_, value) => setPage(value.toString())}
            />
          </Box>
        )}
      </Container>
    </PageContainer>
  );
};

export default NodeSetting;
