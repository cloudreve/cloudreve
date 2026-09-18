import {
  Box,
  Grid2 as Grid,
  Link,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
  useMediaQuery,
  useTheme,
} from "@mui/material";
import dayjs from "dayjs";
import duration from "dayjs/plugin/duration";
import { lazy, Suspense, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link as RouterLink } from "react-router-dom";
import { Task } from "../../../../api/dashboard";
import { FileType } from "../../../../api/explorer";
import { TaskStatus, TaskSummary, TaskType } from "../../../../api/workflow";
import { formatDuration } from "../../../../util/datetime";
import FacebookCircularProgress from "../../../Common/CircularProgress";
import { NoWrapTypography, StyledTableContainerPaper } from "../../../Common/StyledComponents";
import TimeBadge from "../../../Common/TimeBadge";
import UserAvatar from "../../../Common/User/UserAvatar";
import FileBadge from "../../../FileManager/FileBadge";
import SettingForm from "../../../Pages/Setting/SettingForm";
import DownloadFileList from "../../../Pages/Tasks/DownloadFileList";
import TaskProgress from "../../../Pages/Tasks/TaskProgress";
import { getTaskStatusText } from "../../../Pages/Tasks/TaskProps";
import UserDialog from "../../User/UserDialog/UserDialog";
import { processTaskContent, userTaskTypes } from "../TaskContent";
import BlobErrors from "./BlobErrors";
dayjs.extend(duration);

const MonacoEditor = lazy(() => import("../../../Viewers/CodeViewer/MonacoEditor"));

