import dayjs from "dayjs";
import i18next from "i18next";
import { closeSnackbar, enqueueSnackbar } from "notistack";
import streamSaver from "streamsaver";
import { getFileEntityUrl } from "../../api/api.ts";
import { FileResponse, FileType, Metadata } from "../../api/explorer.ts";
import { GroupPermission } from "../../api/user.ts";
import {
  DefaultCloseAction,
  BatchDownloadProgressAction,
  BatchDownloadSecondaryAction,
  BatchDownloadCompleteAction,
} from "../../component/Common/Snackbar/snackbar.tsx";
import SessionManager from "../../session";
import { getFileLinkedUri } from "../../util";
import Boolset from "../../util/boolset.ts";
import { formatLocalTime } from "../../util/datetime.ts";
import {
  getFileSystemDirectoryPaths,
  streamFileToFileSystemDirectory,
  verifyFileSystemRWPermission,
} from "../../util/filesystem.ts";
import "../../util/zip.js";
import { closeContextMenu } from "../fileManagerSlice.ts";
import { DialogSelectOption, setBatchDownloadLog, setBatchDownloadProgress } from "../globalStateSlice.ts";
import { AppThunk } from "../store.ts";
import { promiseId, selectOption } from "./dialog.ts";
import { longRunningTaskWithSnackbar, refreshSingleFileSymbolicLinks, walk, walkAll } from "./file.ts";

enum MultipleDownloadOption {
  Browser,
  StreamSaver,
  Backend,
}

enum DownloadOverwriteOption {
  Skip,
  Overwrite,
  OverwriteAll,
  SkipAll,
}

export function downloadFiles(index: number, files: FileResponse[]): AppThunk {
  return async (dispatch, _getState) => {
    dispatch(closeContextMenu({ index, value: undefined }));
    if (files.length == 1 && files[0].type == FileType.file) {
      await dispatch(downloadSingleFile(files[0]));
    } else {
      await dispatch(downloadMultipleFiles(files));
    }
  };
}

export function downloadMultipleFiles(files: FileResponse[]): AppThunk {
  return async (dispatch, _getState) => {
    // Prepare download options
    const options: MultipleDownloadOption[] = [MultipleDownloadOption.StreamSaver];
    // @ts-ignore
    if (window.isSecureContext && window.showDirectoryPicker) {
      options.push(MultipleDownloadOption.Browser);
    }

    const groupPermission = new Boolset(SessionManager.currentUser()?.group?.permission);
    if (
      groupPermission.enabled(GroupPermission.archive_download) &&
      (files.length > 1 || !files[0].metadata?.[Metadata.share_redirect])
    ) {
      options.push(MultipleDownloadOption.Backend);
    }

    let finalOption = options[0];
    if (options.length > 1) {
      try {
        finalOption = (await dispatch(
          selectOption(getDownloadSelectOption(options), "fileManager.selectArchiveMethod"),
        )) as MultipleDownloadOption;
      } catch (e) {
        // User cancel selection
        return;
      }
    }

    if (finalOption == MultipleDownloadOption.Backend) {
      await dispatch(backendBatchDownload(files));
    } else if (finalOption == MultipleDownloadOption.Browser) {
      await dispatch(browserBatchDownload(files));
    } else {
      await dispatch(streamSaverDownload(files));
    }
  };
}

export function backendBatchDownload(files: FileResponse[]): AppThunk {
  return async (dispatch, _getState) => {
    const downloadUrl = await longRunningTaskWithSnackbar(
      dispatch(
        getFileEntityUrl({
          uris: files.map((f) => getFileLinkedUri(f)),
          archive: true,
        }),
      ),
      "application:fileManager.preparingBathDownload",
    );

    window.location.assign(downloadUrl.urls[0].url);
  };
}

export const cancelSignals: {
  [key: string]: AbortController;
} = {};

