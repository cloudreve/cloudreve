import { Box, Divider, Grid, IconButton, Skeleton, Stack, Typography, useMediaQuery, useTheme } from "@mui/material";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { QueueMetric, QueueType } from "../../../../api/dashboard.ts";
import Setting from "../../../Icons/Setting.tsx";
import { StorageBar, StorageBlock, StoragePart } from "../../../Pages/Setting/StorageSetting.tsx";
import { BorderedCard } from "../../Common/AdminCard.tsx";
import QueueSettingDialog from "./QueueSettingDialog.tsx";

export interface QueueCardProps {
  queue?: QueueType;
  settings: {
    [key: string]: string;
  };
  setSettings: (settings: { [key: string]: string }) => void;
  metrics?: QueueMetric;
  loading: boolean;
}

export const QueueCard = ({ queue, settings, metrics, setSettings, loading }: QueueCardProps) => {
  const { t } = useTranslation("dashboard");
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const [settingDialogOpen, setSettingDialogOpen] = useState(false);

  if (loading) {
    return (
      <Grid item xs={12} md={6} lg={4}>
        <BorderedCard>
          <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
            <Skeleton variant="text" width={150} height={28} />
            <Skeleton variant="circular" width={24} height={24} />
          </Box>
          <Skeleton variant="text" width="80%" height={20} sx={{ mt: 1 }} />
          <Divider sx={{ my: 2 }} />
          <Skeleton variant="rectangular" height={8} width="100%" sx={{ borderRadius: 1 }} />
          <Stack spacing={isMobile ? 1 : 2} direction={isMobile ? "column" : "row"} sx={{ mt: 1 }}>
            {Array.from(Array(5)).map((_, index) => (
              <Skeleton key={index} variant="text" width={isMobile ? "100%" : 80} height={20} />
            ))}
          </Stack>
        </BorderedCard>
      </Grid>
    );
  }

  return (
    <Grid item xs={12} md={6} lg={4}>
      <BorderedCard>
        <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <Typography variant="subtitle1" fontWeight={600}>
            {t(`queue.queueName_${queue}`)}
          </Typography>
          <IconButton size="small" onClick={() => setSettingDialogOpen(true)}>
            <Setting fontSize="small" />
          </IconButton>
        </Box>
        <Typography variant="body2" color="text.secondary">
          {t(`queue.queueName_${queue}Des`)}
        </Typography>
        <Divider sx={{ my: 2 }} />
        {metrics && (
          <>
            <StorageBar>
              <StoragePart
                sx={{
                  backgroundColor: (theme) => theme.palette.success.light,
                  width: `${(metrics.success_tasks / metrics.submitted_tasks) * 100}%`,
                }}
              />
              <StoragePart
                sx={{
                  backgroundColor: (theme) => theme.palette.error.light,
                  width: `${(metrics.failure_tasks / metrics.submitted_tasks) * 100}%`,
                }}
              />
              <StoragePart
                sx={{
                  backgroundColor: (theme) => theme.palette.action.active,
                  width: `${(metrics.suspending_tasks / metrics.submitted_tasks) * 100}%`,
                }}
              />
              <StoragePart
                sx={{
                  backgroundColor: (theme) => theme.palette.info.light,
                  width: `${(metrics.busy_workers / metrics.submitted_tasks) * 100}%`,
                }}
              />
            </StorageBar>
            <Stack spacing={isMobile ? 1 : 2} direction={isMobile ? "column" : "row"} sx={{ mt: 1 }}>
              <Typography variant={"caption"}>
                <StorageBlock
                  sx={{
                    backgroundColor: (theme) => theme.palette.success.light,
                  }}
                />
                {t("queue.success", {
                  count: metrics.success_tasks,
                })}
              </Typography>
              <Typography variant={"caption"}>
                <StorageBlock
                  sx={{
                    backgroundColor: (theme) => theme.palette.error.light,
                  }}
                />
                {t("queue.failed", {
                  count: metrics.failure_tasks,
                })}
              </Typography>
              <Typography variant={"caption"}>
                <StorageBlock
                  sx={{
                    backgroundColor: (theme) => theme.palette.info.light,
                  }}
                />
                {t("queue.busyWorker", {
                  count: metrics.busy_workers,
                })}
              </Typography>
              <Typography variant={"caption"}>
                <StorageBlock
                  sx={{
                    backgroundColor: (theme) => theme.palette.action.active,
                  }}
                />
                {t("queue.suspending", {
                  count: metrics.suspending_tasks,
                })}
              </Typography>
              <Typography variant={"caption"}>
                <StorageBlock
                  sx={{
                    backgroundColor: (theme) => theme.palette.grey[theme.palette.mode === "light" ? 200 : 800],
                  }}
                />
                {t("queue.submited", {
                  count: metrics.submitted_tasks,
                })}
              </Typography>
            </Stack>
          </>
        )}

        {queue && (
          <QueueSettingDialog
            open={settingDialogOpen}
            onClose={() => setSettingDialogOpen(false)}
            queue={queue}
            settings={settings}
            setSettings={setSettings}
          />
        )}
      </BorderedCard>
    </Grid>
  );
};

export default QueueCard;
