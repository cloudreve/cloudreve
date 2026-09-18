import React, { useMemo, useState } from "react";
import {
  Accordion,
  AccordionDetails,
  alpha,
  AppBar,
  Box,
  Dialog,
  DialogContent,
  Fade,
  IconButton,
  Slide,
  SlideProps,
  styled,
  Toolbar,
  Tooltip,
  Typography,
} from "@mui/material";
import { lighten, useTheme } from "@mui/material/styles";
import useMediaQuery from "@mui/material/useMediaQuery";
import UploadTask from "./UploadTask.tsx";
import MoreActions from "./MoreActions.js";
import { Virtuoso } from "react-virtuoso";
import Base, { Status } from "../core/uploader/base";
import { useTranslation } from "react-i18next";
import { useAppSelector } from "../../../redux/hooks.ts";
import { defaultPath } from "../../../hooks/useNavigation.tsx";
import SessionManager, { UserSettings } from "../../../session";
import Nothing from "../../Common/Nothing.tsx";
import Dismiss from "../../Icons/Dismiss.tsx";
import MoreHorizontal from "../../Icons/MoreHorizontal.tsx";
import Add from "../../Icons/Add.tsx";
import { ExpandMoreRounded } from "@mui/icons-material";
import { UploadProgressTotal } from "../../../redux/globalStateSlice.ts";

const Transition = React.forwardRef(function Transition(props: SlideProps, ref) {
  return <Slide direction="up" ref={ref} {...props} />;
});

const StyledDialog = styled(Dialog)(({ fullScreen, theme }) => ({
  "& .MuiDialog-container": {
    alignItems: "flex-end",
    justifyContent: "flex-end",
  },
  "& .MuiDialog-paper": {
    border: `1px solid ${theme.palette.divider}`,
  },
  ...(fullScreen
    ? {
        margin: 0,
        position: "fixed",
        inset: "-1!important",
      }
    : { top: "auto!important", left: "auto!important" }),
}));

const StyledDialogContent = styled(DialogContent)(({ theme }) => ({
  [theme.breakpoints.up("md")]: {
    width: 500,
    minHeight: 300,
    maxHeight: "calc(100vh - 140px)",
  },
  padding: 0,
  paddingTop: "0!important",
}));

const CaretDownIcon = styled(ExpandMoreRounded)<{ expanded: boolean }>(({ theme, expanded }) => ({
  transform: `rotate(${expanded ? 0 : 180}deg)`,
  transition: theme.transitions.create("transform", {
    duration: theme.transitions.duration.shortest,
    easing: theme.transitions.easing.easeInOut,
  }),
}));

const sorters: {
  [key: string]: (a: Base, b: Base) => number;
} = {
  default: (a, b) => a.id - b.id,
  reverse: (a, b) => b.id - a.id,
};

const filters: {
  [key: string]: (u: Base) => boolean;
} = {
  default: (_u) => true,
  ongoing: (u) => u.status < Status.finished,
};

export interface TaskListProps {
  open: boolean;
  onClose: () => void;
  selectFile: (path: string) => void;
  taskList: Base[];
  onCancel: (filter: (u: any) => boolean) => void;
  uploadManager: any;
  progress?: UploadProgressTotal;
  setUploaders: (f: (u: Base[]) => Base[]) => void;
}

