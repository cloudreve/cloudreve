import { Box, FormHelperText, ListItemText, SelectChangeEvent } from "@mui/material";
import { useTranslation } from "react-i18next";
import { TaskStatus } from "../../../api/workflow";
import { DenseSelect, SquareChip } from "../../Common/StyledComponents";
import { SquareMenuItem } from "../../FileManager/ContextMenu/ContextMenu";
import { getTaskStatusText } from "../../Pages/Tasks/TaskProps";

interface TaskStatusSelectorProps {
  value: TaskStatus[];
  onChange: (event: SelectChangeEvent<unknown>) => void;
  renderValue?: (selected: unknown) => React.ReactNode;
  helperText?: string;
  showAllOption?: boolean;
  allOptionText?: string;
  fullWidth?: boolean;
  displayEmpty?: boolean;
}

const TaskStatusSelector = ({
  value,
  onChange,
  renderValue,
  helperText,
  showAllOption = false,
  allOptionText,
  fullWidth = true,
  displayEmpty = false,
}: TaskStatusSelectorProps) => {
  const { t } = useTranslation("dashboard");

  const defaultRenderValue = (selected: unknown) => {
    const values = Array.isArray(selected) ? selected : [];
    return (
      <Box display="flex" flexWrap="wrap" gap={0.5}>
        {values.map((val) => (
          <SquareChip key={val} label={getTaskStatusText(val as TaskStatus, t)} size="small" />
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
              primary={allOptionText || t("task.allTaskStatuses")}
              slotProps={{ primary: { variant: "body2", style: { fontStyle: "italic" } } }}
            />
          </SquareMenuItem>
        )}
        {Object.values(TaskStatus).map((status) => (
          <SquareMenuItem value={status} key={status}>
            <ListItemText primary={getTaskStatusText(status, t)} slotProps={{ primary: { variant: "body2" } }} />
          </SquareMenuItem>
        ))}
      </DenseSelect>
      {helperText && <FormHelperText>{helperText}</FormHelperText>}
    </>
  );
};

export default TaskStatusSelector;
