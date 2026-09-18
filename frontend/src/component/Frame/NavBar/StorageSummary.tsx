import { LinearProgress, linearProgressClasses, Skeleton, styled, Typography } from "@mui/material";
import { memo, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { updateUserCapacity } from "../../../redux/thunks/filemanager.ts";
import { sizeToString } from "../../../util";
import { RadiusFrame } from "../RadiusFrame.tsx";

const StyledBox = styled(RadiusFrame)(({ theme }) => ({
  padding: theme.spacing(1, 2, 1, 2),
  margin: theme.spacing(0, 2, 0, 2),
}));

const StorageHeaderContainer = styled("div")(() => ({
  display: "flex",
  justifyContent: "space-between",
  alignItems: "center",
}));

const BorderLinearProgress = styled(LinearProgress)<{ warning: boolean }>(({ theme, warning }) => ({
  height: 8,
  borderRadius: 5,
  [`&.${linearProgressClasses.colorPrimary}`]: {
    backgroundColor: theme.palette.grey[theme.palette.mode === "light" ? 200 : 800],
  },
  [`& .${linearProgressClasses.bar}`]: {
    borderRadius: 5,
    backgroundColor: warning ? theme.palette.warning.main : theme.palette.primary.main,
  },
  marginTop: theme.spacing(1),
}));

const StorageSummary = memo(() => {
  const { t } = useTranslation("application");
  const dispatch = useAppDispatch();
  const capacity = useAppSelector((state) => state.fileManager[0].capacity);
  useEffect(() => {
    if (!capacity) {
      dispatch(updateUserCapacity(0));
      return;
    }
  }, [capacity]);
  return (
    <StyledBox withBorder>
      <StorageHeaderContainer>
        <Typography variant={"subtitle2"}>{t("application:navbar.storage")}</Typography>
      </StorageHeaderContainer>
      {capacity && (
        <BorderLinearProgress
          warning={capacity.used > capacity.total}
          variant="determinate"
          value={Math.min(100, (capacity.used / capacity.total) * 100)}
        />
      )}
      {!capacity && <Skeleton sx={{ mt: 1, height: 8 }} variant="rounded" />}
      <Typography variant={"caption"} color={"text.secondary"}>
        {capacity ? (
          `${sizeToString(capacity.used)} / ${sizeToString(capacity.total)}`
        ) : (
          <Skeleton sx={{ width: "50%" }} variant="text" />
        )}
      </Typography>
    </StyledBox>
  );
});

export default StorageSummary;
