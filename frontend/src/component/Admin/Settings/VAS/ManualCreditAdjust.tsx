import { Stack } from "@mui/material";
import { useSnackbar } from "notistack";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useAppDispatch } from "../../../../redux/hooks.ts";
import { adminAdjustCredit } from "../../../../api/api.ts";
import { DenseFilledTextField, SecondaryButton } from "../../../Common/StyledComponents.tsx";
import SettingForm from "../../../Pages/Setting/SettingForm.tsx";
import { NoMarginHelperText } from "../Settings.tsx";

const ManualCreditAdjust = () => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const { enqueueSnackbar } = useSnackbar();
  const [email, setEmail] = useState("");
  const [delta, setDelta] = useState(0);
  const [des, setDes] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const onSubmit = () => {
    setSubmitting(true);
    dispatch(adminAdjustCredit({ email: email.trim(), delta, des: des || undefined }))
      .then(() => {
        enqueueSnackbar(t("vas.adjustSuccess"), { variant: "success" });
        setDelta(0);
        setDes("");
      })
      .catch(() => enqueueSnackbar(t("vas.adjustFailed"), { variant: "error" }))
      .finally(() => setSubmitting(false));
  };

  return (
    <Stack spacing={2} direction={{ xs: "column", lg: "row" }} alignItems={{ lg: "flex-end" }}>
      <SettingForm title={t("vas.adjustEmail")} lgWidth={4}>
        <DenseFilledTextField fullWidth value={email} onChange={(e) => setEmail(e.target.value)} />
      </SettingForm>
      <SettingForm title={t("vas.adjustDelta")} lgWidth={3}>
        <DenseFilledTextField
          fullWidth
          type="number"
          value={delta}
          onChange={(e) => setDelta(parseInt(e.target.value) || 0)}
        />
        <NoMarginHelperText>{t("vas.adjustDeltaDes")}</NoMarginHelperText>
      </SettingForm>
      <SettingForm title={t("vas.adjustNote")} lgWidth={4}>
        <DenseFilledTextField fullWidth value={des} onChange={(e) => setDes(e.target.value)} />
      </SettingForm>
      <SecondaryButton
        variant="contained"
        onClick={onSubmit}
        disabled={submitting || !email.trim() || delta === 0}
      >
        {t("vas.adjustSubmit")}
      </SecondaryButton>
    </Stack>
  );
};

export default ManualCreditAdjust;
