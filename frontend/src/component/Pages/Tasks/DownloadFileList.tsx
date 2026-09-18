import {
  Box,
  Button,
  Grow,
  Stack,
  Table,
  TableCell,
  TableContainer,
  TableRow,
  TextField,
  Typography,
  useTheme,
} from "@mui/material";
import { useSnackbar } from "notistack";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { TableVirtuoso } from "react-virtuoso";
import { sendCancelDownloadTask, sendSetDownloadTarget } from "../../../api/api.ts";
import { FileType } from "../../../api/explorer.ts";
import { DownloadTaskFile, TaskSummary } from "../../../api/workflow.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { confirmOperation } from "../../../redux/thunks/dialog.ts";
import { fileBase, sizeToString } from "../../../util";
import { StyledCheckbox, StyledTableContainerPaper } from "../../Common/StyledComponents.tsx";
import FileIcon from "../../FileManager/Explorer/FileIcon.tsx";
import Dismiss from "../../Icons/Dismiss.tsx";
import { getProgressColor } from "./TaskCard.tsx";

export interface DownloadFileListProps {
  taskId: string;
  summary?: TaskSummary;
  downloading?: boolean;
  readonly?: boolean;
}

const DownloadFileList = ({ taskId, summary, downloading, readonly }: DownloadFileListProps) => {
  const [selectedMask, setSelectedMask] = useState<{
    [key: number]: boolean;
  }>({});
  const [changeApplied, setChangeApplied] = useState<boolean>(true);
  const [loading, setLoading] = useState<boolean>(false);
  const [height, setHeight] = useState(33);
  const [filterText, setFilterText] = useState("");
  const files = summary?.props?.download?.files;
  const { t } = useTranslation();
  const theme = useTheme();
  const dispatch = useAppDispatch();
  const { enqueueSnackbar } = useSnackbar();

  const progressColor = useMemo(() => {
    return getProgressColor(theme);
  }, [theme]);

  const filteredFiles = useMemo(() => {
    if (!files) {
      return [];
    }
    if (!filterText) {
      return files;
    }
    return files.filter((file) => file.name.toLowerCase().includes(filterText.toLowerCase()));
  }, [files, filterText]);

  const setFileSelected = (value: DownloadTaskFile, selected: boolean) => {
    setSelectedMask((prev) => {
      return {
        ...prev,
        [value.index]: selected,
      };
    });
    setChangeApplied(false);
  };

  const submitChanges = () => {
    setLoading(true);
    dispatch(
      sendSetDownloadTarget(taskId, {
        files: Object.keys(selectedMask).map((key) => ({
          index: parseInt(key),
          download: selectedMask[parseInt(key)],
        })),
      }),
    )
      .then(() => {
        setChangeApplied(true);
        enqueueSnackbar({
          message: t("download.operationSubmitted"),
          variant: "success",
        });
      })
      .finally(() => {
        setLoading(false);
      });
  };

  const selectAll = () => {
    const newMask = { ...selectedMask };
    filteredFiles.forEach((file) => {
      newMask[file.index] = true;
    });
    setSelectedMask(newMask);
    setChangeApplied(false);
  };

  const reverseSelect = () => {
    const newMask = { ...selectedMask };
    filteredFiles.forEach((file) => {
      const currentSelection = newMask[file.index] ?? file.selected;
      newMask[file.index] = !currentSelection;
    });
    setSelectedMask(newMask);
    setChangeApplied(false);
  };

  const cancelTask = () => {
    dispatch(confirmOperation(t("download.cancelTaskConfirm"))).then(() => {
      setLoading(true);
      dispatch(sendCancelDownloadTask(taskId))
        .then(() => {
          enqueueSnackbar({
            message: t("download.taskCanceled"),
            variant: "success",
          });
        })
        .finally(() => {
          setLoading(false);
        });
    });
  };

  return (
    <Box sx={{ py: readonly ? 0 : 2 }}>
      {files && (
        <>
          {!readonly && files.length > 20 && (
            <Stack direction="row" spacing={1} sx={{ mb: 1 }}>
              <TextField
                label={t("download.filterByName")}
                variant="outlined"
                size="small"
                value={filterText}
                onChange={(e) => setFilterText(e.target.value)}
                sx={{ flexGrow: 1 }}
              />
              <Button variant="outlined" onClick={selectAll} disabled={loading || !downloading}>
                {t("download.selectAll")}
              </Button>
              <Button variant="outlined" onClick={reverseSelect} disabled={loading || !downloading}>
                {t("download.reverseSelect")}
              </Button>
            </Stack>
          )}
          <TableContainer component={StyledTableContainerPaper}>
            <TableVirtuoso
              style={{
                height: height,
                overflow: "auto",
                maxHeight: 300,
              }}
              totalListHeightChanged={(h) => {
                setHeight(h + 0.5);
              }}
              components={{
                // eslint-disable-next-line react/display-name
                Table: (props) => <Table {...props} size={"small"} />,
                // eslint-disable-next-line react/display-name
                TableRow: (props) => {
                  const index = props["data-index"];
                  const percentage = (files[index]?.progress ?? 0) * 100;
                  const progressBgColor = theme.palette.background.default;
                  return (
                    <TableRow
                      {...props}
                      key={index}
                      sx={{
                        background: `linear-gradient(to right, ${progressColor} 0%,${progressColor} ${percentage}%,${progressBgColor} ${percentage}%,${progressBgColor} 100%)`,
                      }}
                    />
                  );
                },
              }}
              data={filteredFiles}
              itemContent={(_index, value) => (
                <>
                  <TableCell component="th" scope="row" sx={{ height: 33, minWidth: 50 }}>
                    <StyledCheckbox
                      onChange={(e) => {
                        setFileSelected(value, e.target.checked);
                      }}
                      disabled={!downloading || readonly}
                      disableRipple
                      checked={selectedMask[value.index] ?? value.selected}
                      size="small"
                    />
                  </TableCell>
                  <TableCell component="th" scope="row" sx={{ minWidth: 300, width: "100%" }}>
                    <Typography variant={"body2"} sx={{ display: "flex", alignItems: "center" }}>
                      <FileIcon
                        sx={{ px: 0, py: 0, mr: 1, height: "20px" }}
                        variant={"small"}
                        iconProps={{
                          fontSize: "small",
                        }}
                        file={{
                          type: FileType.file,
                          name: value.name,
                        }}
                      />
                      <Typography variant="inherit">{fileBase(value.name)}</Typography>
                    </Typography>
                  </TableCell>
                  <TableCell component="th" scope="row" sx={{ minWidth: 120 }}>
                    <Typography noWrap variant={"body2"}>
                      {sizeToString(value.size)}
                    </Typography>
                  </TableCell>
                  <TableCell component="th" scope="row" sx={{ minWidth: 105 }}>
                    <Typography noWrap variant={"body2"}>
                      {((value.progress ?? 0) * 100).toFixed(2)} %
                    </Typography>
                  </TableCell>
                </>
              )}
            />
            {filteredFiles.length === 0 && (
              <Typography variant="body2" color="text.secondary" align="center" sx={{ my: 1 }}>
                {t("download.noFilesFound")}
              </Typography>
            )}
          </TableContainer>
        </>
      )}
      {downloading && !readonly && summary?.phase == "monitor" && (
        <Box
          sx={{
            display: "flex",
            mt: 2,
            justifyContent: "space-between",
          }}
        >
          <Box>
            <Grow unmountOnExit in={!changeApplied}>
              <Button disabled={loading} onClick={submitChanges} color={"primary"} variant={"contained"}>
                {t("download.saveChanges")}
              </Button>
            </Grow>
          </Box>
          <Button color={"error"} onClick={cancelTask} disabled={loading} startIcon={<Dismiss />} variant={"contained"}>
            {t("download.cancelTask")}
          </Button>
        </Box>
      )}
    </Box>
  );
};

export default DownloadFileList;
