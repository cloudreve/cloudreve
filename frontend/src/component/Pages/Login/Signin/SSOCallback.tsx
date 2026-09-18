import { Box, CircularProgress, Typography } from "@mui/material";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { sendSSOExchange } from "../../../../api/api.ts";
import { setHeadlessFrameLoading } from "../../../../redux/globalStateSlice.ts";
import { useAppDispatch } from "../../../../redux/hooks.ts";
import { refreshUserSession } from "../../../../redux/thunks/session.ts";
import { useQuery } from "../../../../util";
import DismissCircleFilled from "../../../Icons/DismissCircleFilled.tsx";

const SSOCallback = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const query = useQuery();
  const [missingTicket, setMissingTicket] = useState(false);

  useEffect(() => {
    dispatch(setHeadlessFrameLoading(true));
    const ticket = query.get("ticket");
    const redirect = query.get("redirect");

    const finish = async () => {
      if (!ticket) {
        setMissingTicket(true);
        dispatch(setHeadlessFrameLoading(false));
        return;
      }
      try {
        const loginRes = await dispatch(sendSSOExchange(ticket));
        dispatch(refreshUserSession(loginRes, redirect));
      } catch {
        navigate("/session?sso_error=sso_exchange_failed", { replace: true });
      }
    };
    finish();
  }, []);

  return (
    <Box
      sx={{
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        pt: 7,
        pb: 9,
      }}
    >
      {missingTicket ? (
        <>
          <DismissCircleFilled fontSize="large" color="error" />
          <Typography variant="body2" sx={{ color: (theme) => theme.palette.error.main, mt: 2 }}>
            {t("login.ssoFailed")}
          </Typography>
        </>
      ) : (
        <CircularProgress />
      )}
    </Box>
  );
};

export default SSOCallback;
