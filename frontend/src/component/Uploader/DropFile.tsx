import React from "react";
import Backdrop from "@mui/material/Backdrop";
import { Typography } from "@mui/material";
import { useTranslation } from "react-i18next";
import CloudArrowIUp from "../Icons/CloudArrowIUp.tsx";

export function DropFileBackground({ open }: { open: boolean }) {
  const { t } = useTranslation();
  return (
    <Backdrop
      sx={{
        zIndex: (theme) => theme.zIndex.drawer + 1,
        color: "#fff",
        flexDirection: "column",
      }}
      open={open}
    >
      <div>
        <CloudArrowIUp style={{ fontSize: 80 }} />
      </div>
      <div>
        <Typography variant={"h4"}>{t("uploader.dropFileHere")}</Typography>
      </div>
    </Backdrop>
  );
}
