import dayjs from "dayjs";
import { default as duration, default as plugin } from "dayjs/plugin/duration";
import timezone from "dayjs/plugin/timezone";
import utc from "dayjs/plugin/utc";
import i18next from "../i18n";
import SessionManager, { UserSettings } from "../session";

dayjs.extend(utc);
dayjs.extend(timezone);
dayjs.extend(duration);

let userTimezone = "";
try {
  userTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
} catch (e) {
  console.log(e);
}

if (!userTimezone || userTimezone == "Etc/Unknown") {
  userTimezone = "Asia/Shanghai";
}

const preferTimeZone: string | undefined = SessionManager.get(UserSettings.TimeZone);
export let timeZone = preferTimeZone ? preferTimeZone : userTimezone;

export function refreshTimeZone() {
  timeZone = SessionManager.get(UserSettings.TimeZone);
  timeZone = timeZone ? timeZone : userTimezone;
}

export function formatLocalTime(time: string | dayjs.Dayjs, ignoreTz?: boolean) {
  let t: dayjs.Dayjs;
  if (typeof time === "string") {
    t = dayjs(time);
  } else {
    t = time;
  }

  return i18next.t("intlDateTime", {
    ns: "common",
    val: ignoreTz ? t.utc().tz(timeZone, true).toDate() : t.toDate(),
    formatParams: {
      val: {
        year: "numeric",
        month: "numeric",
        day: "numeric",
        hour: "numeric",
        minute: "numeric",
        second: "numeric",
        timeZone: ignoreTz ? undefined : timeZone,
      },
    },
  });
}

export function formatDuration(duration: plugin.Duration): string {
  const s = duration.asSeconds();
  const m = duration.asMinutes();
  const h = duration.asHours();
  if (s < 60) {
    return duration.format(i18next.t("seconds", { ns: "common" }));
  } else if (m < 60) {
    return duration.format(i18next.t("minutes", { ns: "common" }));
  } else if (h < 24) {
    return duration.format(i18next.t("hours", { ns: "common" }));
  } else {
    return i18next.t("days", { ns: "common", d: Math.round(duration.asDays()) });
  }
}

// export function validateTimeZone(name) {
//   try {
//     dayjs().tz(name).format();
//   } catch (e) {
//     return false;
//   }
//   return true;
// }
