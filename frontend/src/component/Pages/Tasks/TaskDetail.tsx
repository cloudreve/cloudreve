import {
  Alert,
  Divider,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from "@mui/material";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { sendCancelTask, sendDeleteTask, sendRetryTask } from "../../../api/api.ts";
import { TaskResponse, TaskStatus } from "../../../api/workflow.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { SecondaryLoadingButton, StyledTableContainerPaper } from "../../Common/StyledComponents.tsx";
import DownloadFileList from "./DownloadFileList.tsx";
import TaskProgress from "./TaskProgress.tsx";
import TaskProps from "./TaskProps.tsx";

export interface TaskDetailProps {
  task: TaskResponse;
  downloading?: boolean;
  onRetried?: () => void;
}

const TaskDetail = ({ task, downloading, onRetried }: TaskDetailProps) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const [retrying, setRetrying] = useState(false);
  const [canceling, setCanceling] = useState(false);

  const retry = () => {
    setRetrying(true);
    dispatch(sendRetryTask(task.id))
      .then(() => onRetried?.())
      .catch(() => {})
      .finally(() => setRetrying(false));
  };

  const cancel = () => {
    setCanceling(true);
    dispatch(sendCancelTask(task.id))
      .then(() => onRetried?.())
      .catch(() => {})
      .finally(() => setCanceling(false));
  };

  const [deleting, setDeleting] = useState(false);

  const deleteRecord = () => {
    setDeleting(true);
    dispatch(sendDeleteTask(task.id))
      .then(() => onRetried?.())
      .catch(() => {})
      .finally(() => setDeleting(false));
  };

  const cancelable = task.status == TaskStatus.queued || task.status == TaskStatus.suspending;
  const deletable =
    task.status == TaskStatus.completed || task.status == TaskStatus.error || task.status == TaskStatus.canceled;
  return (
    <Stack spacing={2}>
      <Stack spacing={1}>
        {task.summary?.props?.download && (
          <>
            <Typography variant={"subtitle1"} fontWeight={600}>
              {t("setting.fileList")}
            </Typography>
            <DownloadFileList downloading={downloading} taskId={task.id} summary={task.summary} />
            <Divider />
          </>
        )}
        <Typography variant={"subtitle1"} fontWeight={600}>
          {t("setting.taskProgress")}
        </Typography>
        {!!task.summary?.props?.failed && (
          <Alert severity={"warning"}>
            {t("setting.partialSuccessWarning", {
              num: task.summary?.props?.failed,
            })}
          </Alert>
        )}
        {cancelable && (
          <SecondaryLoadingButton
            size="small"
            variant="outlined"
            color="error"
            loading={canceling}
            onClick={cancel}
            sx={{ alignSelf: "flex-start" }}
          >
            {t("common:cancel")}
          </SecondaryLoadingButton>
        )}
        {deletable && (
          <SecondaryLoadingButton
            size="small"
            variant="outlined"
            color="error"
            loading={deleting}
            onClick={deleteRecord}
            sx={{ alignSelf: "flex-start" }}
          >
            {t("common:delete")}
          </SecondaryLoadingButton>
        )}
        {task.status == TaskStatus.error && (
          <Alert
            severity={"error"}
            action={
              <SecondaryLoadingButton
                size="small"
                variant="outlined"
                color="error"
                loading={retrying}
                onClick={retry}
              >
                {t("uploader.retry")}
              </SecondaryLoadingButton>
            }
          >
            {task.error}
          </Alert>
        )}
        <TaskProgress
          taskId={task.id}
          taskStatus={task.status}
          taskType={task.type}
          summary={task.summary}
          node={task.node}
        />
        <Divider />
      </Stack>
      <Stack spacing={1}>
        <Typography variant={"subtitle1"} fontWeight={600}>
          {t("setting.taskDetails")}
        </Typography>
        <TaskProps task={task} />
        {task.error_history && <Divider sx={{ pt: 2 }} />}
      </Stack>
      {task.error_history && (
        <Stack spacing={1}>
          <Typography variant={"subtitle1"} fontWeight={600}>
            {t("setting.retryErrorHistory")}
          </Typography>
          <TableContainer component={StyledTableContainerPaper} sx={{ maxHeight: 300 }}>
            <Table sx={{ width: "100%" }} size="small">
              <TableHead>
                <TableRow>
                  <TableCell>#</TableCell>
                  <TableCell>{t("common:error")}</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {task.error_history.map((error, index) => (
                  <TableRow hover key={index} sx={{ "&:last-child td, &:last-child th": { border: 0 } }}>
                    <TableCell component="th" scope="row">
                      {index + 1}
                    </TableCell>
                    <TableCell>{error}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </Stack>
      )}
    </Stack>
  );
};

export default TaskDetail;
