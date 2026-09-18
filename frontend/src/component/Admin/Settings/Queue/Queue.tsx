import { Box, Grid, Stack } from "@mui/material";
import { useContext, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { getQueueMetrics } from "../../../../api/api.ts";
import { QueueMetric } from "../../../../api/dashboard.ts";
import { useAppDispatch } from "../../../../redux/hooks.ts";
import { SecondaryButton } from "../../../Common/StyledComponents.tsx";
import ArrowSync from "../../../Icons/ArrowSync.tsx";
import { SettingContext } from "../SettingWrapper.tsx";
import QueueCard from "./QueueCard.tsx";

const Queue = () => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const { formRef, setSettings, values } = useContext(SettingContext);
  const [metrics, setMetrics] = useState<QueueMetric[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchQueueMetrics = () => {
    setLoading(true);
    dispatch(getQueueMetrics())
      .then((res) => {
        setMetrics(res);
      })
      .finally(() => {
        setLoading(false);
      });
  };

  useEffect(() => {
    fetchQueueMetrics();
  }, []);

  return (
    <Box component={"form"} ref={formRef} sx={{ p: 2, pt: 0 }}>
      <Stack direction="row" spacing={2} sx={{ mb: 2 }}>
        <SecondaryButton onClick={fetchQueueMetrics} disabled={loading} variant={"contained"} startIcon={<ArrowSync />}>
          {t("node.refresh")}
        </SecondaryButton>
      </Stack>
      <Grid container spacing={2}>
        {!loading &&
          metrics.map((metric) => (
            <QueueCard
              key={metric.name}
              metrics={metric}
              queue={metric.name}
              settings={values}
              setSettings={setSettings}
              loading={loading}
            />
          ))}
        {loading &&
          Array.from(Array(5)).map((_, index) => (
            <QueueCard key={`loading-${index}`} settings={values} setSettings={setSettings} loading={true} />
          ))}
      </Grid>
    </Box>
  );
};

export default Queue;
