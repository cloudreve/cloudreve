import { useTranslation } from "react-i18next";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { useNavigate } from "react-router-dom";
import { useSnackbar } from "notistack";
import { useQuery } from "../../../util";
import React, { useEffect, useState } from "react";
import { Box, Button, Typography } from "@mui/material";
import PageTitle from "../../../router/PageTitle.tsx";
import CheckmarkCircle from "../../Icons/CheckmarkCircle.tsx";
import { setHeadlessFrameLoading } from "../../../redux/globalStateSlice.ts";
import { sendEmailActivate } from "../../../api/api.ts";

const Activate = () => {
  const { t } = useTranslation();
  const { reg_captcha } = useAppSelector((state) => state.siteConfig.login.config);
  const navigate = useNavigate();
  const { enqueueSnackbar } = useSnackbar();
  const dispatch = useAppDispatch();
  const query = useQuery();

  const [success, setSuccess] = useState(true);
  const [email, setEmail] = useState("");

  useEffect(() => {
    const sign = query.get("sign");
    const id = query.get("id");
    if (!sign || !id) {
      setSuccess(false);
      navigate("/session");
      return;
    }

    dispatch(setHeadlessFrameLoading(true));
    dispatch(sendEmailActivate(id, decodeURIComponent(sign)))
      .then((u) => {
        setEmail(u?.email ?? "");
        setSuccess(true);
      })
      .catch(() => {
        navigate("/session");
      })
      .finally(() => {
        dispatch(setHeadlessFrameLoading(false));
      });
  }, []);

  return (
    <Box>
      <PageTitle title={t("login.activateTitle")} />
      <Box sx={{ overflow: "hidden" }}>
        {success && (
          <>
            <Box
              sx={{
                display: "flex",
                flexDirection: "column",
                alignItems: "center",
                py: 7,
              }}
            >
              <CheckmarkCircle fontSize={"large"} color={"success"} />
              <Typography
                variant={"h6"}
                sx={{
                  color: (theme) => theme.palette.success.main,
                  mt: 1,
                }}
              >
                {t("application:login.accountActivated")}
              </Typography>
            </Box>
            <Button
              onClick={() => navigate("/session?phase=email&email=" + encodeURIComponent(email))}
              sx={{ mt: 2 }}
              variant="contained"
              color="primary"
            >
              {t("login.backToSingIn")}
            </Button>
          </>
        )}
      </Box>
    </Box>
  );
};

export default Activate;
