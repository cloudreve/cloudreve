import { Box, Breadcrumbs, Button, Link, Table, TableCell, TableContainer, Typography, useTheme } from "@mui/material";
import { useSnackbar } from "notistack";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { TableVirtuoso } from "react-virtuoso";
import { getArchiveListFiles } from "../../../api/api.ts";
import { ArchivedFile, FileType } from "../../../api/explorer.ts";
import { closeArchiveViewer, setExtractArchiveDialog } from "../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { fileBase, fileExtension, getFileLinkedUri, sizeToString } from "../../../util";
import AutoHeight from "../../Common/AutoHeight.tsx";
import EncodingSelector, { defaultEncodingValue } from "../../Common/Form/EncodingSelector.tsx";
import { SecondaryButton, StyledCheckbox, StyledTableContainerPaper } from "../../Common/StyledComponents.tsx";
import TimeBadge from "../../Common/TimeBadge.tsx";
import FileIcon from "../../FileManager/Explorer/FileIcon.tsx";
import ChevronRight from "../../Icons/ChevronRight.tsx";
import Folder from "../../Icons/Folder.tsx";
import Home from "../../Icons/Home.tsx";
import ViewerDialog, { ViewerLoading } from "../ViewerDialog.tsx";

const ArchivePreview = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const theme = useTheme();
  const { enqueueSnackbar } = useSnackbar();
  const viewerState = useAppSelector((state) => state.globalState.archiveViewer);

  const [loading, setLoading] = useState(false);
  const [files, setFiles] = useState<ArchivedFile[]>([]);
  const [currentPath, setCurrentPath] = useState<string>("");
  const [selectedFiles, setSelectedFiles] = useState<string[]>([]);
  const [filterText, setFilterText] = useState("");
  const [height, setHeight] = useState(33);
  const [encoding, setEncoding] = useState(defaultEncodingValue);

  const isZip = useMemo(() => {
    return fileExtension(viewerState?.file?.name ?? "") === "zip";
  }, [viewerState?.file?.name]);

  const currentFiles = useMemo(() => {
    if (!files) return [];

    if (!currentPath) {
      return files.filter((file) => !file.name.includes("/"));
    }

    // 如果在子目录，显示该目录下的文件和文件夹
    const pathPrefix = currentPath.endsWith("/") ? currentPath : currentPath + "/";
    const pathFiles = files.filter((file) => file.name.startsWith(pathPrefix) && file.name !== currentPath);

    // 去重并转换为相对路径
    const relativePaths = new Set<string>();
    const result: ArchivedFile[] = [];

    pathFiles.forEach((file) => {
      const relativePath = file.name.substring(pathPrefix.length);
      const firstSlash = relativePath.indexOf("/");

      if (firstSlash === -1) {
        if (!relativePaths.has(relativePath)) {
          relativePaths.add(relativePath);
          result.push({
            ...file,
            name: relativePath,
          });
        }
      } else {
        const dirName = relativePath.substring(0, firstSlash);
        if (!relativePaths.has(dirName)) {
          relativePaths.add(dirName);
          result.push({
            name: dirName,
            size: 0,
            updated_at: file.updated_at,
            is_directory: true,
          });
        }
      }
    });

    return result;
  }, [files, currentPath]);

  // 过滤文件
  const filteredFiles = useMemo(() => {
    if (!filterText) return currentFiles;
    return currentFiles.filter((file) => file.name.toLowerCase().includes(filterText.toLowerCase()));
  }, [currentFiles, filterText]);

  // 面包屑路径
  const breadcrumbPaths = useMemo(() => {
    if (!currentPath) return [];
    return currentPath.split("/").filter(Boolean);
  }, [currentPath]);

  // 规范化路径，去除开头可能存在的 `/`
  const normalizeName = (name: string) => {
    if (name && typeof name === "string" && name.startsWith("/")) {
      return name.slice(1);
    }
    return name;
  };

  useEffect(() => {
    if (!viewerState || !viewerState.open) {
      setEncoding(defaultEncodingValue);
      return;
    }

    setLoading(true);
    setFiles([]);
    setCurrentPath("");
    setSelectedFiles([]);
    setFilterText("");

    dispatch(
      getArchiveListFiles({
        uri: getFileLinkedUri(viewerState.file),
        entity: viewerState.version,
        text_encoding: encoding !== defaultEncodingValue ? encoding : undefined,
      }),
    )
      .then((res) => {
        if (res.files) {
          // 补齐目录
          const allItems: ArchivedFile[] = [];
          const allDirs = new Set<string>();

          // 目录项
          res.files
            .filter((item) => item.is_directory)
            .forEach((item) => {
              const normalizedName = normalizeName(item.name);
              allItems.push({
                ...item,
                name: normalizedName,
              });
              allDirs.add(normalizedName);
            });

          // 文件项，并补齐缺失目录
          res.files
            .filter((item) => !item.is_directory)
            .forEach((item) => {
              const normalizedName = normalizeName(item.name);
              allItems.push({
                ...item,
                name: normalizedName,
              });

              const dirElements = normalizedName.split("/");
              for (let i = 1; i < dirElements.length; i++) {
                const dirName = dirElements.slice(0, i).join("/");
                if (!allDirs.has(dirName)) {
                  allDirs.add(dirName);
                  allItems.push({
                    name: dirName,
                    size: 0,
                    updated_at: "1970-01-01T00:00:00Z",
                    is_directory: true,
                  });
                }
              }
            });

          // 排序文件
          // 先目录，后文件，分别按名称排序
          allItems.sort((a, b) => {
            if (a.is_directory && !b.is_directory) return -1;
            if (!a.is_directory && b.is_directory) return 1;
            return a.name.localeCompare(b.name);
          });

          setFiles(allItems);
        }
      })
      .catch(() => {
        onClose();
      })
      .finally(() => {
        setLoading(false);
      });
  }, [viewerState, encoding]);

  const onClose = useCallback(() => {
    dispatch(closeArchiveViewer());
  }, [dispatch]);

  const navigateToDirectory = useCallback(
    (dirName: string) => {
      if (!currentPath) {
        setCurrentPath(dirName);
      } else {
        setCurrentPath(currentPath + "/" + dirName);
      }
      setSelectedFiles([]);
    },
    [currentPath],
  );

  const navigateToBreadcrumb = useCallback(
    (index: number) => {
      if (index === -1) {
        setCurrentPath("");
      } else {
        const newPath = breadcrumbPaths.slice(0, index + 1).join("/");
        setCurrentPath(newPath);
      }
      setSelectedFiles([]);
    },
    [breadcrumbPaths],
  );

  const toggleFileSelection = useCallback(
    (fileName: string) => {
      const fullPath = currentPath ? currentPath + "/" + fileName : fileName;
      setSelectedFiles((prev) => {
        if (prev.includes(fullPath)) {
          return prev.filter((f) => f !== fullPath);
        } else {
          return [...prev, fullPath];
        }
      });
    },
    [currentPath],
  );

  const toggleSelectAll = useCallback(() => {
    const allFiles = filteredFiles.map((file) => (currentPath ? currentPath + "/" + file.name : file.name));

    const allSelected = allFiles.every((file) => selectedFiles.includes(file));

    if (allSelected) {
      setSelectedFiles((prev) => prev.filter((file) => !allFiles.includes(file)));
    } else {
      setSelectedFiles((prev) => [...new Set([...prev, ...allFiles])]);
    }
  }, [filteredFiles, selectedFiles, currentPath]);

  // 解压选中的文件
  const extractSelectedFiles = useCallback(() => {
    if (selectedFiles.length === 0) {
      return;
    }

    dispatch(setExtractArchiveDialog({ open: true, file: viewerState?.file, mask: selectedFiles, encoding }));
  }, [selectedFiles, t, enqueueSnackbar, encoding]);

  const extractArchive = useCallback(() => {
    if (!viewerState?.file) {
      return;
    }
    dispatch(setExtractArchiveDialog({ open: true, file: viewerState?.file, encoding }));
  }, [viewerState?.file, encoding]);

  return (
    <>
      <ViewerDialog
        file={viewerState?.file}
        loading={loading}
        dialogProps={{
          open: !!(viewerState && viewerState.open),
          onClose: onClose,
          fullWidth: true,
          maxWidth: "lg",
        }}
      >
        <AutoHeight>
          <div>
            {loading && <ViewerLoading />}
            {!loading && (
              <Box sx={{ p: 2 }}>
                <Box sx={{ mb: 2 }}>
                  <Breadcrumbs separator={<ChevronRight fontSize="small" />}>
                    <Link
                      component="button"
                      variant="body2"
                      color="inherit"
                      onClick={() => navigateToBreadcrumb(-1)}
                      sx={{
                        display: "flex",
                        alignItems: "center",
                        textDecoration: "none",
                        "&:hover": { textDecoration: "underline" },
                      }}
                    >
                      <Home fontSize="small" sx={{ mr: 0.5 }} />
                      {t("fileManager.rootFolder")}
                    </Link>
                    {breadcrumbPaths.map((path, index) => {
                      const isLast = index === breadcrumbPaths.length - 1;
                      return isLast ? (
                        <Typography
                          variant="body2"
                          key={index}
                          color="text.primary"
                          sx={{ display: "flex", alignItems: "center" }}
                        >
                          <Folder fontSize="small" sx={{ mr: 0.5 }} />
                          {path}
                        </Typography>
                      ) : (
                        <Link
                          key={index}
                          component="button"
                          variant="body2"
                          color="inherit"
                          onClick={() => navigateToBreadcrumb(index)}
                          sx={{
                            display: "flex",
                            alignItems: "center",
                            textDecoration: "none",
                            "&:hover": { textDecoration: "underline" },
                          }}
                        >
                          <Folder fontSize="small" sx={{ mr: 0.5 }} />
                          {path}
                        </Link>
                      );
                    })}
                  </Breadcrumbs>
                </Box>

                {filteredFiles.length > 0 ? (
                  <TableContainer component={StyledTableContainerPaper}>
                    <TableVirtuoso
                      style={{
                        height: Math.min(height, 400),
                        overflow: "auto",
                      }}
                      totalListHeightChanged={(h) => {
                        setHeight(h + 0.5);
                      }}
                      components={{
                        // eslint-disable-next-line react/display-name
                        Table: (props) => <Table {...props} size="small" />,
                      }}
                      data={filteredFiles}
                      itemContent={(_index, file) => {
                        const fullPath = currentPath ? currentPath + "/" + file.name : file.name;
                        const isSelected = selectedFiles.includes(fullPath);

                        return (
                          <>
                            <TableCell sx={{ width: 50, padding: "4px 8px" }}>
                              <StyledCheckbox
                                checked={isSelected}
                                onChange={() => toggleFileSelection(file.name)}
                                size="small"
                              />
                            </TableCell>
                            <TableCell
                              sx={{
                                minWidth: 300,
                                width: "100%",
                                padding: "4px 8px",
                              }}
                            >
                              <Typography variant="body2" sx={{ display: "flex", alignItems: "center" }}>
                                <FileIcon
                                  sx={{ px: 0, py: 0, mr: 1, height: "20px" }}
                                  variant="small"
                                  iconProps={{ fontSize: "small" }}
                                  file={{
                                    type: file.is_directory ? FileType.folder : FileType.file,
                                    name: file.name,
                                  }}
                                />
                                {file.is_directory ? (
                                  <Typography
                                    component="button"
                                    variant="inherit"
                                    onClick={() => navigateToDirectory(file.name)}
                                    sx={{
                                      color: "primary.main",
                                      fontWeight: 500,
                                      textDecoration: "none",
                                      background: "none",
                                      border: "none",
                                      cursor: "pointer",
                                      padding: 0,
                                      "&:hover": { textDecoration: "underline" },
                                    }}
                                  >
                                    {fileBase(file.name)}
                                  </Typography>
                                ) : (
                                  <Typography
                                    variant="inherit"
                                    sx={{
                                      color: "inherit",
                                      fontWeight: 400,
                                    }}
                                  >
                                    {fileBase(file.name)}
                                  </Typography>
                                )}
                              </Typography>
                            </TableCell>
                            <TableCell sx={{ minWidth: 100, padding: "4px 8px" }}>
                              <Typography variant="body2" noWrap>
                                {file.is_directory ? "-" : sizeToString(file.size)}
                              </Typography>
                            </TableCell>
                            <TableCell sx={{ minWidth: 120, padding: "4px 8px" }}>
                              <Typography variant="body2" noWrap>
                                {file.updated_at ? <TimeBadge variant="inherit" datetime={file.updated_at} /> : "-"}
                              </Typography>
                            </TableCell>
                          </>
                        );
                      }}
                    />
                  </TableContainer>
                ) : (
                  <Typography variant="body2" color="text.secondary" align="center" sx={{ py: 4 }}>
                    {t("fileManager.nothingFound")}
                  </Typography>
                )}

                {!viewerState?.version && (
                  <Box
                    sx={{
                      mt: 2,
                      display: "flex",
                      flexWrap: "wrap",
                      gap: 1,
                      justifyContent: "space-between",
                      alignItems: "flex-end",
                    }}
                  >
                    <Box sx={{ display: "flex", flexWrap: "wrap", gap: 1 }}>
                      <Button variant="contained" onClick={extractArchive} color="primary">
                        {t("fileManager.extractArchive")}
                      </Button>
                      {selectedFiles.length > 0 && (
                        <SecondaryButton variant={"contained"} onClick={extractSelectedFiles}>
                          {t("fileManager.extractSelected")}
                        </SecondaryButton>
                      )}
                    </Box>
                    {isZip && (
                      <Box>
                        <EncodingSelector
                          value={encoding}
                          onChange={setEncoding}
                          size="small"
                          variant="filled"
                          fullWidth
                          showIcon
                          label={t("modals.selectEncoding")}
                        />
                      </Box>
                    )}
                  </Box>
                )}
              </Box>
            )}
          </div>
        </AutoHeight>
      </ViewerDialog>
    </>
  );
};

export default ArchivePreview;