export function browserBatchDownload(files: FileResponse[]): AppThunk {
  return async (dispatch, getState) => {
    const downloadId = promiseId();
    cancelSignals[downloadId] = new AbortController();

    // Select download folder
    let handle: FileSystemDirectoryHandle;
    if (!window.showDirectoryPicker || !window.isSecureContext) {
      return;
    }
    try {
      // can't use suggestedName for showDirectoryPicker (only available showSaveFilePicker)
      handle = await window.showDirectoryPicker({
        startIn: "downloads",
        mode: "readwrite",
      });
      // we should obtain the readwrite permission for the directory at first
      if (!(await verifyFileSystemRWPermission(handle))) {
        enqueueSnackbar({
          message: i18next.t("application:fileManager.directoryDownloadPermissionError"),
          variant: "error",
        });
        throw new Error(i18next.t("application:fileManager.directoryDownloadPermissionError"));
      }
    } catch (e) {
      return;
    }

    dispatch(
      setBatchDownloadProgress({
        id: downloadId,
        progress: {
          isDownloading: true,
          filesCompleted: 0,
          totalFiles: 0,
          totalBytes: 0,
          totalExpectedBytes: 0,
          currentFile: "",
          speed: 0,
        },
      }),
    );

    const beforeUnloadHandler = (e: BeforeUnloadEvent) => {
      e.preventDefault();
      e.returnValue = "";
    };
    window.addEventListener("beforeunload", beforeUnloadHandler);

    const snackbarId = enqueueSnackbar({
      message: i18next.t("fileManager.batchDownloadStarted"),
      variant: "loading",
      persist: true,
      action: BatchDownloadProgressAction(downloadId),
      secondaryAction: BatchDownloadSecondaryAction(downloadId),
      getProgress: () => {
        const progress = getState().globalState.batchDownloadProgress?.[downloadId];
        if (!progress || progress.totalExpectedBytes === 0) return 0;
        return Math.min(100, Math.round((progress.totalBytes / progress.totalExpectedBytes) * 100));
      },
    });

    // Track cancellation locally because walk() catches and swallows AbortError
    // rather than re-throwing it, so the download resolves normally after cancel.
    let cancelled = false;
    const origAbort = cancelSignals[downloadId].abort.bind(cancelSignals[downloadId]);
    cancelSignals[downloadId].abort = () => {
      cancelled = true;
      origAbort();
    };

    try {
      await dispatch(startBrowserBatchDownloadTo(handle, downloadId, files));
      dispatch(setBatchDownloadProgress({ id: downloadId, progress: { isDownloading: false } }));
      closeSnackbar(snackbarId);
      if (!cancelled) {
        enqueueSnackbar({
          message: i18next.t("fileManager.downloadComplete"),
          variant: "success",
          autoHideDuration: 5000,
          action: BatchDownloadCompleteAction(downloadId),
        });
      } else {
        enqueueSnackbar({
          message: i18next.t("modals.directoryDownloadCancelled"),
          variant: "warning",
          autoHideDuration: 3000,
          action: DefaultCloseAction,
        });
      }
    } catch (e) {
      closeSnackbar(snackbarId);
      dispatch(setBatchDownloadProgress({ id: downloadId, progress: { isDownloading: false } }));
    } finally {
      window.removeEventListener("beforeunload", beforeUnloadHandler);
    }
  };
}

