import { alpha, Box, Divider, Grow, IconButton, ListItemButton, ListItemText, styled, Tooltip } from "@mui/material";
import MuiAccordion from "@mui/material/Accordion";
import MuiAccordionDetails from "@mui/material/AccordionDetails";
import MuiAccordionSummary from "@mui/material/AccordionSummary";
import Chip from "@mui/material/Chip";
import { lighten, useTheme } from "@mui/material/styles";
import useMediaQuery from "@mui/material/useMediaQuery";
import React, { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { FileType } from "../../../api/explorer.ts";
import { navigateToPath } from "../../../redux/thunks/filemanager.ts";
import { sizeToString } from "../../../util";
import { NoWrapBox } from "../../Common/StyledComponents.tsx";
import FileTypeIcon from "../../FileManager/Explorer/FileTypeIcon.tsx";
import ArrowClockwiseFilled from "../../Icons/ArrowClockwiseFilled.tsx";
import Delete from "../../Icons/Delete.tsx";
import DocumentArrowDownFilled from "../../Icons/DocumentArrowDownFilled.tsx";
import Play from "../../Icons/Play.tsx";
import RenameFilled from "../../Icons/RenameFilled.tsx";
import { useUpload } from "../UseUpload.js";
import { SelectType } from "../core";
import { CreateUploadSessionError, UploaderError } from "../core/errors";
import Base, { Status } from "../core/uploader/base";
import TaskDetail from "./TaskDetail.js";

const Accordion = styled(MuiAccordion)(() => ({
  maxWidth: "100%",
  boxShadow: "none",
  "&:not(:last-child)": {
    borderBottom: 0,
  },
  "&:before": {
    display: "none",
  },
  "& .Mui-expanded": {
    margin: "0!important",
  },
}));

const AccordionSummary = styled(MuiAccordionSummary)(() => ({
  minHeight: "56px",
  padding: 0,
  display: "block",
  "& .MuiAccordionSummary-content": {
    margin: 0,
    display: "block",
    "& .Mui-expanded": {
      margin: "0!important",
    },
  },
}));

const AccordionDetails = styled(MuiAccordionDetails)(({ theme }) => ({
  "& .MuiAccordionActions-root": {
    paddingLeft: 16,
    paddingRight: 16,
    paddingTop: 8,
    paddingBottom: 8,
    display: "block",
    backgroundColor: theme.palette.background.default,
  },
}));

const getSpeedText = (speed: number, speedAvg: number, useSpeedAvg: boolean) => {
  let displayedSpeed = speedAvg;
  if (!useSpeedAvg) {
    displayedSpeed = speed;
  }

  return `${sizeToString(displayedSpeed ? displayedSpeed : 0)}/s`;
};

const getErrMsg = (error?: Error) => {
  if (error) {
    let errMsg = error.message;
    if (error instanceof UploaderError) {
      errMsg = error.Message();
    }

    return errMsg;
  }
};

export interface UploadTaskProps {
  uploader: Base;
  useAvgSpeed: boolean;
  onCancel: (filter: (u: Base) => boolean) => void;
  onClose: (u: any, reason: string) => void;
  selectFile: (path: string, type: SelectType, uploader: Base) => void;
  onRefresh: () => void;
}

export default function UploadTask({
  uploader,
  useAvgSpeed,
  onCancel,
  onClose,
  selectFile,
  onRefresh,
}: UploadTaskProps) {
  const { t } = useTranslation("application", { keyPrefix: "uploader" });
  const theme = useTheme();
  const [taskHover, setTaskHover] = useState(false);
  const [expanded, setExpanded] = useState(false);
  const { status, error, progress, speed, speedAvg, retry } = useUpload(uploader);
  const fullScreen = useMediaQuery(theme.breakpoints.down("md"));
  const [loading, setLoading] = useState(false);
  const navigateToDst = (path: string) => {
    onClose(null, "backdropClick");
    navigateToPath(0, path);
  };

  const toggleDetail = (_event: any, newExpanded: boolean) => {
    setExpanded(!!newExpanded);
  };

  useEffect(() => {
    if (status >= Status.finished) {
      onRefresh();
    }
  }, [status]);

  const statusText = useMemo(() => {
    // const parent = filename(uploader.task.dst);
    const parent = "TODO: parent";
    switch (status) {
      case Status.added:
      case Status.initialized:
      case Status.queued:
        return <div>{t("pendingInQueue")}</div>;
      case Status.preparing:
        return <div>{t("preparing")}</div>;
      case Status.error:
        return (
          <Box
            sx={{
              color: theme.palette.warning.main,
              wordBreak: "break-all",
              ...(theme.breakpoints.up("sm")
                ? {
                    whiteSpace: "nowrap",
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                  }
                : {}),
            }}
          >
            {getErrMsg(error)}
            <br />
          </Box>
        );
      case Status.finishing:
        return <div>{t("processing")}</div>;
      case Status.resumable:
        return (
          progress && (
            <div>
              {t("progressDescription", {
                uploaded: sizeToString(progress.total.loaded),
                total: sizeToString(progress.total.size),
                percentage: progress.total.percent.toFixed(2),
              })}
            </div>
          )
        );
      case Status.processing:
        if (progress) {
          return (
            <div>
              {t("progressDescriptionFull", {
                speed: getSpeedText(speed, speedAvg, useAvgSpeed),
                uploaded: sizeToString(progress.total.loaded),
                total: sizeToString(progress.total.size),
                percentage: progress.total.percent.toFixed(2),
              })}
            </div>
          );
        }
        return <div>{t("progressDescriptionPlaceHolder")}</div>;
      case Status.finished:
        return (
          <Box
            sx={{
              color: theme.palette.success.main,
            }}
          >
            {t("uploaded")}
            <br />
          </Box>
        );
      default:
        return <div>{t("unknownStatus")}</div>;
    }
  }, [status, error, progress, speed, speedAvg, useAvgSpeed]);

  const resumeLabel = useMemo(
    () =>
      uploader.task.resumed && !fullScreen ? (
        <Chip
          sx={{
            ml: 1,
            height: 18,
          }}
          size="small"
          label={t("resumed")}
        />
      ) : null,
    [status, fullScreen],
  );

  const continueLabel = useMemo(
    () =>
      status === Status.resumable && !fullScreen ? (
        <Chip
          sx={{
            ml: 1,
            height: 18,
            fontSize: (theme) => theme.typography.caption.fontSize,
          }}
          size="small"
          color={"secondary"}
          label={t("resumable")}
        />
      ) : null,
    [status, fullScreen],
  );

  const progressBar = useMemo(
    () =>
      (status === Status.processing || status === Status.finishing || status === Status.resumable) && progress ? (
        <Box
          sx={{
            backgroundColor:
              theme.palette.mode === "light"
                ? lighten(theme.palette.primary.main, 0.8)
                : alpha(theme.palette.primary.main, 0.18),
            width: progress.total.percent + "%",
            transition: "width .4s linear",
            zIndex: 1,
            height: "100%",
            position: "absolute",
            left: 0,
            top: 0,
          }}
        />
      ) : null,
    [status, progress, theme],
  );

  const taskDetail = useMemo(() => {
    return <TaskDetail error={(error && getErrMsg(error)) ?? ""} uploader={uploader} />;
  }, [uploader, expanded]);

  const cancel = () => {
    setLoading(true);
    uploader.cancel().then(() => {
      setLoading(false);
      onCancel((u) => u.id != uploader.id);
    });
  };

  const stopRipple = (e: React.MouseEvent<any> | React.TouchEvent<any>) => {
    e.stopPropagation();
  };

  const secondaryAction = useMemo(() => {
    if (!taskHover && !fullScreen) {
      return <></>;
    }

    const isConflict = error instanceof CreateUploadSessionError && error.IsConflictError();

    const actions: {
      show: boolean;
      title: string;
      click: () => void;
      icon: JSX.Element;
      loading: boolean;
    }[] = [
      {
        show: status === Status.error && !isConflict,
        title: t("retry"),
        click: () => retry(),
        icon: <ArrowClockwiseFilled sx={{ fontSize: "20px" }} />,
        loading: false,
      },
      {
        show: status === Status.error && isConflict,
        title: t("rename"),
        click: () => retry({ new_prefix: t("fileCopyName") }),
        icon: <RenameFilled sx={{ fontSize: "20px" }} />,
        loading: false,
      },
      {
        show: status === Status.error && isConflict,
        title: t("overwrite"),
        click: () => retry({ overwrite: true }),
        icon: <DocumentArrowDownFilled sx={{ fontSize: "20px" }} />,
        loading: false,
      },
      {
        show: true,
        title: status === Status.finished ? t("deleteTask") : t("cancelAndDelete"),
        click: cancel,
        icon: <Delete sx={{ fontSize: "20px" }} />,
        loading: loading,
      },
      {
        show: status === Status.resumable,
        title: t("selectAndResume"),
        click: () => selectFile(uploader.task.dst, SelectType.File, uploader),
        icon: <Play sx={{ fontSize: "20px" }} />,
        loading: false,
      },
    ];

    return (
      <>
        {actions.map((a) => (
          <>
            {a.show && (
              <Grow in={a.show}>
                <Tooltip title={a.title}>
                  <IconButton
                    onMouseDown={stopRipple}
                    onTouchStart={stopRipple}
                    disabled={a.loading}
                    onClick={(e) => {
                      e.stopPropagation();
                      a.click();
                    }}
                    size="small"
                  >
                    {a.icon}
                  </IconButton>
                </Tooltip>
              </Grow>
            )}
          </>
        ))}
      </>
    );
  }, [status, loading, taskHover, fullScreen, uploader, t]);

  const fileIcon = useMemo(() => {
    if (!fullScreen) {
      return <FileTypeIcon name={uploader.task.name} fileType={FileType.file} />;
    }
  }, [uploader, fullScreen]);

  return (
    <>
      <Accordion disableGutters square expanded={expanded} onChange={toggleDetail}>
        <AccordionSummary aria-controls="panel1d-content" id="panel1d-header">
          <Box
            sx={{
              position: "relative",
            }}
            onMouseLeave={() => setTaskHover(false)}
            onMouseOver={() => setTaskHover(true)}
          >
            {progressBar}
            <ListItemButton
              sx={{
                borderRadius: 0,
                position: "relative",
                zIndex: 9,
              }}
            >
              {fileIcon}
              <ListItemText
                sx={{
                  marginLeft: "16px",
                  marginRight: "20px",
                  my: 0,
                }}
                primary={
                  <Box
                    sx={{
                      display: "flex",
                      alignItems: "center",
                    }}
                  >
                    <NoWrapBox
                      sx={{
                        overflow: "hidden",
                      }}
                    >
                      {uploader.task.name}
                    </NoWrapBox>
                    <div>{resumeLabel}</div>
                    <div>{continueLabel}</div>
                  </Box>
                }
                secondary={
                  <NoWrapBox
                    sx={{
                      overflow: "hidden",
                    }}
                  >
                    {statusText}
                  </NoWrapBox>
                }
                slotProps={{
                  primary: {
                    variant: "body2",
                  },

                  secondary: {
                    variant: "caption",
                  },
                }}
              />
              <Box sx={{ display: "flex", gap: 1 }}>{secondaryAction}</Box>
            </ListItemButton>
          </Box>
        </AccordionSummary>
        <AccordionDetails>{taskDetail}</AccordionDetails>
      </Accordion>
      <Divider />
    </>
  );
}
