import { Box, Button, ListItemText, Popover, PopoverProps, Stack } from "@mui/material";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { TaskStatus, TaskType } from "../../../api/workflow";
import { DenseFilledTextField, DenseSelect } from "../../Common/StyledComponents";
import { SquareMenuItem } from "../../FileManager/ContextMenu/ContextMenu";
import SettingForm from "../../Pages/Setting/SettingForm";
import { getTaskStatusText } from "../../Pages/Tasks/TaskProps";

export interface TaskFilterPopoverProps extends PopoverProps {
  status: string;
  setStatus: (status: string) => void;
  user: string;
  setUser: (user: string) => void;
  correlationID: string;
  setCorrelationID: (correlationID: string) => void;
  type: string;
  setType: (type: string) => void;
  clearFilters: () => void;
}

const TaskFilterPopover = ({
  status,
  setStatus,
  user,
  setUser,
  correlationID,
  setCorrelationID,
  type,
  setType,
  clearFilters,
  onClose,
  open,
  ...rest
}: TaskFilterPopoverProps) => {
  const { t } = useTranslation("dashboard");

  // Create local state to track changes before applying
  const [localStatus, setLocalStatus] = useState(status);
  const [localUser, setLocalUser] = useState(user);
  const [localCorrelationID, setLocalCorrelationID] = useState(correlationID);
  const [localType, setLocalType] = useState(type);

  // Initialize local state when popup opens
  useEffect(() => {
    if (open) {
      setLocalStatus(status);
      setLocalUser(user);
      setLocalCorrelationID(correlationID);
      setLocalType(type);
    }
  }, [open]);

  // Apply filters and close popover
  const handleApplyFilters = () => {
    setStatus(localStatus);
    setUser(localUser);
    setCorrelationID(localCorrelationID);
    setType(localType);
    onClose?.({}, "backdropClick");
  };

  // Reset filters and close popover
  const handleResetFilters = () => {
    setLocalStatus("");
    setLocalUser("");
    setLocalCorrelationID("");
    setLocalType("");
    clearFilters();
    onClose?.({}, "backdropClick");
  };

  return (
    <Popover
      anchorOrigin={{
        vertical: "bottom",
        horizontal: "left",
      }}
      transformOrigin={{
        vertical: "top",
        horizontal: "left",
      }}
      slotProps={{
        paper: {
          sx: {
            p: 2,
            width: 300,
            maxWidth: "100%",
          },
        },
      }}
      onClose={onClose}
      open={open}
      {...rest}
    >
      <Stack spacing={2}>
        <SettingForm title={t("event.userID")} noContainer lgWidth={12}>
          <DenseFilledTextField
            fullWidth
            value={localUser}
            onChange={(e) => setLocalUser(e.target.value)}
            placeholder={t("user.emptyNoFilter")}
            size="small"
          />
        </SettingForm>

        <SettingForm title={t("event.correlationId")} noContainer lgWidth={12}>
          <DenseFilledTextField
            fullWidth
            value={localCorrelationID}
            onChange={(e) => setLocalCorrelationID(e.target.value)}
            placeholder={t("user.emptyNoFilter")}
            size="small"
          />
        </SettingForm>

        <SettingForm title={t("task.type")} noContainer lgWidth={12}>
          <DenseSelect
            fullWidth
            displayEmpty
            value={localType != "" ? localType : " "}
            onChange={(e) => setLocalType(e.target.value === " " ? "" : (e.target.value as string))}
          >
            {[
              TaskType.create_archive,
              TaskType.extract_archive,
              TaskType.remote_download,
              TaskType.media_metadata,
              TaskType.entity_recycle_routine,
              TaskType.explicit_entity_recycle,
              TaskType.upload_sentinel_check,
              TaskType.import,
              TaskType.full_text_rebuild,
              TaskType.full_text_index,
              TaskType.full_text_copy,
              TaskType.full_text_change_owner,
              TaskType.full_text_delete,
            ].map((type) => (
              <SquareMenuItem key={type} value={type}>
                <ListItemText
                  primary={t(`task.${type}`)}
                  slotProps={{
                    primary: {
                      variant: "body2",
                    },
                  }}
                />
              </SquareMenuItem>
            ))}
            <SquareMenuItem value={" "}>
              <ListItemText
                primary={<em>{t("user.all")}</em>}
                slotProps={{
                  primary: {
                    variant: "body2",
                  },
                }}
              />
            </SquareMenuItem>
          </DenseSelect>
        </SettingForm>

        <SettingForm title={t("task.status")} noContainer lgWidth={12}>
          <DenseSelect
            fullWidth
            displayEmpty
            value={localStatus == "" ? " " : localStatus}
            onChange={(e) => setLocalStatus(e.target.value === " " ? "" : (e.target.value as string))}
          >
            {[
              TaskStatus.queued,
              TaskStatus.processing,
              TaskStatus.suspending,
              TaskStatus.error,
              TaskStatus.completed,
            ].map((status) => (
              <SquareMenuItem key={status} value={status}>
                <ListItemText
                  primary={getTaskStatusText(status, t)}
                  slotProps={{
                    primary: {
                      variant: "body2",
                    },
                  }}
                />
              </SquareMenuItem>
            ))}
            <SquareMenuItem value={" "}>
              <ListItemText
                primary={<em>{t("user.all")}</em>}
                slotProps={{
                  primary: {
                    variant: "body2",
                  },
                }}
              />
            </SquareMenuItem>
          </DenseSelect>
        </SettingForm>
        <Box display="flex" justifyContent="space-between">
          <Button variant="outlined" size="small" onClick={handleResetFilters}>
            {t("user.reset")}
          </Button>
          <Button variant="contained" size="small" onClick={handleApplyFilters}>
            {t("user.apply")}
          </Button>
        </Box>
      </Stack>
    </Popover>
  );
};

export default TaskFilterPopover;
