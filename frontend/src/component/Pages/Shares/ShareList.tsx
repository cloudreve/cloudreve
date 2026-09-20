import * as React from "react";
import { useCallback, useState } from "react";
import { Box, Button, Container, FormControl, Grid, ListItemText, SelectChangeEvent, Stack } from "@mui/material";
import { useSnackbar } from "notistack";
import { useTranslation } from "react-i18next";
import PageHeader from "../PageHeader.tsx";
import { getShares, sendDeleteShares } from "../../../api/api.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { confirmOperation } from "../../../redux/thunks/dialog.ts";
import Nothing from "../../Common/Nothing.tsx";
import { DefaultCloseAction } from "../../Common/Snackbar/snackbar.tsx";
import { setSelected } from "../../../redux/fileManagerSlice.ts";
import ShareCard from "./ShareCard.tsx";
import SaveShareLinkDialog from "./SaveShareLinkDialog.tsx";
import { Share } from "../../../api/explorer.ts";
import { DenseSelect } from "../../Common/StyledComponents.tsx";
import { SquareMenuItem } from "../../FileManager/ContextMenu/ContextMenu.tsx";
import PageContainer from "../PageContainer.tsx";

const defaultPageSize = 50;

const ShareList = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const { enqueueSnackbar } = useSnackbar();
  const [nextPageToken, setNextPageToken] = useState<string | undefined>("");
  const [shares, setShares] = useState<Share[]>([]);
  const [loading, setLoading] = useState(false);
  const [orderDirection, setOrderDirection] = useState("desc");
  const [selecting, setSelecting] = useState(false);
  const [selected, setSelectedIds] = useState<Set<string>>(new Set());
  const [saveLinkOpen, setSaveLinkOpen] = useState(false);

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

  const onToggleSelect = useCallback(
    (id: string) => {
      setSelectedIds((prev) => {
        const next = new Set(prev);
        if (next.has(id)) {
          next.delete(id);
        } else {
          next.add(id);
        }
        return next;
      });
    },
    [setSelectedIds],
  );

  const exitSelecting = useCallback(() => {
    setSelecting(false);
    setSelectedIds(new Set());
  }, []);

  const deleteSelected = useCallback(() => {
    dispatch(confirmOperation(t("fileManager.deleteShareWarning"))).then(() => {
      dispatch(sendDeleteShares([...selected])).then(() => {
        enqueueSnackbar({
          message: t("application:share.shareCanceled"),
          variant: "success",
          action: DefaultCloseAction,
        });
        setShares((shares) => shares.filter((share) => !selected.has(share.id)));
        exitSelecting();
      });
    });
  }, [dispatch, selected, t, enqueueSnackbar, exitSelecting]);

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
            <Stack direction="row" spacing={1} alignItems="center">
              {selecting && (
                <>
                  <Button
                    size="small"
                    variant="text"
                    onClick={() => setSelectedIds(new Set(shares.map((s) => s.id)))}
                  >
                    {t("application:fileManager.selectAll")}
                  </Button>
                  <Button
                    size="small"
                    variant="text"
                    color="error"
                    disabled={selected.size == 0}
                    onClick={deleteSelected}
                  >
                    {t("fileManager.delete")} ({selected.size})
                  </Button>
                </>
              )}
              <Button size="small" variant="text" onClick={() => setSaveLinkOpen(true)}>
                {t("application:share.saveShareLink")}
              </Button>
              <Button size="small" variant="text" onClick={selecting ? exitSelecting : () => setSelecting(true)}>
                {selecting ? t("common:cancel") : t("common:select")}
              </Button>
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
            </Stack>
          }
          onRefresh={() => refresh()}
          loading={loading}
          title={t("application:navbar.myShare")}
        />

        <Grid container spacing={1}>
          {shares.map((share) => (
            <ShareCard
              share={share}
              key={share.id}
              onShareDeleted={onShareDeleted}
              selecting={selecting}
              selected={selected.has(share.id)}
              onToggleSelect={onToggleSelect}
            />
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
        <SaveShareLinkDialog open={saveLinkOpen} onClose={() => setSaveLinkOpen(false)} />
      </Container>
    </PageContainer>
  );
};

export default ShareList;
