import { Button, Chip, Container, Grid2, Paper, Stack, Typography } from "@mui/material";
import dayjs from "dayjs";
import { useSnackbar } from "notistack";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useSearchParams } from "react-router-dom";
import { getCredit, getShopSkus, purchaseSku } from "../../../api/api.ts";
import { CreditInfo, ShopSku } from "../../../api/user.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { sizeToString } from "../../../util/index.ts";
import { formatDuration } from "../../../util/datetime.ts";
import FacebookCircularProgress from "../../Common/CircularProgress.tsx";
import RedeemCodeInput from "../../Common/Form/RedeemCodeInput.tsx";
import Nothing from "../../Common/Nothing.tsx";
import ResponsiveTabs from "../../Common/ResponsiveTabs.tsx";
import PageContainer from "../PageContainer.tsx";
import PageHeader, { PageTabQuery } from "../PageHeader.tsx";

export enum ShopPageTab {
  Membership = "membership",
  Storage = "storage",
  Traffic = "traffic",
  Redeem = "redeem",
}

const Shop = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const { enqueueSnackbar } = useSnackbar();
  const [searchParams] = useSearchParams();
  const [tab, setTab] = useState(searchParams.get(PageTabQuery) ?? ShopPageTab.Membership);
  const [skus, setSkus] = useState<ShopSku[] | undefined>(undefined);
  const [info, setInfo] = useState<CreditInfo | undefined>(undefined);
  const [buying, setBuying] = useState<string | undefined>(undefined);

  const loadInfo = () => {
    dispatch(getCredit()).then((res) => setInfo(res));
  };

  useEffect(() => {
    dispatch(getShopSkus()).then((res) => setSkus(res));
    loadInfo();
  }, []);

  const tabs = useMemo(
    () => [
      { label: t("application:shop.memberships"), value: ShopPageTab.Membership },
      { label: t("application:shop.storagePacks"), value: ShopPageTab.Storage },
      { label: t("application:shop.trafficPacks"), value: ShopPageTab.Traffic },
      { label: t("application:shop.redeem"), value: ShopPageTab.Redeem },
    ],
    [],
  );

  const filtered = useMemo(
    () =>
      (skus ?? []).filter((s) =>
        tab === ShopPageTab.Membership
          ? s.type === "group"
          : tab === ShopPageTab.Traffic
            ? s.type === "traffic"
            : s.type === "storage",
      ),
    [skus, tab],
  );

  const onPurchase = (s: ShopSku) => {
    setBuying(s.id);
    dispatch(purchaseSku(s.id))
      .then((res) => {
        setInfo(res);
        enqueueSnackbar(t("shop.purchased", { name: s.name }), { variant: "success" });
      })
      .finally(() => setBuying(undefined));
  };

  const skuSubtitle = (s: ShopSku) => {
    const parts = [
      s.type === "group" ? s.group : sizeToString(s.amount),
      s.duration > 0 ? formatDuration(dayjs.duration(s.duration, "seconds")) : t("shop.permanent"),
    ];
    return parts.filter(Boolean).join(" · ");
  };

  return (
    <PageContainer>
      <Container maxWidth="lg">
        <PageHeader
          title={t("application:navbar.shop")}
          secondaryAction={
            info ? <Chip variant="outlined" label={t("shop.balance", { credits: info.credits })} /> : undefined
          }
        />
        <ResponsiveTabs value={tab} onChange={(_e, v) => setTab(v)} tabs={tabs} />
        {tab === ShopPageTab.Redeem && (
          <Paper variant="outlined" sx={{ p: 2, maxWidth: 480 }}>
            <Typography variant="subtitle2" gutterBottom>
              {t("setting.redeemGiftCode")}
            </Typography>
            <RedeemCodeInput onRedeemed={loadInfo} />
          </Paper>
        )}
        {tab !== ShopPageTab.Redeem && skus === undefined && (
          <Grid2 container sx={{ pt: 10, justifyContent: "center" }}>
            <FacebookCircularProgress />
          </Grid2>
        )}
        {tab !== ShopPageTab.Redeem && skus !== undefined && filtered.length === 0 && (
          <Nothing primary={t("shop.noProducts")} />
        )}
        {tab !== ShopPageTab.Redeem && skus !== undefined && filtered.length > 0 && (
          <Grid2 container spacing={2} sx={{ pt: 2 }}>
            {filtered.map((s) => (
              <Grid2 key={s.id} size={{ xs: 12, sm: 6, md: 4 }}>
                <Paper variant="outlined" sx={{ p: 2, height: "100%" }}>
                  <Stack spacing={1.5} sx={{ height: "100%" }}>
                    <Stack direction="row" spacing={1} alignItems="center">
                      <Typography variant="subtitle1" sx={{ flexGrow: 1 }}>
                        {s.name}
                      </Typography>
                      {s.label && <Chip size="small" color="primary" label={s.label} />}
                    </Stack>
                    <Typography variant="body2" color="text.secondary">
                      {skuSubtitle(s)}
                    </Typography>
                    {s.des && (
                      <Stack spacing={0.5} sx={{ flexGrow: 1 }}>
                        {s.des
                          .split("\n")
                          .filter((l) => l.trim())
                          .map((l, i) => (
                            <Typography key={i} variant="body2" color="text.secondary">
                              {l}
                            </Typography>
                          ))}
                      </Stack>
                    )}
                    <Button
                      variant="contained"
                      disabled={s.points == null || buying === s.id || (info != null && info.credits < s.points)}
                      onClick={() => onPurchase(s)}
                    >
                      {s.points != null
                        ? t("shop.buyWithPoints", { points: s.points })
                        : t("shop.pointsUnavailable")}
                    </Button>
                  </Stack>
                </Paper>
              </Grid2>
            ))}
          </Grid2>
        )}
      </Container>
    </PageContainer>
  );
};

export default Shop;
