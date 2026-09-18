import { Box } from "@mui/material";
import { useTranslation } from "react-i18next";
import { Condition } from "./ConditionBox.tsx";
import { FilledTextField } from "../../../Common/StyledComponents.tsx";

export const MetadataCondition = ({
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
      <FilledTextField
        variant="filled"
        label={t("application:fileManager.metadataKey")}
        value={condition.metadata_key ?? ""}
        onChange={(e) => onChange({ ...condition, metadata_key: e.target.value })}
        disabled={condition.metadata_key_readonly}
        type="text"
        fullWidth
      />
      <FilledTextField
        variant="filled"
        label={t("application:fileManager.metadataValue")}
        value={condition.metadata_value ?? ""}
        onChange={(e) => onChange({ ...condition, metadata_value: e.target.value })}
        type="text"
        fullWidth
      />
    </Box>
  );
};