export default function TaskList({
  open,
  onClose,
  selectFile,
  taskList,
  onCancel,
  uploadManager,
  progress,
  setUploaders,
}: TaskListProps) {
  const { t } = useTranslation("application", { keyPrefix: "uploader" });
  const theme = useTheme();
  const fullScreen = useMediaQuery(theme.breakpoints.down("md"));
  const path = useAppSelector((state) => state.fileManager[0].pure_path);
  const [expanded, setExpanded] = useState(true);
  const [useAvgSpeed, setUseAvgSpeed] = useState<boolean>(SessionManager.getWithFallback(UserSettings.UseAvgSpeed));
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const [filter, setFilter] = useState(SessionManager.getWithFallback(UserSettings.TaskFilter));
  const [sorter, setSorter] = useState(SessionManager.getWithFallback(UserSettings.TaskSorter));
  const [refreshList, setRefreshList] = useState(false);

  const handleActionClick = (event: React.MouseEvent<HTMLElement>) => {
    setAnchorEl(event.currentTarget);
  };

  const handleActionClose = () => {
    setAnchorEl(null);
  };

  const close = (_e: any, reason: string) => {
    if (reason !== "backdropClick") {
      onClose();
    } else {
      setExpanded(false);
    }
  };
  const handlePanelChange = (_event: any, isExpanded: boolean) => {
    setExpanded(isExpanded);
  };

  useMemo(() => {
    if (open) {
      setExpanded(true);
    }
  }, [taskList]);

  const stopPop = (func: (e: React.MouseEvent<HTMLElement>) => void) => (e: React.MouseEvent<HTMLElement>) => {
    e.stopPropagation();
    func(e);
  };

  const progressBar = useMemo(
    () =>
      progress && progress.totalSize > 0 ? (
        <Fade in={progress.totalSize > 0 && !expanded}>
          <div>
            <Box
              sx={{
                backgroundColor:
                  theme.palette.mode === "light"
                    ? lighten(theme.palette.primary.main, 0.8)
                    : alpha(theme.palette.primary.main, 0.18),
                width: (progress.processedSize / progress.totalSize) * 100 + "%",
                transition: "width .4s linear",
                zIndex: -1,
                height: "100%",
                position: "absolute",
                left: 0,
                top: 0,
              }}
            />
          </div>
        </Fade>
      ) : null,
    [progress, expanded, theme],
  );

  const list = useMemo(() => {
    const currentList = taskList.filter(filters[filter]).sort(sorters[sorter]);
    if (currentList.length === 0) {
      return <Nothing size={0.5} top={63} primary={t("taskListEmpty")} />;
    }

    return (
      <Virtuoso
        style={{
          height: (fullScreen ? 500 : 66) * currentList.length,
          maxHeight: "calc(100vh - 56px)",
          ...(theme.breakpoints.up("md")
            ? {
                minHeight: 300,
                maxHeight: "calc(100vh - 140px)",
              }
            : {}),
        }}
        increaseViewportBy={180}
        data={currentList}
        itemContent={(_index, uploader) => (
          <UploadTask
            selectFile={selectFile}
            onClose={close}
            onCancel={onCancel}
            key={uploader.id}
            useAvgSpeed={useAvgSpeed}
            uploader={uploader}
            onRefresh={() => setRefreshList((r) => !r)}
          />
        )}
      />
    );
  }, [taskList, useAvgSpeed, fullScreen, filter, sorter, refreshList]);

  const retryFailed = () => {
    taskList.forEach((task) => {
      if (task.status === Status.error) {
        task.retry();
      }
    });
  };

  return (
    <>
      <MoreActions
        onClose={handleActionClose}
        uploadManager={uploadManager}
        anchorEl={anchorEl}
        useAvgSpeed={useAvgSpeed}
        setUseAvgSpeed={(v) => {
          SessionManager.set(UserSettings.UseAvgSpeed, v);
          setUseAvgSpeed(v);
        }}
        filter={filter}
        sorter={sorter}
        setFilter={(v) => {
          SessionManager.set(UserSettings.TaskFilter, v);
          setFilter(v);
        }}
        setSorter={(v) => {
          SessionManager.set(UserSettings.TaskSorter, v);
          setSorter(v);
        }}
        retryFailed={retryFailed}
        cleanFinished={() => setUploaders((u) => u.filter(filters["ongoing"]))}
      />
      <StyledDialog
        fullScreen={fullScreen}
        open={open}
        onClose={close}
        TransitionComponent={Transition}
        disableEnforceFocus={!expanded}
        hideBackdrop={!expanded}
        disableScrollLock={!expanded}
        //disable elevation
      >
        <Accordion
          sx={
            fullScreen
              ? {
                  boxShadow: "none",
                }
              : undefined
          }
          expanded={expanded || fullScreen}
          onChange={handlePanelChange}
        >
          <AppBar
            color={"default"}
            onClick={() => setExpanded(!expanded)}
            sx={{
              boxShadow: "none",
              position: "relative",
              cursor: "pointer",
            }}
          >
            {progressBar}
            <Toolbar disableGutters sx={{ px: 1, minHeight: "52px!important" }}>
              <Tooltip title={t("hideTaskList")}>
                <IconButton color="inherit" onClick={stopPop(() => close(null, ""))} aria-label="Close">
                  <Dismiss fontSize={"small"} />
                </IconButton>
              </Tooltip>
              <Typography
                variant="subtitle1"
                color="inherit"
                sx={{
                  flex: "1",
                  ml: 1,
                  fontWeight: theme.typography.h6.fontWeight,
                }}
              >
                {t("uploadTasks")}
              </Typography>
              <Box sx={{ display: "flex", gap: 1 }}>
                <Tooltip title={t("moreActions")}>
                  <IconButton color="inherit" onClick={stopPop(handleActionClick)}>
                    <MoreHorizontal fontSize={"small"} />
                  </IconButton>
                </Tooltip>
                {path && (
                  <Tooltip title={t("addNewFiles")}>
                    <IconButton color="inherit" onClick={stopPop(() => selectFile(path ?? defaultPath))}>
                      <Add fontSize={"small"} />
                    </IconButton>
                  </Tooltip>
                )}
                {!fullScreen && (
                  <Tooltip title={t("toggleTaskList")}>
                    <IconButton sx={{ p: "6px" }} color="inherit" onClick={stopPop(() => setExpanded(!expanded))}>
                      <CaretDownIcon expanded={expanded} />
                    </IconButton>
                  </Tooltip>
                )}
              </Box>
            </Toolbar>
          </AppBar>
          <AccordionDetails sx={{ p: 0 }}>
            <StyledDialogContent>{list}</StyledDialogContent>
          </AccordionDetails>
        </Accordion>
      </StyledDialog>
    </>
  );
}
