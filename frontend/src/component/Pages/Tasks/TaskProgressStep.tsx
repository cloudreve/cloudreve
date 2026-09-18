import { Step, StepIcon, StepIconProps, StepLabel, Typography } from "@mui/material";
import { StepProps } from "@mui/material/Step/Step";
import { bindHover, bindPopover } from "material-ui-popup-state";
import { usePopupState } from "material-ui-popup-state/hooks";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { TaskStatus } from "../../../api/workflow.ts";
import FacebookCircularProgress from "../../Common/CircularProgress.tsx";
import CheckCircleFilled from "../../Icons/CheckCircleFilled.tsx";
import DismissCircleFilled from "../../Icons/DismissCircleFilled.tsx";
import StepProgressPopover from "./StepProgressPopover.tsx";

interface ProgressStepIconProps extends StepIconProps {}

const ProgressStepIcon = (status: string) => (props: ProgressStepIconProps) => {
  const { active, completed, icon, ...rest } = props;

  let newIcon = icon;
  if (active) {
    newIcon = <FacebookCircularProgress sx={{ pt: "5px" }} size={24} />;
    if (status == TaskStatus.error) {
      newIcon = <DismissCircleFilled sx={{ fontSize: 28.5, color: (theme) => theme.palette.error.main }} />;
    }
  } else if (completed) {
    newIcon = <CheckCircleFilled sx={{ fontSize: 28.5, color: (theme) => theme.palette.primary.main }} />;
  }

  if (active && status == TaskStatus.error) {
    newIcon = <DismissCircleFilled sx={{ fontSize: 28.5, color: (theme) => theme.palette.error.main }} />;
  }

  if (active && status == TaskStatus.canceled) {
    newIcon = (
      <DismissCircleFilled
        sx={{
          fontSize: 28.5,
          color: (theme) => theme.palette.action.active,
        }}
      />
    );
  }

  return <StepIcon icon={newIcon} active={active} completed={completed} {...rest} />;
};

export interface TaskProgressStepProps extends StepProps {
  taskId: string;
  taskStatus: string;
  title: string;
  description?: string;
  showProgress?: boolean;
  progressing?: boolean;
}

const TaskProgressStep = ({
  taskId,
  taskStatus,
  title,
  description,
  showProgress,
  progressing,
  ...rest
}: TaskProgressStepProps) => {
  const popupState = usePopupState({
    variant: "popover",
    popupId: `progress_${taskId}_${title}`,
  });
  const { open, ...restPopup } = bindPopover(popupState);

  const StepIconComponent = useMemo(() => {
    return ProgressStepIcon(taskStatus);
  }, [taskStatus]);
  const { t } = useTranslation();
  return (
    <>
      <Step {...rest} {...(showProgress && progressing ? bindHover(popupState) : [])}>
        <StepLabel
          optional={description ? <Typography variant={"caption"}>{t(description)}</Typography> : undefined}
          slots={{
            stepIcon: StepIconComponent,
          }}
        >
          {t(title)}
        </StepLabel>
      </Step>
      {showProgress && progressing && <StepProgressPopover taskId={taskId} open={open} {...restPopup} />}
    </>
  );
};

export default TaskProgressStep;
