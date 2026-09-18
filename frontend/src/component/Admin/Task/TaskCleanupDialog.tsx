import { Button, DialogContent, SelectChangeEvent, Stack, Typography } from "@mui/material";
import { DateTimePicker, LocalizationProvider } from "@mui/x-date-pickers";
import { AdapterDayjs } from "@mui/x-date-pickers/AdapterDayjs";
import dayjs from "dayjs";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { sendCleanupTask } from "../../../api/api";
import { CleanupTaskService } from "../../../api/dashboard";
import { TaskStatus, TaskType } from "../../../api/workflow";
import { useAppDispatch } from "../../../redux/hooks";
import DraggableDialog from "../../Dialogs/DraggableDialog";
import SettingForm from "../../Pages/Setting/SettingForm";
import TaskStatusSelector from "./TaskStatusSelector";
import TaskTypeSelector from "./TaskTypeSelector";

export interface TaskCleanupDialogProps {
  open: boolean;
  onClose: () => void;
  onCleanupComplete?: () => void;
}

const TaskCleanupDialog = ({ open, onClose, onCleanupComplete }: TaskCleanupDialogProps) => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const [notAfter, setNotAfter] = useState<dayjs.Dayjs | null>(null);
  const [selectedTypes, setSelectedTypes] = useState<TaskType[]>([]);
  const [selectedStatuses, setSelectedStatuses] = useState<TaskStatus[]>([]);
  const [loading, setLoading] = useState(false);

  const handleCleanup = async () => {
    if (!notAfter) {
      return;
    }

    setLoading(true);
    try {
      const args: CleanupTaskService = {
        not_after: notAfter.toISOString(),
        types: selectedTypes.length > 0 ? selectedTypes : undefined,
        status: selectedStatuses.length > 0 ? selectedStatuses : undefined,
      };

      await dispatch(sendCleanupTask(args));
      onCleanupComplete?.();
      onClose();
    } finally {
      setLoading(false);
    }
  };

  const handleReset = () => {
    setNotAfter(null);
    setSelectedTypes([]);
    setSelectedStatuses([]);
  };

  return (
    <DraggableDialog
      title={t("task.cleanupTasks")}
      dialogProps={{
        open,
        onClose,
        maxWidth: "sm",
        fullWidth: true,
      }}
      showActions={true}
      showCancel={true}
      onAccept={handleCleanup}
      loading={loading}
      disabled={!notAfter}
      okText={t("event.cleanup")}
      secondaryAction={
        <Button onClick={handleReset} disabled={loading}>
          {t("user.reset")}
        </Button>
      }
    >
      <DialogContent>
        <Stack spacing={3}>
          <Typography variant="body2" color="text.secondary">
            {t("task.cleanupTasksDescription")}
          </Typography>

          <SettingForm title={t("task.cleanupNotAfter")} noContainer lgWidth={12}>
            <LocalizationProvider dateAdapter={AdapterDayjs}>
              <DateTimePicker
                value={notAfter}
                onChange={(newValue) => setNotAfter(newValue)}
                slotProps={{
                  textField: {
                    fullWidth: true,
                    size: "small",
                  },
                }}
              />
            </LocalizationProvider>
          </SettingForm>

          <SettingForm title={t("task.type")} noContainer lgWidth={12}>
            <TaskTypeSelector
              value={selectedTypes}
              onChange={(e: SelectChangeEvent<unknown>) => setSelectedTypes(e.target.value as TaskType[])}
              helperText={t("task.cleanupTaskTypesDes")}
              showAllOption={false}
              displayEmpty={true}
            />
          </SettingForm>

          <SettingForm title={t("task.status")} noContainer lgWidth={12}>
            <TaskStatusSelector
              value={selectedStatuses}
              onChange={(e: SelectChangeEvent<unknown>) => setSelectedStatuses(e.target.value as TaskStatus[])}
              helperText={t("task.cleanupTaskStatusesDes")}
              showAllOption={false}
              displayEmpty={true}
            />
          </SettingForm>
        </Stack>
      </DialogContent>
    </DraggableDialog>
  );
};

export default TaskCleanupDialog;
