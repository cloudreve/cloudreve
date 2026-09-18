import { Box, Chip, styled, Typography } from "@mui/material";
import { useMemo } from "react";
import { Trans, useTranslation } from "react-i18next";
import { FileType } from "../../../api/explorer.ts";
import { TaskSummary, TaskType } from "../../../api/workflow.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { newMyUri } from "../../../util/uri.ts";
import FileBadge from "../../FileManager/FileBadge.tsx";

export interface TaskSummaryTitleProps {
  type: string;
  summary?: TaskSummary;
  isInDashboard?: boolean;
}

const StyledFileBadge = styled(FileBadge)(() => ({
  paddingLeft: 8,
  paddingRight: 8,
  marginLeft: 4,
  marginRight: 4,
  maxWidth: "200px",
}));

const StyledChip = styled(Chip)(() => ({
  marginLeft: 8,
  height: "20px",
}));

const TaskSummaryTitle = ({ type, summary, isInDashboard = false }: TaskSummaryTitleProps) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const policyOption = useAppSelector((state) => state.globalState.policyOptionCache);

  const selectedCount = useMemo(() => {
    let selected = 0;
    for (const file of summary?.props.download?.files ?? []) {
      if (file.selected) {
        selected++;
      }
    }

    return selected;
  }, [summary?.props.download?.files]);

  switch (type) {
    case TaskType.remote_download:
      return (
        <Box
          sx={{
            display: "flex",
            alignItems: "center",
            height: "100%",
          }}
        >
          <Typography variant={"inherit"} sx={{}}>
            {isInDashboard && t("dashboard:task.remoteDownload")}
            {summary?.props.download?.name ?? t("download.unknownTaskName")}
            {selectedCount > 1 && <StyledChip color={"primary"} size="small" label={selectedCount} />}
          </Typography>
        </Box>
      );
    case TaskType.create_archive:
      return (
        <Trans
          i18nKey="setting.createArchiveTo"
          components={[
            <span key={0}>
              {summary?.props.src_multiple?.slice(0, 3).map((src) => (
                <StyledFileBadge
                  variant={"outlined"}
                  simplifiedFile={{
                    type: FileType.file,
                    path: src,
                  }}
                />
              ))}
            </span>,
            <StyledFileBadge
              variant={"outlined"}
              simplifiedFile={{
                type: FileType.file,
                path: summary?.props.dst ? summary?.props.dst : newMyUri("").toString(),
              }}
            />,
          ]}
          values={{
            more: (summary?.props.src_multiple?.length ?? 0) > 3 ? "..." : "",
          }}
        />
      );
    case TaskType.import:
      return (
        <Trans
          i18nKey="setting.importFileTo"
          values={{
            policy: policyOption
              ? policyOption.find((p) => p.id == summary?.props.dst_policy_id)?.name ?? "Unknown"
              : "",
          }}
          components={[
            <StyledFileBadge
              variant={"outlined"}
              simplifiedFile={{
                type: FileType.folder,
                path: summary?.props.dst ? summary?.props.dst : newMyUri("").toString(),
              }}
            />,
          ]}
        />
      );
    case TaskType.full_text_rebuild:
      return (
        <Typography variant={"inherit"}>
          {t("setting.rebuildFTSIndex", {
            total: summary?.props.total ?? "-",
          })}
        </Typography>
      );
    default:
      return (
        <Trans
          i18nKey="setting.extractFileTo"
          components={[
            <StyledFileBadge
              variant={"outlined"}
              simplifiedFile={{
                type: FileType.file,
                path: summary?.props.src ? summary?.props.src : newMyUri("").toString(),
              }}
            />,
            <StyledFileBadge
              variant={"outlined"}
              simplifiedFile={{
                type: FileType.folder,
                path: summary?.props.dst ? summary?.props.dst : newMyUri("").toString(),
              }}
            />,
          ]}
          values={{
            more: (summary?.props.src_multiple?.length ?? 0) > 3 ? "..." : "",
          }}
        />
      );
  }
};

export default TaskSummaryTitle;
