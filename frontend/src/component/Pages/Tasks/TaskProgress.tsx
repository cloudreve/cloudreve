import { Box, Stepper, useMediaQuery, useTheme } from "@mui/material";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { NodeTypes, TaskStatus, TaskSummary, TaskType } from "../../../api/workflow.ts";
import PieceProgress from "./PieceProgress.tsx";
import TaskProgressStep from "./TaskProgressStep.tsx";

export interface TaskProgressProps {
  taskId: string;
  taskStatus: string;
  taskType: string;
  summary?: TaskSummary;
  node?: {
    type: string;
  };
}

interface StepModel {
  title: string;
  state: string;
  description?: string;
  supportProgress?: boolean;
}

const queueingStep: StepModel = {
  title: "setting.queueToStart",
  state: "",
};

const completedStep: StepModel = {
  title: "setting.finished",
  state: "",
};

const stepOptions: {
  [key: string]: StepModel[][];
} = {
  [TaskType.remote_download]: [
    // Master
    [
      queueingStep,
      {
        title: "fileManager.download",
        state: "monitor",
        description: "setting.downloadDes",
      },
      {
        title: "setting.transferring",
        state: "transfer",
        description: "setting.downloadTransferDes",
        supportProgress: true,
      },
      {
        title: "setting.awaitSeeding",
        state: "seeding",
        description: "setting.awaitSeedingDes",
      },
      completedStep,
    ],
    // Slave
    [
      queueingStep,
      {
        title: "fileManager.download",
        state: "monitor",
        description: "setting.downloadDes",
      },
      {
        title: "setting.transferring",
        state: "transfer",
        description: "setting.downloadTransferDes",
        supportProgress: true,
      },
      {
        title: "setting.awaitSeeding",
        state: "seeding",
        description: "setting.awaitSeedingDes",
      },
      completedStep,
    ],
  ],
  [TaskType.import]: [
    // Master
    [
      queueingStep,
      {
        title: "setting.importingFiles",
        state: "",
        description: "setting.importingFilesDes",
        supportProgress: true,
      },
    ],
  ],
  [TaskType.relocate]: [
    // Master
    [
      queueingStep,
      {
        title: "setting.indexingFiles",
        state: "",
        description: "setting.indexingFilesDes",
      },
      {
        title: "setting.transferring",
        state: "transfer",
        description: "setting.transferringRelocateDes",
        supportProgress: true,
      },
      {
        title: "setting.committingChanges",
        state: "finish",
        description: "setting.relocateFinishing",
      },
    ],
  ],
  [TaskType.extract_archive]: [
    // Master
    [
      queueingStep,
      {
        title: "setting.downloadingZip",
        description: "setting.downloadingZipDes",
        state: "download_zip",
        supportProgress: true,
      },
      {
        title: "setting.extractingFiles",
        state: "",
        description: "setting.extractingFilesDes",
        supportProgress: true,
      },
    ],
    // Slave
    [
      queueingStep,
      {
        title: "setting.sendTask",
        description: "setting.sendTaskDes",
        state: "",
      },
      {
        title: "setting.extractingFiles",
        state: "await_slave_complete",
        description: "setting.extractingFilesDes",
        supportProgress: true,
      },
    ],
  ],
  [TaskType.full_text_rebuild]: [
    // Master
    [
      queueingStep,
      {
        title: "setting.ftsNukingIndex",
        state: "nuke",
        description: "setting.ftsNukingIndexDes",
      },
      {
        title: "setting.ftsIndexingFiles",
        state: "index",
        description: "setting.ftsIndexingFilesDes",
        supportProgress: true,
      },
      completedStep,
    ],
  ],
  [TaskType.create_archive]: [
    // Master
    [
      queueingStep,
      {
        title: "setting.prepare",
        state: "",
        description: "setting.preparingWorkspaceDes",
      },
      {
        title: "setting.compressFiles",
        state: "compress_files",
        description: "setting.compressFilesDes",
        supportProgress: true,
      },
      {
        title: "setting.transferring",
        state: "upload_archive",
        description: "setting.uploadArchiveFileDes",
        supportProgress: true,
      },
    ],
    // Slave
    [
      queueingStep,
      {
        title: "setting.indexingFiles",
        state: "",
        description: "setting.indexForArchiveDes",
      },
      {
        title: "setting.compressFiles",
        state: "await_slave_compressing",
        description: "setting.compressFilesDes",
        supportProgress: true,
      },
      {
        title: "setting.transferring",
        state: "await_slave_uploading",
        description: "setting.uploadArchiveFileDes",
        supportProgress: true,
      },
      {
        title: "setting.committingChanges",
        state: "complete_upload",
        description: "setting.createArchiveFinishing",
      },
    ],
  ],
};

const TaskProgress = ({ taskId, taskStatus, taskType, summary, node }: TaskProgressProps) => {
  const { t } = useTranslation();
  const [activeStep, setActiveStep] = useState(0);
  const steps = useMemo((): StepModel[] => {
    return stepOptions[taskType]?.[node?.type == NodeTypes.slave ? 1 : 0] ?? [];
  }, [taskId, node?.type]);
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("md"));

  useEffect(() => {
    if (taskStatus == TaskStatus.queued) {
      setActiveStep(0);
      return;
    }
    if (taskStatus == TaskStatus.completed) {
      setActiveStep(steps.length);
      return;
    }
    let active = 1;
    for (let i = 1; i < steps.length; i++) {
      if (steps[i].state == summary?.phase) {
        active = i;
      }
    }

    setActiveStep(active);
  }, [steps, taskStatus, summary?.phase]);

  return (
    <Box sx={{ p: 2 }}>
      <Stepper activeStep={activeStep} orientation={isMobile ? "vertical" : "horizontal"}>
        {steps.map((step, index) => (
          <TaskProgressStep
            progressing={activeStep == index && taskStatus != TaskStatus.error}
            taskId={taskId}
            taskStatus={taskStatus}
            description={step.description}
            title={step.title}
            showProgress={step.supportProgress}
            key={taskId + "_" + step.title}
          />
        ))}
      </Stepper>
      {taskType == TaskType.remote_download && summary?.props.download?.pieces && summary?.phase == "monitor" && (
        <PieceProgress total={summary?.props.download.num_pieces ?? 1} pieces={summary?.props.download?.pieces} />
      )}
    </Box>
  );
};

export default TaskProgress;
