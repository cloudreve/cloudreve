import { Box } from "@mui/material";
import { useTranslation } from "react-i18next";
import { Condition } from "./ConditionBox.tsx";
import { DateTimePicker, LocalizationProvider } from "@mui/x-date-pickers";
import { AdapterDayjs } from "@mui/x-date-pickers/AdapterDayjs";
import dayjs from "dayjs";

export const DateTimeCondition = ({
  condition,
  onChange,
  field,
}: {
  onChange: (condition: Condition) => void;
  condition: Condition;
  field: string;
}) => {
  const { t } = useTranslation();
  return (
    <LocalizationProvider dateAdapter={AdapterDayjs}>
      <Box
        sx={{
          display: "flex",
          gap: 1,
        }}
      >
        <DateTimePicker
          localeText={{
            clearButtonLabel: "Vider",
          }}
          label={t("application:navbar.notBefore")}
          // @ts-ignore
          value={dayjs.unix(condition[field + "_gte"] as number)}
          onChange={(newValue) =>
            onChange({
              ...condition,
              [field + "_gte"]: newValue ? newValue.unix() : 0,
            })
          }
        />
        <DateTimePicker
          label={t("application:navbar.notAfter")}
          // @ts-ignore
          value={dayjs.unix(condition[field + "_lte"] as number)}
          onChange={(newValue) =>
            onChange({
              ...condition,
              [field + "_lte"]: newValue ? newValue.unix() : 0,
            })
          }
        />
      </Box>
    </LocalizationProvider>
  );
};
