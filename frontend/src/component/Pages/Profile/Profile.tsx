import * as React from "react";
import { useCallback, useEffect, useState } from "react";
import { Box, Container, FormControl, Grid, ListItemText, SelectChangeEvent } from "@mui/material";
import { useTranslation } from "react-i18next";
import PageHeader from "../PageHeader.tsx";
import { getUserShares } from "../../../api/api.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import Nothing from "../../Common/Nothing.tsx";
import { setSelected } from "../../../redux/fileManagerSlice.ts";
import { Share } from "../../../api/explorer.ts";
import { DenseSelect } from "../../Common/StyledComponents.tsx";
import { SquareMenuItem } from "../../FileManager/ContextMenu/ContextMenu.tsx";
import ShareCard from "../Shares/ShareCard.tsx";
import { useParams } from "react-router-dom";
import { ShareLinksInProfileLevel, User } from "../../../api/user.ts";
import { loadUserInfo } from "../../../redux/thunks/session.ts";
import { UserProfile } from "../../Common/User/UserPopover.tsx";
import PageContainer from "../PageContainer.tsx";

const defaultPageSize = 50;

const Profile = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();

  const [nextPageToken, setNextPageToken] = useState<string | undefined>("");
  const [shares, setShares] = useState<Share[]>([]);
  const [loading, setLoading] = useState(false);
  const [orderDirection, setOrderDirection] = useState("desc");
  const [user, setUser] = useState<User | undefined>();

  const { id } = useParams<{ id: string }>();

  useEffect(() => {
    if (!id) {
      return;
    }
    dispatch(loadUserInfo(id)).then((u) => {
      if (u) {
        setUser(u);
      }
    });
  }, [id]);

  const loadNextPage = useCallback(
    (originShares: Share[], token?: string, direction?: string) => () => {
      setLoading(true);
      dispatch(
        getUserShares(
          {
            page_size: defaultPageSize,
            order_direction: direction ?? orderDirection,
            next_page_token: token,
          },
          id ?? "",
        ),
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
        <PageHeader title={user?.nickname ?? "-"} />
        <Box
          sx={{
            backgroundColor: (theme) =>
              theme.palette.mode == "light" ? "rgba(0, 0, 0, 0.06)" : "rgba(255, 255, 255, 0.09)",
            borderRadius: (theme) => `${theme.shape.borderRadius}px`,
            mb: 4,
            p: 2,
          }}
        >
          {user && <UserProfile user={user} open={true} displayOnly />}
        </Box>

        <PageHeader
          secondaryAction={
            user &&
            user.share_links_in_profile !== ShareLinksInProfileLevel.hide_share && (
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
            )
          }
          skipChangingDocumentTitle
          onRefresh={
            user && user.share_links_in_profile !== ShareLinksInProfileLevel.hide_share ? () => refresh() : undefined
          }
          loading={loading}
          title={t("application:share.somebodyShare", {
            name: user?.nickname ?? "-",
          })}
        />

        <Grid container spacing={1}>
          {user &&
            user.share_links_in_profile !== ShareLinksInProfileLevel.hide_share &&
            shares.map((share) => <ShareCard share={share} onShareDeleted={onShareDeleted} />)}
          {nextPageToken != undefined &&
            user &&
            user?.share_links_in_profile !== ShareLinksInProfileLevel.hide_share && (
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
          {user && user.share_links_in_profile === ShareLinksInProfileLevel.hide_share && (
            <>
              <Box sx={{ p: 1, width: "100%", textAlign: "center" }}>
                <Nothing size={0.8} top={63} primary={t("setting.userHideShare")} />
              </Box>
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

export default Profile;
