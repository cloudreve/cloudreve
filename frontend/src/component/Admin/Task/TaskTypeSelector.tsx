import { Box, FormHelperText, ListItemText, SelectChangeEvent } from "@mui/material";
import { useTranslation } from "react-i18next";
import { TaskType } from "../../../api/workflow";
import { DenseSelect, SquareChip } from "../../Common/StyledComponents";
import { SquareMenuItem } from "../../FileManager/ContextMenu/ContextMenu";

interface TaskTypeSelectorProps {
  value: TaskType[];
  onChange: (event: SelectChangeEvent<unknown>) => void;
  renderValue?: (selected: unknown) => React.ReactNode;
  helperText?: string;
  showAllOption?: boolean;
  allOptionText?: string;
  fullWidth?: boolean;
  displayEmpty?: boolean;
}

const TaskTypeSelector = ({
  value,
  onChange,
  renderValue,
  helperText,
  showAllOption = false,
  allOptionText,
  fullWidth = true,
  displayEmpty = false,
}: TaskTypeSelectorProps) => {
  const { t } = useTranslation("dashboard");

  const defaultRenderValue = (selected: unknown) => {
    const values = Array.isArray(selected) ? selected : [];
    return (
      <Box display="flex" flexWrap="wrap" gap={0.5}>
        {values.map((val) => (
          <SquareChip key={val} label={t(`task.${val}`)} size="small" />
        ))}
      </Box>
    );
  };

  return (
    <>
      <DenseSelect
        fullWidth={fullWidth}
        multiple
        value={value}
        onChange={onChange}
        renderValue={renderValue || defaultRenderValue}
        displayEmpty={displayEmpty}
      >
        {showAllOption && (
          <SquareMenuItem value={[]} disabled>
            <ListItemText
              primary={allOptionText || t("task.allTaskTypes")}
              slotProps={{ primary: { variant: "body2", style: { fontStyle: "italic" } } }}
            />
          </SquareMenuItem>
        )}
        {Object.values(TaskType).map((type) => (
          <SquareMenuItem value={type} key={type}>
            <ListItemText primary={t(`task.${type}`)} slotProps={{ primary: { variant: "body2" } }} />
          </SquareMenuItem>
        ))}
      </DenseSelect>
      {helperText && <FormHelperText>{helperText}</FormHelperText>}
    </>
  );
};

export default TaskTypeSelector;
