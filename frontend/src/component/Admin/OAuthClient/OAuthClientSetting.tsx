import { Box, Container, Stack, Typography } from "@mui/material";
import Grid from "@mui/material/Grid2";
import { useQueryState } from "nuqs";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { getOAuthClientList } from "../../../api/api";
import { GetOAuthClientResponse } from "../../../api/dashboard";
import { useAppDispatch } from "../../../redux/hooks";
import { SecondaryButton } from "../../Common/StyledComponents";
import Add from "../../Icons/Add";
import ArrowSync from "../../Icons/ArrowSync";
import PageContainer from "../../Pages/PageContainer";
import PageHeader from "../../Pages/PageHeader";
import { BorderedCardClickable } from "../Common/AdminCard";
import TablePagination from "../Common/TablePagination";
import { OrderByQuery, OrderDirectionQuery, PageQuery, PageSizeQuery } from "../StoragePolicy/StoragePolicySetting";
import OAuthClientCard from "./OAuthClientCard";

const OAuthClientSetting = () => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [clients, setClients] = useState<GetOAuthClientResponse[]>([]);
  const [page, setPage] = useQueryState(PageQuery, { defaultValue: "1" });
  const [pageSize, setPageSize] = useQueryState(PageSizeQuery, {
    defaultValue: "11",
  });
  const [orderBy, setOrderBy] = useQueryState(OrderByQuery, {
    defaultValue: "",
  });
  const [orderDirection, setOrderDirection] = useQueryState(OrderDirectionQuery, { defaultValue: "desc" });
  const [count, setCount] = useState(0);

  const pageInt = parseInt(page) ?? 1;
  const pageSizeInt = parseInt(pageSize) ?? 11;

  useEffect(() => {
    fetchClients();
  }, [page, pageSize, orderBy, orderDirection]);

  const fetchClients = useCallback(() => {
    setLoading(true);
    dispatch(
      getOAuthClientList({
        page: pageInt,
        page_size: pageSizeInt,
        order_by: orderBy ?? "",
        order_direction: orderDirection ?? "desc",
        conditions: {},
      }),
    )
      .then((res) => {
        setClients(res.clients);
        setPageSize(res.pagination.page_size.toString());
        setCount(res.pagination.total_items ?? 0);
        setLoading(false);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [dispatch, pageInt, pageSizeInt, orderBy, orderDirection, setPageSize]);

  const handleAddNew = useCallback(() => {
    navigate("/admin/oauth/new");
  }, [navigate]);

  return (
    <PageContainer>
      <Container maxWidth="xl">
        <PageHeader title={t("oauth.oauthClients")} />
        <Stack direction="row" spacing={1} sx={{ mb: 2 }}>
          <SecondaryButton onClick={fetchClients} disabled={loading} variant={"contained"} startIcon={<ArrowSync />}>
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
              onClick={handleAddNew}
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
              <Typography variant="h6">{t("oauth.newOAuthClient")}</Typography>
            </BorderedCardClickable>
          </Grid>
          {!loading && clients.map((c) => <OAuthClientCard key={c.id} client={c} onRefresh={fetchClients} />)}
          {loading &&
            clients.length > 0 &&
            clients.map((c) => <OAuthClientCard key={`loading-${c.id}`} loading={true} />)}
          {loading &&
            clients.length === 0 &&
            Array.from(Array(5)).map((_, index) => (
              <OAuthClientCard key={`loading-placeholder-${index}`} loading={true} />
            ))}
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

export default OAuthClientSetting;