const TaskForm = ({ values }: { values: Task }) => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("md"));
  const { t } = useTranslation("dashboard");
  const [userDialogOpen, setUserDialogOpen] = useState(false);
  const [userDialogID, setUserDialogID] = useState<number>(0);

  const userClicked = (e: React.MouseEvent<HTMLElement>) => {
    e.preventDefault();
    setUserDialogOpen(true);
    setUserDialogID(values?.edges?.user?.id ?? 0);
  };

  const processedSummary = useMemo(() => {
    return processTaskContent({ ...values.summary } as TaskSummary, values?.user_hash_id ?? "");
  }, [values]);

  const privateState = useMemo((): any => {
    let res: any = {};
    if (values.private_state) {
      try {
        res = JSON.parse(values.private_state);
      } catch (error) {
        console.error(error);
      }
    }
    return res;
  }, [values]);

  const isUserTask = useMemo(() => {
    return userTaskTypes.includes(values.type ?? "");
  }, [values]);

  return (
    <>
      <UserDialog open={userDialogOpen} onClose={() => setUserDialogOpen(false)} userID={userDialogID} />
      <Box>
        <Grid container spacing={isMobile ? 2 : 3} alignItems={"stretch"}>
          <SettingForm title={t("file.id")} noContainer lgWidth={2}>
            <Typography variant={"body2"} color={"textSecondary"}>
              {values.id}
            </Typography>
          </SettingForm>

          <SettingForm title={t("task.type")} noContainer lgWidth={2}>
            <Typography variant={"body2"} color={"textSecondary"}>
              {t(`task.${values.type}`)}
            </Typography>
          </SettingForm>

          <SettingForm title={t("task.status")} noContainer lgWidth={2}>
            <Typography variant={"body2"} color={"textSecondary"}>
              {getTaskStatusText(values?.status ?? TaskStatus.queued, t)}
            </Typography>
          </SettingForm>

          <SettingForm title={t("task.node")} noContainer lgWidth={2}>
            <NoWrapTypography variant={"body2"} color={"textSecondary"}>
              {values?.node?.name ? (
                <Link component={RouterLink} to={`/admin/node/${values?.node?.id}`} underline="hover">
                  {values?.node?.name}
                </Link>
              ) : (
                "-"
              )}
            </NoWrapTypography>
          </SettingForm>

          <SettingForm title={t("file.creator")} noContainer lgWidth={4}>
            <NoWrapTypography variant={"body2"} color={"textSecondary"}>
              {values?.edges?.user ? (
                <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
                  <UserAvatar
                    sx={{ width: 24, height: 24 }}
                    overwriteTextSize
                    user={{
                      id: values?.user_hash_id ?? "",
                      nickname: values?.edges?.user?.nick ?? "",
                      created_at: values?.edges?.user?.created_at ?? "",
                    }}
                  />
                  <NoWrapTypography variant="inherit">
                    <Link
                      sx={{
                        whiteSpace: "nowrap",
                        overflow: "hidden",
                        textOverflow: "ellipsis",
                      }}
                      onClick={userClicked}
                      underline="hover"
                      href="#/"
                    >
                      {values?.edges?.user?.nick}
                    </Link>
                  </NoWrapTypography>
                </Box>
              ) : (
                "-"
              )}
            </NoWrapTypography>
          </SettingForm>
          <SettingForm title={t("user.createdAt")} noContainer lgWidth={4}>
            <Typography variant={"body2"} color={"textSecondary"}>
              <TimeBadge datetime={values?.created_at ?? ""} variant="inherit" timeAgoThreshold={0} />
            </Typography>
          </SettingForm>

          <SettingForm title={t("task.updatedAt")} noContainer lgWidth={4}>
            <Typography variant={"body2"} color={"textSecondary"}>
              <TimeBadge datetime={values?.updated_at ?? ""} variant="inherit" timeAgoThreshold={0} />
            </Typography>
          </SettingForm>

          <SettingForm title={t("event.correlationId")} noContainer lgWidth={4}>
            <Typography variant={"body2"} color={"textSecondary"}>
              {values?.correlation_id ?? "-"}
            </Typography>
          </SettingForm>

          <SettingForm title={t("application:setting.executeDuration")} noContainer lgWidth={2}>
            <Typography variant={"body2"} color={"textSecondary"}>
              {formatDuration(dayjs.duration((values?.public_state?.executed_duration ?? 0) / 1e6))}
            </Typography>
          </SettingForm>

          <SettingForm title={t("application:setting.retryCount")} noContainer lgWidth={2}>
            <Typography variant={"body2"} color={"textSecondary"}>
              {values?.public_state?.retry_count ?? "-"}
            </Typography>
          </SettingForm>

          {processedSummary?.props?.src && (
            <SettingForm title={t("application:setting.input")} noContainer lgWidth={4}>
              <FileBadge
                variant={"outlined"}
                simplifiedFile={{
                  path: processedSummary?.props.src,
                  type: FileType.file,
                }}
              />
            </SettingForm>
          )}

          {processedSummary?.props?.src_multiple && (
            <SettingForm title={t("application:setting.input")} noContainer lgWidth={4}>
              <Stack
                spacing={0.5}
                sx={{
                  maxHeight: 150,
                  overflowY: "scroll",
                }}
              >
                {processedSummary?.props.src_multiple.map((src, index) => (
                  <FileBadge
                    key={index}
                    variant={"outlined"}
                    simplifiedFile={{
                      path: src,
                      type: FileType.file,
                    }}
                  />
                ))}
              </Stack>
            </SettingForm>
          )}

          {processedSummary?.props?.dst && (
            <SettingForm title={t("application:setting.output")} noContainer lgWidth={4}>
              <FileBadge
                variant={"outlined"}
                simplifiedFile={{
                  path: processedSummary?.props.dst,
                  type: values?.type == TaskType.extract_archive ? FileType.folder : FileType.file,
                }}
              />
            </SettingForm>
          )}

          {values?.public_state?.error && (
            <SettingForm title={t("task.errorMsg")} noContainer lgWidth={12}>
              <Typography variant={"body2"} color={"textSecondary"}>
                {values?.public_state?.error}
              </Typography>
            </SettingForm>
          )}

          {processedSummary?.props?.download && (
            <SettingForm title={t("application:setting.fileList")} noContainer lgWidth={12}>
              <DownloadFileList taskId={""} summary={processedSummary} readonly={true} />
            </SettingForm>
          )}

          {isUserTask && (
            <SettingForm title={t("application:setting.taskProgress")} noContainer lgWidth={12}>
              <TaskProgress
                taskId={values.task_hash_id ?? ""}
                taskStatus={values.status?.toString() ?? ""}
                taskType={values.type ?? ""}
                summary={values.summary}
                node={values.node ? { type: values.node.type?.toString() ?? "" } : undefined}
              />
            </SettingForm>
          )}

          {values?.public_state?.error_history && (
            <SettingForm title={t("application:setting.retryErrorHistory")} noContainer lgWidth={12}>
              <TableContainer component={StyledTableContainerPaper}>
                <Table sx={{ width: "100%" }} size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>#</TableCell>
                      <TableCell>{t("common:error")}</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {values?.public_state?.error_history.map((error, index) => (
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
            </SettingForm>
          )}

          {values.type == TaskType.entity_recycle_routine && privateState?.errors && (
            <SettingForm title={t("task.entityError")} noContainer lgWidth={12}>
              <BlobErrors privateState={privateState} />
            </SettingForm>
          )}

          <SettingForm title={t("event.rawContent")} noContainer lgWidth={12}>
            <Suspense fallback={<FacebookCircularProgress />}>
              <MonacoEditor
                theme={theme.palette.mode === "dark" ? "vs-dark" : "vs"}
                language="json"
                value={JSON.stringify(values, null, 4)}
                height="300px"
                minHeight="300px"
                options={{
                  wordWrap: "on",
                  minimap: { enabled: false },
                  scrollBeyondLastLine: false,
                  readOnly: true,
                }}
              />
            </Suspense>
          </SettingForm>
        </Grid>
      </Box>
    </>
  );
};

export default TaskForm;
