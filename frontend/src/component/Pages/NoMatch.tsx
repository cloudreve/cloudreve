import { LoadingButton } from "@mui/lab";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { Box, Typography } from "@mui/material";
import React from "react";
import DismissCircleFilled from "../Icons/DismissCircleFilled.tsx";

const NoMatch = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();

  return (
    <Box
      sx={{
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        pt: 7,
      }}
    >
      <DismissCircleFilled fontSize={"large"} color={"action"} />
      <Typography
        variant={"h6"}
        sx={{
          color: (theme) => theme.palette.action.active,
          mt: 1,
        }}
      >
        {t("common:pageNotFound")}
      </Typography>
      <LoadingButton
        sx={{
          mt: 7,
        }}
        onClick={() => navigate("/")}
        fullWidth
        variant="contained"
        color="primary"
      >
        <span>{t("application:navbar.backToHomepage")}</span>
      </LoadingButton>
    </Box>
  );
};

export default NoMatch;
