import { Alert, Box, Button, Typography } from "@mui/material";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link as RouterLink } from "react-router-dom";
import { getCredit, purchaseShare } from "../../api/api.ts";
import { CreditInfo } from "../../api/user.ts";
import { Share } from "../../api/explorer.ts";
import { useAppDispatch } from "../../redux/hooks.ts";
import SessionManager from "../../session/index.ts";
import CoinStack from "../Icons/CoinStack.tsx";

interface PaidShareGateProps {
  shareInfo: Share;
  // Called after a successful purchase; the caller typically reloads state.
  onPurchased?: () => void;
}

// PaidShareGate renders the purchase CTA for a priced share the visitor has
// not paid for. Anonymous visitors get a sign-in prompt; buyers resume via
// the ticket stored by purchaseShare.
const PaidShareGate = ({ shareInfo, onPurchased }: PaidShareGateProps) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const [loading, setLoading] = useState(false);
  const [credit, setCredit] = useState<CreditInfo | null>(null);

  const user = SessionManager.currentLoginOrNull();
  const price = shareInfo.price ?? 0;
  const paid = shareInfo.paid ?? false;

  useEffect(() => {
    if (user && price > 0 && !paid) {
      dispatch(getCredit())
        .then(setCredit)
        .catch(() => setCredit(null));
    }
  }, [user, price, paid, dispatch]);

  const purchase = useCallback(async () => {
    setLoading(true);
    try {
      await dispatch(purchaseShare(shareInfo.id));
      onPurchased?.();
    } catch (_e) {
      // snackbar already reported by the request layer
    } finally {
      setLoading(false);
    }
  }, [dispatch, shareInfo.id, onPurchased]);

  if (price <= 0 || paid || (user && shareInfo.owner.id == user.user.id)) {
    return null;
  }

  return (
    <Alert
      severity="info"
      icon={<CoinStack />}
      sx={{ alignItems: "center" }}
      action={
        user ? (
          <Button
            variant="contained"
            size="small"
            disabled={loading || (credit != null && credit.credits < price)}
            onClick={purchase}
          >
            {t("application:share.purchaseForPoints", { price })}
          </Button>
        ) : (
          <Button variant="contained" size="small" component={RouterLink} to="/session">
            {t("application:share.signInToPurchase")}
          </Button>
        )
      }
    >
      <Box>
        <Typography variant="body2">{t("application:share.paidShareDes", { price })}</Typography>
        {user && credit != null && (
          <Typography variant="caption" color="text.secondary">
            {t("application:share.yourBalance", { credits: credit.credits })}
          </Typography>
        )}
      </Box>
    </Alert>
  );
};

export default PaidShareGate;
