import { useTranslation } from "react-i18next";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { useNavigate } from "react-router-dom";
import { useQuery } from "../../../util";
import React, { useEffect, useState } from "react";
import { Box, Button, Typography } from "@mui/material";
import PageTitle from "../../../router/PageTitle.tsx";
import CheckmarkCircle from "../../Icons/CheckmarkCircle.tsx";
import { setHeadlessFrameLoading } from "../../../redux/globalStateSlice.ts";
import { sendEmailChangeActivate } from "../../../api/api.ts";

const ActivateEmailChange = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const dispatch = useAppDispatch();
  const query = useQuery();

  const [success, setSuccess] = useState(true);

  useEffect(() => {
    const sign = query.get("sign");
    const id = query.get("id");
    if (!sign || !id) {
      setSuccess(false);
      navigate("/session");
      return;
    }

    dispatch(setHeadlessFrameLoading(true));
    dispatch(sendEmailChangeActivate(id, decodeURIComponent(sign)))
      .then(() => {
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
      <PageTitle title={t("login.emailChangedTitle")} />
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
                {t("application:login.emailChanged")}
              </Typography>
            </Box>
            <Button
              onClick={() => navigate("/session")}
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

export default ActivateEmailChange;
