import * as React from "react";
import { useCallback, useState } from "react";
import { Box, Container, FormControl, Grid, ListItemText, SelectChangeEvent } from "@mui/material";
import { useTranslation } from "react-i18next";
import PageHeader from "../PageHeader.tsx";
import { getShares } from "../../../api/api.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import Nothing from "../../Common/Nothing.tsx";
import { setSelected } from "../../../redux/fileManagerSlice.ts";
import ShareCard from "./ShareCard.tsx";
import { Share } from "../../../api/explorer.ts";
import { DenseSelect } from "../../Common/StyledComponents.tsx";
import { SquareMenuItem } from "../../FileManager/ContextMenu/ContextMenu.tsx";
import PageContainer from "../PageContainer.tsx";

const defaultPageSize = 50;

const ShareList = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const [nextPageToken, setNextPageToken] = useState<string | undefined>("");
  const [shares, setShares] = useState<Share[]>([]);
  const [loading, setLoading] = useState(false);
  const [orderDirection, setOrderDirection] = useState("desc");

  const loadNextPage = useCallback(
    (originShares: Share[], token?: string, direction?: string) => () => {
      setLoading(true);
      dispatch(
        getShares({
          page_size: defaultPageSize,
          order_direction: direction ?? orderDirection,
          next_page_token: token,
        }),
      )
        .then((res) => {
          setShares([...originShares, ...res.shares]);
          if (res.pagination?.next_token) {
            setNextPageToken(res.pagination.next_token);
          } else {
            setNextPageToken(undefined);
          }
        })
        .catch(() => {
          setNextPageToken(undefined);
        })
        .finally(() => {
          setLoading(false);
        });
    },
    [dispatch, orderDirection, setSelected],
  );

  const refresh = (direction?: string) => {
    loadNextPage([], "", direction)();
  };

  const onShareDeleted = useCallback(
    (id: string) => {
      setShares((shares) => shares.filter((share) => share.id !== id));
    },
    [setShares],
  );

  const onSelectChange = useCallback(
    (e: SelectChangeEvent<unknown>) => {
      setOrderDirection(e.target.value as string);
      refresh(e.target.value as string);
    },
    [refresh, setOrderDirection],
  );

  return (
    <PageContainer>
      <Container maxWidth="lg">
        <PageHeader
          secondaryAction={
            <FormControl variant="outlined">
              <DenseSelect variant="outlined" value={orderDirection} onChange={onSelectChange}>
                <SquareMenuItem value={"desc"}>
                  <ListItemText
                    slotProps={{
                      primary: { variant: "body2" },
                    }}
                  >
                    {t("application:share.createdAtDesc")}
                  </ListItemText>
                </SquareMenuItem>
                <SquareMenuItem value={"asc"}>
                  <ListItemText
                    slotProps={{
                      primary: { variant: "body2" },
                    }}
                  >
                    {t("application:share.createdAtAsc")}
                  </ListItemText>
                </SquareMenuItem>
              </DenseSelect>
            </FormControl>
          }
          onRefresh={() => refresh()}
          loading={loading}
          title={t("application:navbar.myShare")}
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

        {nextPageToken == undefined && shares.length == 0 && (
          <Box sx={{ p: 1, width: "100%", textAlign: "center" }}>
            <Nothing size={0.8} top={63} primary={t("setting.listEmpty")} />
          </Box>
        )}
      </Container>
    </PageContainer>
  );
};

export default ShareList;