function startBrowserBatchDownloadTo(
  handle: FileSystemDirectoryHandle,
  downloadId: string,
  files: FileResponse[],
): AppThunk<Promise<void>> {
  return async (dispatch, _getState): Promise<void> => {
    let log = "";
    let failed = 0;
    let skipAll = false;
    let overwriteAll = false;
    let filesCompleted = 0;
    let totalBytes = 0;
    let totalFiles = 0;
    let totalExpectedBytes = 0;

    // Speed tracking
    let lastBytes = 0;
    let lastTime = Date.now();
    const speedSamples: number[] = [];

    let lastDispatchTime = 0;
    let currentFile = "";

    const appendLog = (msg: string) => {
      log = log + msg + "\n";
      dispatch(setBatchDownloadLog({ id: downloadId, logs: log }));
    };

    const dispatchProgress = () => {
      const speed = speedSamples.length > 0 ? speedSamples.reduce((a, b) => a + b, 0) / speedSamples.length : 0;

      dispatch(
        setBatchDownloadProgress({
          id: downloadId,
          progress: {
            filesCompleted,
            totalFiles,
            totalBytes,
            totalExpectedBytes,
            currentFile,
            speed,
            isDownloading: true,
          },
        }),
      );
      lastDispatchTime = Date.now();
    };

    const updateProgress = (force?: boolean) => {
      // Calculate speed (rolling 5-sample average, sampled every 500ms)
      const now = Date.now();
      const elapsed = (now - lastTime) / 1000;
      if (elapsed >= 0.5) {
        const recentBytes = totalBytes - lastBytes;
        const instantSpeed = recentBytes / elapsed;
        speedSamples.push(instantSpeed);
        if (speedSamples.length > 5) speedSamples.shift();
        lastBytes = totalBytes;
        lastTime = now;
      }

      // Throttle dispatches to at most every 500ms unless forced (file complete, etc.)
      if (force || now - lastDispatchTime >= 500) {
        dispatchProgress();
      }
    };

    // Count total files and total expected bytes by walking the tree
    await dispatch(
      walk(files, async (children) => {
        const childFiles = children.filter((f) => f.type == FileType.file);
        totalFiles += childFiles.length;
        totalExpectedBytes += childFiles.reduce((sum, f) => sum + (f.size || 0), 0);
      }),
    );

    updateProgress(true);

    // get the files in the directory to compare with queue files
    const fsPaths = await getFileSystemDirectoryPaths(handle, "");

    await dispatch(
      walk(files, async (children, relativePath) => {
        const childFiles = children.filter((f) => f.type == FileType.file);
        try {
          const entityUrls = await dispatch(
            getFileEntityUrl({
              uris: childFiles.map((f) => getFileLinkedUri(f)),
              download: true,
              skip_error: true,
            }),
          );

          for (let i = 0; i < entityUrls.urls.length; i++) {
            if (!entityUrls.urls[i]) {
              appendLog(
                i18next.t("modals.directoryDownloadErrorNotification", {
                  name: childFiles[i].name,
                  msg: "failed to get download url",
                }),
              );
              failed++;
              filesCompleted++;
              updateProgress(true);
              continue;
            }
            const name = (relativePath == "" ? "" : relativePath + "/") + childFiles[i].name;
            if (fsPaths.has(name)) {
              if (skipAll) {
                appendLog(
                  i18next.t("modals.directoryDownloadSkipNotifiction", {
                    name,
                  }),
                );
                filesCompleted++;
                updateProgress(true);
                continue;
              }

              if (overwriteAll) {
                appendLog(
                  i18next.t("modals.directoryDownloadReplaceNotifiction", {
                    name,
                  }),
                );
              } else {
                // No overwrite options, ask for one
                let overwriteOption = DownloadOverwriteOption.Skip;
                try {
                  overwriteOption = (await dispatch(
                    selectOption(getDownloadOverwriteOption(name), "fileManager.selectDirectoryDuplicationMethod"),
                  )) as DownloadOverwriteOption;
                } catch (e) {
                  // User cancel, use skip option
                  overwriteOption = DownloadOverwriteOption.Skip;
                }

                if (overwriteOption == DownloadOverwriteOption.Skip) {
                  appendLog(
                    i18next.t("modals.directoryDownloadSkipNotifiction", {
                      name,
                    }),
                  );
                  filesCompleted++;
                  updateProgress(true);
                  continue;
                } else if (overwriteOption == DownloadOverwriteOption.SkipAll) {
                  appendLog(
                    i18next.t("modals.directoryDownloadSkipNotifiction", {
                      name,
                    }),
                  );
                  skipAll = true;
                  filesCompleted++;
                  updateProgress(true);
                  continue;
                } else if (overwriteOption == DownloadOverwriteOption.OverwriteAll) {
                  appendLog(
                    i18next.t("modals.directoryDownloadReplaceNotifiction", {
                      name,
                    }),
                  );
                  overwriteAll = true;
                } else {
                  appendLog(
                    i18next.t("modals.directoryDownloadReplaceNotifiction", {
                      name,
                    }),
                  );
                }
              }
            }

            appendLog(i18next.t("modals.directoryDownloadStarted", { name }));
            currentFile = name;

            updateProgress(true);
            try {
              const res = await fetch(entityUrls.urls[i].url, {
                signal: cancelSignals[downloadId].signal,
              });

              // Stream the response directly to disk to avoid buffering
              // the entire file in memory
              const reader = res.body?.getReader();
              if (!reader) {
                throw new Error("Response body is not readable");
              }

              await streamFileToFileSystemDirectory(handle, reader, name, (bytes) => {
                totalBytes += bytes;
                updateProgress();
              });
              filesCompleted++;
              updateProgress(true);
              appendLog(i18next.t("modals.directoryDownloadFinished", { name }));
            } catch (e) {
              // User cancel download
              if (e instanceof Error && e.name == "AbortError") {
                appendLog(i18next.t("modals.directoryDownloadCancelled"));
                throw e;
              }
              failed++;
              filesCompleted++;
              updateProgress(true);
              appendLog(
                i18next.t("modals.directoryDownloadErrorNotification", {
                  name: name,
                  msg: (e as Error).message,
                }),
              );
            }
          }
        } catch (e) {
          if (e instanceof Error && e.name == "AbortError") {
            throw e;
          }
          failed += childFiles.length;
          filesCompleted += childFiles.length;
          updateProgress(true);
          appendLog(
            i18next.t("modals.directoryDownloadError", {
              msg: (e as Error).message,
            }),
          );
        }
      }),
    );

    if (failed === 0) {
      appendLog(i18next.t("fileManager.directoryDownloadFinished"));
    } else {
      appendLog(i18next.t("fileManager.directoryDownloadFinishedWithError", { failed }));
    }
  };
}

