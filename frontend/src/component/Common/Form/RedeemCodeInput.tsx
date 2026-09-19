import { Button, InputAdornment } from "@mui/material";
import { useSnackbar } from "notistack";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { redeemGiftCode } from "../../../api/api.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { DenseFilledTextField } from "../StyledComponents.tsx";

export interface RedeemCodeInputProps {
  onRedeemed?: () => void;
}

const RedeemCodeInput = ({ onRedeemed }: RedeemCodeInputProps) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const { enqueueSnackbar } = useSnackbar();
  const [code, setCode] = useState("");
  const [redeeming, setRedeeming] = useState(false);

  const onRedeem = () => {
    if (!code.trim()) {
      return;
    }
    setRedeeming(true);
    dispatch(redeemGiftCode(code.trim()))
      .then(() => {
        enqueueSnackbar(t("setting.giftCodeRedeemed"), { variant: "success" });
        setCode("");
        onRedeemed?.();
      })
      .finally(() => setRedeeming(false));
  };

  return (
    <DenseFilledTextField
      fullWidth
      value={code}
      onChange={(e) => setCode(e.target.value)}
      placeholder={t("setting.giftCodePlaceholder")}
      slotProps={{
        input: {
          endAdornment: (
            <InputAdornment position="end">
              <Button variant="contained" onClick={onRedeem} disabled={redeeming || !code.trim()}>
                {t("setting.redeem")}
              </Button>
            </InputAdornment>
          ),
        },
      }}
    />
  );
};

export default RedeemCodeInput;
