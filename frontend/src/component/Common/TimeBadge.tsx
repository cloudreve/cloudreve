import { Tooltip, Typography, TypographyProps } from "@mui/material";
import { useMemo } from "react";
import dayjs from "dayjs";
import { useTranslation } from "react-i18next";
import { formatLocalTime } from "../../util/datetime.ts";
import TimeAgo from "timeago-react";

const defaultTimeAgoThreshold = 7 * 60 * 24; // 7 days

export interface TimeBadgeProps extends TypographyProps {
  datetime: string | dayjs.Dayjs;
  // If the time difference is less than this value in minutes, use time ago format
  timeAgoThreshold?: number;
}

const TimeBadge = ({ timeAgoThreshold = defaultTimeAgoThreshold, datetime, sx, ...rest }: TimeBadgeProps) => {
  const { t } = useTranslation();
  const timeStr = useMemo(() => {
    if (typeof datetime === "string") {
      return datetime;
    }
    return datetime.toISOString();
  }, [datetime]);

  const useTimeAgo = useMemo(() => {
    let t: dayjs.Dayjs;
    if (typeof datetime === "string") {
      t = dayjs(datetime);
    } else {
      t = datetime;
    }
    const now = dayjs();
    const diff = now.diff(t, "minute");
    return diff < timeAgoThreshold;
  }, [timeAgoThreshold, datetime]);

  const fullTime = useMemo(() => {
    return formatLocalTime(datetime);
  }, [datetime]);
  return (
    <Tooltip title={useTimeAgo ? fullTime : ""}>
      <Typography
        component={"span"}
        sx={{
          ...sx,
        }}
        {...rest}
      >
        {useTimeAgo ? (
          <TimeAgo
            datetime={timeStr}
            locale={t("timeAgoLocaleCode", {
              ns: "common",
            })}
          />
        ) : (
          fullTime
        )}
      </Typography>
    </Tooltip>
  );
};

export default TimeBadge;
