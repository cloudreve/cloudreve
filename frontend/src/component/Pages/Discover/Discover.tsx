import { Box, Container, FormControl, Grid, ListItemText, Stack } from "@mui/material";
import { debounce } from "lodash";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getPublicShares } from "../../../api/api.ts";
import { Share } from "../../../api/explorer.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { OutlineIconTextField } from "../../Common/Form/OutlineIconTextField.tsx";
import Nothing from "../../Common/Nothing.tsx";
import { DenseSelect } from "../../Common/StyledComponents.tsx";
import { SquareMenuItem } from "../../FileManager/ContextMenu/ContextMenu.tsx";
import Search from "../../Icons/Search.tsx";
import PageContainer from "../PageContainer.tsx";
import PageHeader from "../PageHeader.tsx";
import ShareCard from "../Shares/ShareCard.tsx";

const defaultPageSize = 50;

// Discover renders the public share directory: shares whose owners opted
// into public listing. Anonymous-accessible.
const Discover = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const [nextPageToken, setNextPageToken] = useState<string | undefined>("");
  const [shares, setShares] = useState<Share[]>([]);
  const [loading, setLoading] = useState(false);
  const [orderDirection, setOrderDirection] = useState("desc");
  const [query, setQuery] = useState("");

  const loadNextPage = useCallback(
    (originShares: Share[], token?: string, direction?: string, q?: string) => () => {
      setLoading(true);
      dispatch(
        getPublicShares({
          page_size: defaultPageSize,
          order_direction: direction ?? orderDirection,
          next_page_token: token,
          query: q ?? query,
        }),
      )
        .then((res) => {
          setShares([...originShares, ...res.shares]);
          setNextPageToken(res.pagination?.next_token);
        })
        .catch(() => {
          setNextPageToken(undefined);
        })
        .finally(() => {
          setLoading(false);
        });
    },
    [dispatch, orderDirection, query],
  );

  const refresh = useCallback(
    (direction?: string, q?: string) => {
      loadNextPage([], "", direction, q)();
    },
    [loadNextPage],
  );

  useEffect(() => {
    refresh();
  }, []);

  const onQueryChange = useMemo(
    () =>
      debounce((value: string) => {
        setQuery(value);
        refresh(undefined, value);
      }, 400),
    [refresh],
  );

  useEffect(() => () => onQueryChange.cancel(), [onQueryChange]);

  const onShareDeleted = useCallback((id: string) => {
    setShares((shares) => shares.filter((share) => share.id !== id));
  }, []);

  return (
    <PageContainer>
      <Container maxWidth="lg">
        <PageHeader
          secondaryAction={
            <Stack direction="row" spacing={1} alignItems="center">
              <FormControl variant="outlined">
                <DenseSelect
                  variant="outlined"
                  value={orderDirection}
                  onChange={(e) => {
                    setOrderDirection(e.target.value as string);
                    refresh(e.target.value as string);
                  }}
                >
                  <SquareMenuItem value={"desc"}>
                    <ListItemText slotProps={{ primary: { variant: "body2" } }}>
                      {t("application:share.createdAtDesc")}
                    </ListItemText>
                  </SquareMenuItem>
                  <SquareMenuItem value={"asc"}>
                    <ListItemText slotProps={{ primary: { variant: "body2" } }}>
                      {t("application:share.createdAtAsc")}
                    </ListItemText>
                  </SquareMenuItem>
                </DenseSelect>
              </FormControl>
            </Stack>
          }
          onRefresh={() => refresh()}
          loading={loading}
          title={t("application:navbar.discover")}
        />

        <OutlineIconTextField
          variant="outlined"
          icon={<Search />}
          placeholder={t("application:discover.searchPlaceholder")}
          fullWidth
          size="small"
          sx={{ mb: 2 }}
          onChange={(e) => onQueryChange(e.target.value)}
        />

        <Grid container spacing={1}>
          {shares.map((share) => (
            <ShareCard share={share} key={share.id} onShareDeleted={onShareDeleted} />
          ))}
          {nextPageToken != undefined && (
            <>
              {[...Array(4)].map((_, i) => (
                <ShareCard
                  onShareDeleted={onShareDeleted}
                  onLoad={i == 0 ? loadNextPage(shares, nextPageToken) : undefined}
                  loading={true}
                  key={i == 0 ? nextPageToken : i}
                />
              ))}
            </>
          )}
        </Grid>

        {nextPageToken == undefined && shares.length == 0 && !loading && (
          <Box sx={{ p: 1, width: "100%", textAlign: "center" }}>
            <Nothing size={0.8} top={63} primary={t("setting.listEmpty")} />
          </Box>
        )}
      </Container>
    </PageContainer>
  );
};

export default Discover;
