import { useAppDispatch, useAppSelector } from "../../redux/hooks.ts";
import { Badge, Box, IconButton, Tooltip } from "@mui/material";
import { useTranslation } from "react-i18next";
import UploadFilled from "../Icons/UploadFilled.tsx";
import { useCallback } from "react";
import { openUploadTaskList } from "../../redux/globalStateSlice.ts";

export const TaskListIconButton = () => {
  const dispatch = useAppDispatch();
  const { t } = useTranslation();
  const queued = useAppSelector((state) => state.globalState.uploadTaskCount);

  const openList = useCallback(() => {
    dispatch(openUploadTaskList());
  }, [dispatch]);

  if (!queued) {
    return null;
  }

  return (
    <Box sx={{ position: "relative" }}>
      <Tooltip title={t("uploader.uploadList")} enterDelay={0}>
        <IconButton onClick={openList} size="large">
          <Badge
            sx={{ fontSize: (theme) => theme.typography.body2.fontSize }}
            badgeContent={queued}
            color={"secondary"}
          >
            <UploadFilled />
          </Badge>
        </IconButton>
      </Tooltip>
    </Box>
  );
};
