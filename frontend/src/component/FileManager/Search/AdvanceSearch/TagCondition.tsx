import { Autocomplete, Chip } from "@mui/material";
import { useTranslation } from "react-i18next";
import { FilledTextField } from "../../../Common/StyledComponents.tsx";
import { Condition } from "./ConditionBox.tsx";

export const TagCondition = ({
  condition,
  onChange,
}: {
  onChange: (condition: Condition) => void;
  condition: Condition;
}) => {
  const { t } = useTranslation();
  return (
    <>
      <Autocomplete
        multiple
        options={[]}
        value={condition.tags ?? []}
        autoSelect
        freeSolo
        onChange={(_, value) => onChange({ ...condition, tags: value })}
        renderTags={(value: readonly string[], getTagProps) =>
          value.map((option: string, index: number) => {
            const { key, ...tagProps } = getTagProps({ index });
            return <Chip variant="outlined" label={option} key={key} {...tagProps} />;
          })
        }
        renderInput={(params) => (
          <FilledTextField
            {...params}
            sx={{
              "& .MuiInputBase-root": {
                py: 1,
              },
            }}
            variant="filled"
            autoFocus
            helperText={t("application:modals.enterForNewTag")}
            margin="dense"
            type="text"
            fullWidth
          />
        )}
      />
    </>
  );
};
