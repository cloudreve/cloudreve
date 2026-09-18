import { Box, Checkbox, IconButton, Link, Skeleton, TableCell, TableRow } from "@mui/material";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Link as RouterLink, useNavigate } from "react-router-dom";
import { batchDeleteTasks } from "../../../api/api";
import { Task } from "../../../api/dashboard";
import { useAppDispatch } from "../../../redux/hooks";
import { confirmOperation } from "../../../redux/thunks/dialog";
import { NoWrapTableCell, NoWrapTypography } from "../../Common/StyledComponents";
import TimeBadge from "../../Common/TimeBadge";
import UserAvatar from "../../Common/User/UserAvatar";
import Delete from "../../Icons/Delete";
import TaskSummaryStatus from "../../Pages/Tasks/TaskSummaryStatus";
import { TaskContent } from "./TaskContent";

export interface TaskRowProps {
  task?: Task;
  loading?: boolean;
  deleting?: boolean;
  selected?: boolean;
  onDelete?: () => void;
  onDetails?: (id: number) => void;
  onSelect?: (id: number) => void;
  openUserDialog?: (id: number) => void;
  openEntity?: (entityID: number) => void;
  openFile?: (fileID: number) => void;
  openTask?: (taskID: number) => void;
}

const TaskRow = ({
  task,
  loading,
  deleting,
  selected,
  onDelete,
  onDetails,
  onSelect,
  openUserDialog,
  openEntity,
  openFile,
}: TaskRowProps) => {
  const navigate = useNavigate();
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const [deleteLoading, setDeleteLoading] = useState(false);
  const [openLoading, setOpenLoading] = useState(false);
  const onRowClick = () => {
    onDetails?.(task?.id ?? 0);
  };

  const onDeleteClick = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    e.preventDefault();
    dispatch(confirmOperation(t("task.confirmDelete"))).then(() => {
      if (task?.id) {
        setDeleteLoading(true);
        dispatch(batchDeleteTasks({ ids: [task.id] }))
          .then(() => {
            onDelete?.();
          })
          .finally(() => {
            setDeleteLoading(false);
          });
      }
    });
  };

  const onSelectClick = (e: React.MouseEvent<HTMLElement>) => {
    e.stopPropagation();
    e.preventDefault();
    onSelect?.(task?.id ?? 0);
  };

  if (loading) {
    return (
      <TableRow sx={{ height: "44px" }}>
        <NoWrapTableCell>
          <Skeleton variant="circular" width={24} height={24} />
        </NoWrapTableCell>
        <NoWrapTableCell>
          <Skeleton variant="text" width={30} />
        </NoWrapTableCell>
        <NoWrapTableCell>
          <Skeleton variant="text" width={200} />
        </NoWrapTableCell>
        <NoWrapTableCell>
          <Skeleton variant="text" width={50} />
        </NoWrapTableCell>
        <NoWrapTableCell>
          <Skeleton variant="text" width={50} />
        </NoWrapTableCell>
        <NoWrapTableCell>
          <Skeleton variant="text" width={100} />
        </NoWrapTableCell>
        <NoWrapTableCell>
          <Skeleton variant="text" width={100} />
        </NoWrapTableCell>
        <NoWrapTableCell>
          <Skeleton variant="circular" width={24} height={24} />
        </NoWrapTableCell>
      </TableRow>
    );
  }

  const stopPropagation = (e: React.MouseEvent<HTMLElement>) => {
    e.stopPropagation();
  };

  const userClicked = (e: React.MouseEvent<HTMLElement>) => {
    e.stopPropagation();
    e.preventDefault();
    openUserDialog?.(task?.edges?.user?.id ?? 0);
  };

  return (
    <TableRow hover key={task?.id} sx={{ cursor: "pointer" }} onClick={onRowClick} selected={selected}>
      <TableCell padding="checkbox">
        <Checkbox
          disabled={deleting}
          size="small"
          disableRipple
          color="primary"
          onClick={onSelectClick}
          checked={selected}
        />
      </TableCell>
      <NoWrapTableCell>
        <NoWrapTypography variant="inherit">{task?.id}</NoWrapTypography>
      </NoWrapTableCell>
      <TableCell>{task && <TaskContent task={task} openEntity={openEntity} openFile={openFile} />}</TableCell>
      <NoWrapTableCell>
        <TaskSummaryStatus
          simplified
          type={task?.type ?? ""}
          status={task?.status ?? ""}
          summary={task?.summary}
          error={task?.public_state?.error}
        />
      </NoWrapTableCell>
      <NoWrapTableCell>
        {task?.edges?.user && (
          <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
            <UserAvatar
              sx={{ width: 24, height: 24 }}
              overwriteTextSize
              user={{
                id: task?.user_hash_id ?? "",
                nickname: task?.edges?.user?.nick ?? "",
                created_at: task?.edges?.user?.created_at ?? "",
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
                {task?.edges?.user?.nick}
              </Link>
            </NoWrapTypography>
          </Box>
        )}
      </NoWrapTableCell>
      <NoWrapTableCell>
        <NoWrapTypography variant="inherit">
          {task?.node?.name ? (
            <Link component={RouterLink} to={`/admin/node/${task?.node?.id}`} underline="hover">
              {task?.node?.name}
            </Link>
          ) : (
            "-"
          )}
        </NoWrapTypography>
      </NoWrapTableCell>
      <NoWrapTableCell>
        <TimeBadge datetime={task?.created_at ?? ""} variant="inherit" timeAgoThreshold={0} />
      </NoWrapTableCell>
      <NoWrapTableCell>
        <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
          <IconButton size="small" onClick={onDeleteClick} disabled={deleteLoading || deleting}>
            <Delete fontSize="small" />
          </IconButton>
        </Box>
      </NoWrapTableCell>
    </TableRow>
  );
};

export default TaskRow;
