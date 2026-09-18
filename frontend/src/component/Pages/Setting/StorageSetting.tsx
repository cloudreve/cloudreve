import { Box, Stack, styled, Typography, useMediaQuery, useTheme } from "@mui/material";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { Capacity, UserSettings } from "../../../api/user.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { updateUserCapacity } from "../../../redux/thunks/filemanager.ts";
import { loadSiteConfig } from "../../../redux/thunks/site.ts";
import { sizeToString } from "../../../util";
import SettingForm from "./SettingForm.tsx";

export const StorageBar = styled(Box)(({ theme }) => ({
  height: "10px",
  borderRadius: `${theme.shape.borderRadius}px`,
  width: "100%",
  backgroundColor: theme.palette.grey[theme.palette.mode === "light" ? 200 : 800],
  overflow: "hidden",
}));

export const StoragePart = styled(Box)(() => ({
  transition: "width .6s ease",
  height: "100%",
  fontSize: "12px",
  lineHeight: "20px",
  float: "left",
}));

export const StorageBlock = styled(Box)(() => ({
  display: "inline-block",
  width: "10px",
  height: "10px",
  borderRadius: "50%",
  marginRight: "5px",
}));

export interface StorageSettingProps {
  setting: UserSettings;
}

export const CapacityBar = ({ capacity, forceRow }: { capacity?: Capacity; forceRow?: boolean }) => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("md")) || forceRow;
  const { t } = useTranslation();
  const [storageBreakdown, setStorageBreakdown] = useState({
    used: 0,
    base: 0,
  });
  useEffect(() => {
    let summary = {
      used: 0,
      base: 0,
    };

    if (!capacity) {
      return;
    }

    summary.used = capacity.used / capacity.total;
    summary.base = 1;
    setStorageBreakdown(summary);
  }, [capacity]);

  return (
    <>
      <StorageBar>
        <StoragePart
          sx={{
            backgroundColor: (theme) => theme.palette.warning.light,
            width: `${storageBreakdown.used * 100}%`,
          }}
        />
      </StorageBar>
      <Stack spacing={isMobile ? 1 : 2} direction={isMobile ? "column" : "row"} sx={{ mt: 1 }}>
        <Typography variant={"caption"}>
          <StorageBlock
            sx={{
              backgroundColor: (theme) => theme.palette.warning.light,
            }}
          />
          {t("vas.used", {
            size: sizeToString(capacity?.used ?? 0),
          })}
        </Typography>
        <Typography variant={"caption"}>
          <StorageBlock
            sx={{
              backgroundColor: (theme) => theme.palette.grey[theme.palette.mode === "light" ? 200 : 800],
            }}
          />
          {t("vas.total", {
            size: sizeToString(capacity?.total ?? 0),
          })}
        </Typography>
      </Stack>
    </>
  );
};

const StorageSetting = ({ setting }: StorageSettingProps) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const theme = useTheme();
  const navigate = useNavigate();
  const isMobile = useMediaQuery(theme.breakpoints.down("md"));
  const capacity = useAppSelector((state) => state.fileManager[0].capacity);

  const [storageBreakdown, setStorageBreakdown] = useState({
    used: 0,
    base: 0,
  });

  useEffect(() => {
    dispatch(updateUserCapacity(0));
    dispatch(loadSiteConfig("vas"));
  }, []);

  return (
    <Stack spacing={3}>
      <SettingForm title={t("vas.quota")}>
        <Box sx={{ mt: 1 }}>
          <CapacityBar capacity={capacity} />
        </Box>
      </SettingForm>
    </Stack>
  );
};

export default StorageSetting;
