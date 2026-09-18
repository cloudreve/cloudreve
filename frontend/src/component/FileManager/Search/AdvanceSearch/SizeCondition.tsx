import { Box } from "@mui/material";
import { useTranslation } from "react-i18next";
import { Condition } from "./ConditionBox.tsx";
import SizeInput from "../../../Common/SizeInput.tsx";

export const SizeCondition = ({
  condition,
  onChange,
}: {
  onChange: (condition: Condition) => void;
  condition: Condition;
}) => {
  const { t } = useTranslation();
  return (
    <Box
      sx={{
        display: "flex",
        gap: 1,
      }}
    >
      <SizeInput
        label={t("application:navbar.minimum")}
        value={condition.size_gte ?? 0}
        onChange={(e) => onChange({ ...condition, size_gte: e })}
        inputProps={{
          fullWidth: true,
        }}
      />
      <SizeInput
        label={t("application:navbar.maximum")}
        value={condition.size_lte ?? 0}
        onChange={(e) => onChange({ ...condition, size_lte: e })}
        inputProps={{
          fullWidth: true,
        }}
      />
    </Box>
  );
};
