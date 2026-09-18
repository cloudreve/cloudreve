import { Box } from "@mui/material";
import { useTranslation } from "react-i18next";
import { CustomProps } from "../../../../api/explorer.ts";
import { getPropsContent } from "../../Sidebar/CustomProps/CustomPropsItem.tsx";
import { Condition } from "./ConditionBox.tsx";

export const CustomPropsConditon = ({
  condition,
  onChange,
  option,
}: {
  onChange: (condition: Condition) => void;
  condition: Condition;
  option: CustomProps;
}) => {
  const { t } = useTranslation();
  return (
    <Box
      sx={{
        width: "100%",
      }}
    >
      {getPropsContent(
        {
          props: option,
          id: option.id,
          value: condition.metadata_value ?? "",
        },
        (value) => onChange({ ...condition, metadata_value: value }),
        false,
        false,
        true,
      )}
    </Box>
  );
};
