import {
  Box,
  darken,
  lighten,
  Skeleton,
  styled,
  SvgIconProps,
  Theme,
  Typography,
  useMediaQuery,
  useTheme,
} from "@mui/material";
import MuiAccordion, { AccordionProps } from "@mui/material/Accordion";
import MuiAccordionDetails from "@mui/material/AccordionDetails";
import MuiAccordionSummary, { AccordionSummaryProps } from "@mui/material/AccordionSummary";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useInView } from "react-intersection-observer";
import { FileType } from "../../../api/explorer.ts";
import { TaskResponse, TaskType } from "../../../api/workflow.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { DefaultButton } from "../../Common/StyledComponents.tsx";
import FileIcon from "../../FileManager/Explorer/FileIcon.tsx";
import Archive from "../../Icons/Archive.tsx";
import ArchiveArrow from "../../Icons/ArchiveArrow.tsx";
import ArrowImport from "../../Icons/ArrowImport.tsx";
import StorageOutlined from "../../Icons/StorageOutlined.tsx";
import TaskDetail from "./TaskDetail.tsx";
import TaskSummaryStatus from "./TaskSummaryStatus.tsx";
import TaskSummaryTitle from "./TaskSummaryTitle.tsx";

const Accordion = styled((props: AccordionProps) => <MuiAccordion disableGutters elevation={0} square {...props} />)(({
  theme,
  expanded,
}) => {
  const bgColor = expanded
    ? theme.palette.mode == "light"
      ? "rgba(0, 0, 0, 0.06)"
      : "rgba(255, 255, 255, 0.09)"
    : "initial";
  return {
    borderRadius: theme.shape.borderRadius,
    backgroundColor: bgColor,
    "&::before": {
      display: "none",
    },
    boxShadow: expanded ? `0 0 0 1px ${theme.palette.divider}` : "none",
    transition: "all 250ms cubic-bezier(0.4, 0, 0.2, 1) 0ms",
    marginBottom: theme.spacing(1),
  };
});

const AccordionSummary = styled((props: AccordionSummaryProps) => <MuiAccordionSummary {...props} />)(() => ({
  flexDirection: "row-reverse",
  minHeight: 0,
  padding: 0,
  "& .MuiAccordionSummary-content": {
    margin: 0,
  },
}));

const AccordionDetails = styled(MuiAccordionDetails)(({ theme }) => ({
  padding: theme.spacing(2),
  borderRadius: `0 0 ${theme.shape.borderRadius}px ${theme.shape.borderRadius}px`,
  backgroundColor: theme.palette.background.default,
}));

export const getProgressColor = (theme: Theme) =>
  theme.palette.mode === "dark" ? darken(theme.palette.primary.main, 0.4) : lighten(theme.palette.primary.main, 0.85);

export const SummaryButton = styled(DefaultButton)<{
  expanded: boolean;
  percentage?: number;
}>(({ theme, expanded, percentage }) => {
  percentage = percentage ?? 0;
  const bgColor = theme.palette.mode == "light" ? "rgba(0, 0, 0, 0.06)" : "rgba(255, 255, 255, 0.09)";
  const progressColor = getProgressColor(theme);
  const progressBgColor = !expanded ? bgColor : "rgba(0,0,0,0)";
  return {
    minHeight: 48,
    justifyContent: "flex-start",
    transition: "all 250ms cubic-bezier(0.4, 0, 0.2, 1) 0ms",
    borderRadius: expanded
      ? `${theme.shape.borderRadius}px ${theme.shape.borderRadius}px 0 0`
      : `${theme.shape.borderRadius}px`,
    backgroundColor: bgColor,
    background: `linear-gradient(to right, ${progressColor} 0%,${progressColor} ${percentage}%,${progressBgColor} ${percentage}%,${progressBgColor} 100%)`,
    "&:hover": {
      backgroundColor: theme.palette.mode == "light" ? "rgba(0, 0, 0, 0.09)" : "rgba(255, 255, 255, 0.13)",
    },
  };
});

export interface TaskCardProps {
  loading?: boolean;
  showProgress?: boolean;
  task?: TaskResponse;
  onLoad?: () => void;
}

const taskIconsMap: {
  [key: string]: (props: SvgIconProps) => JSX.Element;
} = {
  [TaskType.create_archive]: Archive,
  [TaskType.extract_archive]: ArchiveArrow,
  [TaskType.relocate]: StorageOutlined,
  [TaskType.import]: ArrowImport,
};

const TaskCard = ({ loading, showProgress, onLoad, task }: TaskCardProps) => {
  const { t } = useTranslation();
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("md"));
  const dispatch = useAppDispatch();
  const { ref, inView } = useInView({
    rootMargin: "200px 0px",
    triggerOnce: true,
    skip: !loading,
  });

  const [expanded, setExpanded] = useState(false);

  useEffect(() => {
    if (!inView) {
      return;
    }

    if (onLoad) {
      onLoad();
    }
  }, [inView]);

  const handleChange = (_event: React.SyntheticEvent, newExpanded: boolean) => {
    if (loading) {
      return;
    }
    setExpanded(newExpanded);
  };

  const TaskIcon = useMemo(() => {
    return taskIconsMap[task?.type ?? ""] ?? Archive;
  }, [task?.type]);

  return (
    <Accordion expanded={expanded} onChange={handleChange} TransitionProps={{ unmountOnExit: true }}>
      <AccordionSummary aria-controls="panel1d-content">
        <SummaryButton
          disabled={loading}
          size={"large"}
          expanded={expanded}
          fullWidth
          percentage={
            showProgress
              ? ((task?.summary?.props?.download?.downloaded ?? 0) * 100) /
                Math.max(task?.summary?.props?.download?.total ?? 0, 1)
              : undefined
          }
          startIcon={
            loading ? (
              <Skeleton ref={loading ? ref : undefined} variant={"rounded"} width={22} height={22} />
            ) : task?.type === TaskType.remote_download ? (
              <FileIcon
                sx={{
                  p: 0,
                  height: 30,
                }}
                file={{
                  type: (task?.summary?.props.download?.files?.length ?? 0) > 1 ? FileType.folder : FileType.file,
                  name: task?.summary?.props.download?.name ?? "",
                }}
              />
            ) : (
              <TaskIcon />
            )
          }
        >
          <Box
            sx={{
              display: "flex",
              width: "100%",
              textAlign: "left",
              justifyContent: "space-between",
            }}
          >
            <Typography
              variant={"inherit"}
              sx={{
                flexGrow: 1,
                wordBreak: "break-all",
              }}
            >
              {loading || !task ? (
                <Skeleton variant={"text"} width={150} />
              ) : (
                <Box component={"span"} sx={{ verticalAlign: "sub" }}>
                  <TaskSummaryTitle type={task.type} summary={task.summary} />
                </Box>
              )}
            </Typography>

            <Typography color={"text.secondary"} variant={"inherit"} sx={{ display: "flex", alignItems: "center" }}>
              {loading || !task ? (
                <Skeleton variant={"text"} width={50} />
              ) : (
                <TaskSummaryStatus
                  simplified={isMobile}
                  type={task.type}
                  status={task.status}
                  error={task.error}
                  summary={task.summary}
                />
              )}
            </Typography>
          </Box>
        </SummaryButton>
      </AccordionSummary>
      <AccordionDetails>{task && <TaskDetail task={task} downloading={showProgress} />}</AccordionDetails>
    </Accordion>
  );
};

export default TaskCard;
