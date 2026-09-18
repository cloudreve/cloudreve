import { Autocomplete, Box, Chip, FormControlLabel, styled } from "@mui/material";
import { useTranslation } from "react-i18next";
import { FilledTextField, StyledCheckbox } from "../../../Common/StyledComponents.tsx";
import { Condition } from "./ConditionBox.tsx";

export const StyledBox = styled(Box)(({ theme }) => ({
  padding: `${theme.spacing(1)} ${theme.spacing(2)}`,
  paddingBottom: theme.spacing(2),
  borderRadius: theme.shape.borderRadius,
  border: `1px solid ${theme.palette.divider}`,
}));
export const FileNameCondition = ({
  condition,
  onChange,
  onNameConditionAdded,
}: {
  onChange: (condition: Condition) => void;
  condition: Condition;
  onNameConditionAdded: (_e: any, newValue: string[]) => void;
}) => {
  const { t } = useTranslation();
  return (
    <>
      <Autocomplete
        multiple
        id="tags-filled"
        options={[]}
        value={condition.names ?? []}
        autoSelect
        freeSolo
        onChange={onNameConditionAdded}
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
            helperText={t("application:navbar.fileNameKeywordsHelp")}
            margin="dense"
            type="text"
            fullWidth
          />
        )}
      />
      <Box sx={{ pt: 1, pl: "10px" }}>
        <FormControlLabel
          slotProps={{
            typography: {
              variant: "body2",
              pl: 1,
              color: "text.secondary",
            },
          }}
          sx={{ mr: 4 }}
          control={
            <StyledCheckbox
              onChange={(e) => {
                onChange({
                  ...condition,
                  case_folding: e.target.checked,
                });
              }}
              disableRipple
              checked={condition.case_folding}
              size="small"
            />
          }
          label={t("application:navbar.caseFolding")}
        />
        <FormControlLabel
          slotProps={{
            typography: {
              variant: "body2",
              pl: 1,
              color: "text.secondary",
            },
          }}
          control={
            <StyledCheckbox
              onChange={(e) => {
                onChange({
                  ...condition,
                  name_op_or: !e.target.checked,
                });
              }}
              disableRipple
              checked={!condition.name_op_or}
              size="small"
            />
          }
          label={t("application:navbar.notNameOpOr")}
        />
      </Box>
    </>
  );
};