export function streamSaverDownload(files: FileResponse[]): AppThunk {
  return async (dispatch, getState) => {
    const allFiles = (
      await longRunningTaskWithSnackbar(dispatch(walkAll(files)), "application:fileManager.preparingBathDownload")
    ).filter((f) => f.type == FileType.file);

    const fileStream = streamSaver.createWriteStream(formatLocalTime(dayjs()) + ".zip");
    const {
      siteConfig: {
        explorer: {
          config: { max_batch_size },
        },
      },
    } = getState();
    const maxBatch = Math.min(10, max_batch_size ?? 1);
    let current = 0;

    const readableZipStream = new (window as any).ZIP({
      start(_ctrl: any) {
        // ctrl.close()
      },
      async pull(ctrl: any) {
        const batch = allFiles.slice(current, current + maxBatch);
        current += batch.length;
        if (batch.length == 0) {
          ctrl.close();
          return;
        }

        try {
          const entityUrls = await dispatch(
            getFileEntityUrl({
              uris: batch.map((f) => getFileLinkedUri(f)),
              skip_error: true,
              download: true,
            }),
          );

          for (let i = 0; i < entityUrls.urls.length; i++) {
            const url = entityUrls.urls[i];
            if (!url) {
              continue;
            }
            const res = await fetch(url.url);
            const stream = () => res.body;
            ctrl.enqueue({ name: batch[i].relativePath, stream });
          }
        } catch (e) {
          console.warn("Failed to get entity urls", e);
        }
      },
    });

    if (window.WritableStream && readableZipStream.pipeTo) {
      try {
        await longRunningTaskWithSnackbar(readableZipStream.pipeTo(fileStream), "fileManager.batchDownloadStarted");
      } catch (e) {
        console.log(e);
      }
    }
  };
}

export function downloadSingleFile(file: FileResponse, preferredEntity?: string): AppThunk {
  return async (dispatch, _getState) => {
    const isSharedFile = file.metadata?.[Metadata.share_redirect] ?? false;
    if (isSharedFile) {
      file = await dispatch(refreshSingleFileSymbolicLinks(file));
    }

    const urlRes = await longRunningTaskWithSnackbar(
      dispatch(
        getFileEntityUrl({
          uris: [getFileLinkedUri(file)],
          entity: preferredEntity,
          download: true,
        }),
      ),
      "application:fileManager.preparingDownload",
    );

    const streamSaverName = urlRes.urls[0].stream_saver_display_name;
    if (streamSaverName) {
      // remove streamSaverParam from query
      const fileStream = streamSaver.createWriteStream(streamSaverName);
      const res = await fetch(urlRes.urls[0].url);
      const readableStream = res.body;
      if (!readableStream) {
        return;
      }

      // more optimized
      if (window.WritableStream && readableStream.pipeTo) {
        const downloadingSnackbar = enqueueSnackbar({
          message: i18next.t("fileManager.downloadingFile", {
            name: streamSaverName,
          }),
          variant: "loading",
          persist: true,
        });
        return readableStream.pipeTo(fileStream).finally(() => closeSnackbar(downloadingSnackbar));
      }
    } else {
      window.location.assign(urlRes.urls[0].url);
    }
  };
}

const getDownloadSelectOption = (options: MultipleDownloadOption[]): DialogSelectOption[] => {
  return options.map((option): DialogSelectOption => {
    switch (option) {
      case MultipleDownloadOption.Backend:
        return {
          value: MultipleDownloadOption.Backend,
          name: i18next.t("fileManager.serverBatchDownload"),
          description: i18next.t("fileManager.serverBatchDownloadDescription"),
        };
      case MultipleDownloadOption.Browser:
        return {
          value: MultipleDownloadOption.Browser,
          name: i18next.t("fileManager.browserDownload"),
          description: i18next.t("fileManager.browserDownloadDescription"),
        };
      default:
        return {
          value: MultipleDownloadOption.StreamSaver,
          name: i18next.t("fileManager.browserBatchDownload"),
          description: i18next.t("fileManager.browserBatchDownloadDescription"),
        };
    }
  });
};

const getDownloadOverwriteOption = (name: string): DialogSelectOption[] => {
  return [
    {
      name: i18next.t("fileManager.directoryDownloadReplace"),
      description: i18next.t("fileManager.directoryDownloadReplaceDescription", { name }),
      value: DownloadOverwriteOption.Overwrite,
    },
    {
      name: i18next.t("fileManager.directoryDownloadSkip"),
      description: i18next.t("fileManager.directoryDownloadSkipDescription", {
        name,
      }),
      value: DownloadOverwriteOption.Skip,
    },
    {
      name: i18next.t("fileManager.directoryDownloadReplaceAll"),
      description: i18next.t("fileManager.directoryDownloadReplaceAllDescription", { name }),
      value: DownloadOverwriteOption.OverwriteAll,
    },
    {
      name: i18next.t("fileManager.directoryDownloadSkipAll"),
      description: i18next.t("fileManager.directoryDownloadSkipAllDescription", { name }),
      value: DownloadOverwriteOption.SkipAll,
    },
  ];
};
