import dayjs from "dayjs";
import i18next from "i18next";
import { closeSnackbar, enqueueSnackbar, SnackbarAction, SnackbarKey } from "notistack";
import {
  getFileDirectLinks,
  getFileEntityUrl,
  getFileList,
  getFileThumb,
  sendCreateFile,
  sendDeleteFiles,
  sendEmptyTrash,
  sendMetadataPatch,
  sendMoveFile,
  sendRenameFile,
  sendRestoreFile,
  sendUnlockFiles,
  setCurrentVersion,
} from "../../api/api.ts";
import {
  ConflictDetail,
  DirectLink,
  FileResponse,
  FileType,
  ListResponse,
  Metadata,
  MetadataPatch,
  NewFileTemplate,
  Viewer,
} from "../../api/explorer.ts";
import { AppError, Response } from "../../api/request.ts";
import { DefaultCloseAction, ViewDstAction } from "../../component/Common/Snackbar/snackbar.tsx";
import { canShowInfo, getActionOpt } from "../../component/FileManager/ContextMenu/useActionDisplayOpt.ts";
import { DeleteOption } from "../../component/FileManager/Dialogs/DeleteConfirmation.tsx";
import { PathSelectionVariantOptions } from "../../component/FileManager/Dialogs/PathSelection.tsx";
import { Tag } from "../../component/FileManager/Dialogs/Tags.tsx";
import { DragItem, DropEffect, DropResult } from "../../component/FileManager/Dnd/DndWrappedFile.tsx";
import { FileManagerIndex } from "../../component/FileManager/FileManager.tsx";
import { MinPageSize } from "../../component/FileManager/TopBar/ViewOptionPopover.tsx";
import { loadingDebounceMs } from "../../constants";
import { defaultPath } from "../../hooks/useNavigation.tsx";
import SessionManager, { UserSettings } from "../../session";
import { addRecentUsedColor, addUsedTags } from "../../session/utils.ts";
import { fileExtension, getFileLinkedUri } from "../../util";
import Boolset from "../../util/boolset.ts";
import { canCopyMoveTo } from "../../util/permission.ts";
import CrUri, { Filesystem } from "../../util/uri.ts";
import {
  addSelected,
  appendTreeCache,
  clearSelected,
  closeContextMenu,
  closeRenameFileModal,
  ContextMenuTypes,
  fileUpdated,
  removeSelected,
  removeThumbCache,
  removeTreeCache,
  setContextMenu,
  setFileDeleteModal,
  setFileDeleteModalLoading,
  setFileList,
  setRenameFileModalError,
  setRenameFileModalLoading,
  setSelected,
  setThumbCache,
} from "../fileManagerSlice.ts";
import {
  CreateNewDialogType,
  setDirectLinkDialog,
  setExtractArchiveDialog,
  setRemoteDownloadDialog,
  setShareLinkDialog,
  setSidebar,
  updateLockConflicts,
} from "../globalStateSlice.ts";
import { ConfigLoadState, Viewers } from "../siteConfigSlice.ts";
import { AppThunk } from "../store.ts";
import { confirmOperation, deleteConfirmation, renameForm, requestCreateNew, selectPath } from "./dialog.ts";
import { downloadSingleFile } from "./download.ts";
import { navigateToPath, refreshFileList, updateUserCapacity } from "./filemanager.ts";
import { queueLoadShareInfo } from "./share.ts";
import { loadSiteConfig } from "./site.ts";
import { openViewer, openViewers } from "./viewer.ts";

const contextMenuCloseAnimationDelay = 250;

function get_platform(): string {
  // 2022 way of detecting. Note : this userAgentData feature is available only in secure contexts (HTTPS)
  if (typeof navigator.userAgentData !== "undefined" && navigator.userAgentData != null) {
    return navigator.userAgentData.platform;
  }
  // Deprecated but still works for most of the browser
  if (typeof navigator.platform !== "undefined") {
    return navigator.platform;
  }
  return "unknown";
}

export const isMac = () => {
  return get_platform().toUpperCase().indexOf("MAC") >= 0;
};

export const isMacbook = isMac();

export function loadFileThumb(index: number, file: FileResponse): AppThunk<Promise<string | null>> {
  return async (dispatch, getState) => {
    const { fileManager } = getState();
    const cache = fileManager[index].tree[file.path]?.thumb;
    if (cache) {
      if (!cache.expires || dayjs(cache.expires).isAfter(dayjs())) {
        return cache.src;
      }
    }

    try {
      const thumb = await dispatch(getFileThumb(getFileLinkedUri(file), fileManager[index].list?.context_hint));
      dispatch(setThumbCache({ index, value: [file.path, thumb] }));
      return thumb.url;
    } catch (e) {
      dispatch(
        fileUpdated({
          index,
          value: [
            {
              file: { ...file, metadata: { ...file.metadata, [Metadata.thumbDisabled]: "" } },
              oldPath: file.path,
              includeMetadata: true,
            },
          ],
        }),
      );
      console.warn("Failed to load thumb", e);
      return null;
    }
  };
}

export function enterFolder(index: number, folder: FileResponse): AppThunk {
  return async (dispatch, _getState) => {
    dispatch(closeContextMenu({ index, value: undefined }));
    setTimeout(() => {
      dispatch(navigateToPath(index, folder.path, folder));
    }, contextMenuCloseAnimationDelay);
  };
}

export function goToParent(index: number, folder: FileResponse): AppThunk {
  return async (dispatch, _getState) => {
    dispatch(closeContextMenu({ index, value: undefined }));
    const path = new CrUri(folder.path).parent().toString();
    setTimeout(() => {
      dispatch(navigateToPath(index, path));
    }, contextMenuCloseAnimationDelay);
  };
}

export function openFile(index: number, file: FileResponse): AppThunk {
  return async (dispatch, _getState) => {
    if (file.type == FileType.folder) {
      dispatch(navigateToPath(index, file.path, file));
      return;
    } else {
      dispatch(openViewers(index, file));
    }
  };
}

let lastSelectedIndex: number = 0;
export function selectFile(index: number, file: FileResponse, e?: React.MouseEvent<HTMLElement>): AppThunk {
  return async (dispatch, getState) => {
    let ctrlKey: boolean = false,
      metaKey: boolean = false,
      shiftKey: boolean = false;
    if (e) {
      ctrlKey = e.ctrlKey;
      metaKey = e.metaKey;
      shiftKey = e.shiftKey;
      if ([ctrlKey, shiftKey].filter(Boolean).length > 1 || [metaKey, shiftKey].filter(Boolean).length > 1) {
        return;
      }
    }

    const {
      fileManager,
      globalState: { sidebarOpen },
    } = getState();
    const { selected, list } = fileManager[index];

    if (!list) {
      return;
    }

    if (sidebarOpen) {
      const cap = new Boolset(file.capability);
      if (canShowInfo(cap)) {
        dispatch(switchSidebarTarget(file));
      }
    }

    const isRangeSelection = shiftKey && !ctrlKey && !metaKey && Object.keys(selected).length !== 0;
    const isCtrlSelection = (ctrlKey && !isMacbook) || (metaKey && isMacbook);

    const fileIndex = list.files.findIndex((f) => f.id === file.id);
    if (isRangeSelection) {
      const begin = Math.min(lastSelectedIndex, fileIndex);
      const end = Math.max(lastSelectedIndex, fileIndex);
      const newShiftSelected = list.files.slice(begin, end + 1);
      return dispatch(setSelected({ index, value: newShiftSelected }));
    }

    lastSelectedIndex = fileIndex;

    if (isCtrlSelection && index != FileManagerIndex.selector) {
      // Ctrl/Command 单击添加/删除
      const currentSelected = selected[file.path];
      if (currentSelected) {
        return dispatch(removeSelected({ index, value: file.path }));
      }
      return dispatch(addSelected({ index, value: [file] }));
    }
    // 单选
    return dispatch(setSelected({ index, value: [file] }));
  };
}

export function switchSidebarTarget(file?: FileResponse): AppThunk {
  return async (dispatch, getState) => {
    const sidebarOpen = getState().globalState.sidebarOpen;
    if (sidebarOpen) {
      dispatch(setSidebar({ open: true, target: file }));
    }
  };
}

export function fileClicked(index: number, file: FileResponse, e?: React.MouseEvent<HTMLElement>): AppThunk {
  return async (dispatch, _getState) => {
    const ctrlKeyPressed = e && (e.ctrlKey || e.metaKey || e.shiftKey);
    const isTrash = file.metadata && file.metadata[Metadata.restore_uri];
    const folderClickAction = SessionManager.getWithFallback(UserSettings.FolderClickAction);
    if (file.type === FileType.folder && !ctrlKeyPressed && !isTrash && folderClickAction === "open") {
      dispatch(openFile(index, file));
    } else {
      dispatch(selectFile(index, file, e));
    }

    if (e) {
      e.stopPropagation();
    }
  };
}

export function fileDoubleClicked(index: number, file: FileResponse, e?: React.MouseEvent<HTMLElement>): AppThunk {
  return async (dispatch, _getState) => {
    const actionOpt = getActionOpt([file], Viewers);
    if (actionOpt.showOpen || actionOpt.showEnter) {
      dispatch(openFile(index, file));
    } else if (file.type == FileType.file && actionOpt.showDownload) {
      dispatch(downloadSingleFile(file));
    }
  };
}

export function fileIconClicked(index: number, file: FileResponse, e?: React.MouseEvent<HTMLElement>): AppThunk {
  return async (dispatch, _getState) => {
    if (e) {
      e.stopPropagation();
      if (!e.shiftKey) {
        if (isMacbook) {
          e.metaKey = true;
        } else {
          e.ctrlKey = true;
        }
      }
    }
    dispatch(selectFile(index, file, e));
    return false;
  };
}

export function selectionFromDragBox(index: number, fileIndex: string[], ctrlKey: boolean, metaKey: boolean): AppThunk {
  return async (dispatch, getState) => {
    const { fileManager } = getState();
    const list = fileManager[index].list?.files;
    const newSelected: FileResponse[] = [];
    fileIndex.forEach((i) => {
      const iInt = parseInt(i, 10);
      if (list && !isNaN(iInt)) {
        newSelected.push(list[iInt]);
      }
    });
    if ((ctrlKey && !isMacbook) || (metaKey && isMacbook)) {
      return dispatch(addSelected({ index, value: newSelected }));
    }

    return dispatch(setSelected({ index, value: newSelected }));
  };
}

export function openFileContextMenu(
  index: number,
  file: FileResponse,
  selectedOverwrite: boolean,
  e: React.MouseEvent<HTMLElement>,
  type: string = ContextMenuTypes.file,
  useExactMousePos: boolean = true,
): AppThunk {
  return async (dispatch, getState) => {
    e.preventDefault();
    e.stopPropagation();
    if (index == FileManagerIndex.selector) {
      return;
    }

    let targets: { [key: string]: FileResponse } | undefined = undefined;
    const { fileManager } = getState();
    if (selectedOverwrite) {
      targets = {
        [file.path]: file,
      };
    } else if (!fileManager[index].selected[file.path]) {
      // If not selected, set to the only selected file
      dispatch(setSelected({ index, value: [file] }));
    }

    const rect = useExactMousePos ? undefined : e.currentTarget?.getBoundingClientRect();
    const { x, y } = rect
      ? { x: rect.x, y: rect.bottom }
      : {
          x: e.clientX,
          y: e.clientY,
        };
    dispatch(
      setContextMenu({
        index,
        value: {
          open: true,
          pos: { x, y },
          type: type,
          targets: targets,
          fmIndex: index,
        },
      }),
    );
  };
}

export function forceUnlockFiles(tokens: string[]): AppThunk<Promise<void>> {
  return async (dispatch, getState) => {
    try {
      await dispatch(confirmOperation(i18next.t("application:modals.forceUnlockDes")));
    } catch (e) {
      return;
    }

    await dispatch(
      sendUnlockFiles({
        tokens,
      }),
    );

    const original = getState().globalState.lockConflictError;
    if (!original) {
      return;
    }
    const currentConflicts: Response<ConflictDetail[]> = {
      ...original,
    };
    currentConflicts.data = currentConflicts.data?.filter((conflict) => {
      return !conflict.token || !tokens.includes(conflict.token);
    });
    dispatch(updateLockConflicts(currentConflicts));
  };
}

export function submitRenameFile(index: number, file: FileResponse, newName: string): AppThunk<Promise<boolean>> {
  return async (dispatch, getState) => {
    let newFile: FileResponse | undefined = undefined;
    try {
      newFile = await dispatch(
        sendRenameFile({
          uri: file.path,
          new_name: newName,
        }),
      );
    } catch (e) {
      return false;
    }

    if (newFile) {
      dispatch(
        fileUpdated({
          index,
          value: [
            {
              file: newFile,
              oldPath: file.path,
            },
          ],
        }),
      );

      const currentPath = getState().fileManager[index].path;
      if (currentPath) {
        if (
          currentPath.startsWith(file.path + "/") ||
          currentPath.startsWith(file.path + "?") ||
          currentPath == file.path
        ) {
          dispatch(navigateToPath(index, currentPath.replace(file.path, newFile.path)));
        }
      }
    }
    return !!newFile;
  };
}

export function inlineRenameSubmit(
  index: number,
  file: FileResponse,
  newName: string,
): AppThunk<Promise<string | undefined>> {
  return async (dispatch, getState) => {
    const current = getState().fileManager[index].list?.files;
    const existingNames = current?.filter((f) => f.name !== file.name).map((f) => f.name) ?? [];
    const error = validateFileNamePure(newName, existingNames);
    if (error) return error;

    await dispatch(submitRenameFile(index, file, newName));
    return undefined;
  };
}

export function renameFile(index: number, file: FileResponse): AppThunk<Promise<void>> {
  return async (dispatch, _getState) => {
    dispatch(closeContextMenu({ index, value: undefined }));
    let newName = "";
    try {
      newName = await dispatch(renameForm(index, file));
    } catch (e) {
      // operation canceled
      return;
    }

    dispatch(setRenameFileModalLoading({ index, value: true }));

    try {
      await dispatch(submitRenameFile(index, file, newName));
    } finally {
      dispatch(closeRenameFileModal({ index, value: undefined }));
    }
  };
}

const special = '\\/:*?"<>|';

export function validateFileNamePure(newName: string, existingNames: string[]): string | undefined {
  if (newName.length === 0 || newName.length > 255) {
    return i18next.t("application:modals.newNameLengthError");
  }

  if (newName.split("").some((c) => special.includes(c))) {
    return i18next.t("application:modals.newNameCharacterError");
  }

  if (newName == "." || newName == "..") {
    return i18next.t("application:modals.newNameDotError");
  }

  if (existingNames.includes(newName)) {
    return i18next.t("application:modals.duplicatedObjectName");
  }

  return undefined;
}

export function validateFileName(
  index: number,
  resolve: (value: string | PromiseLike<string>) => void,
  newName: string,
): AppThunk<Promise<void>> {
  return async (dispatch, getState) => {
    const current = getState().fileManager[index].list?.files;
    const existingNames = current?.map((f) => f.name) ?? [];
    const error = validateFileNamePure(newName, existingNames);

    if (error) {
      dispatch(setRenameFileModalError({ index, value: error }));
      return;
    }

    dispatch(setRenameFileModalError({ index, value: undefined }));
    resolve(newName);
  };
}

export function deleteFile(index: number, files: FileResponse[]): AppThunk<Promise<void>> {
  return async (dispatch, _getState) => {
    dispatch(closeContextMenu({ index, value: undefined }));

    // Ask for confirmation and advanced delete options
    let opt: DeleteOption;
    try {
      opt = await dispatch(deleteConfirmation(index, files));
    } catch (e) {
      // operation canceled
      return;
    }

    dispatch(setFileDeleteModalLoading({ index, value: true }));

    let deleted = files;
    try {
      await dispatch(
        sendDeleteFiles({
          uris: files.map((f) => f.path),
          unlink: opt.unlink,
          skip_soft_delete: opt.skip_soft_delete,
        }),
      );
    } catch (e) {
      dispatch(setFileDeleteModal({ index, value: [false, files, undefined, true] }));
      const partialSuccess = e instanceof AppError && e.aggregatedError;
      if (!partialSuccess) {
        return;
      }

      // remove failed files from list
      if (e instanceof AppError) {
        const err = e.aggregatedError;
        deleted = deleted.filter((f) => !err || !err[f.path]);
      }
    }

    dispatch(setFileDeleteModal({ index, value: [false, files, undefined, true] }));

    await dispatch(processFileListDiff(FileManagerIndex.main, deleted));
    await dispatch(processFileListDiff(FileManagerIndex.selector, deleted, false));

    dispatch(updateUserCapacity(index));
  };
}

function processFileListDiff(index: number, deleted: FileResponse[], refreshIfNeeded: boolean = true): AppThunk {
  return async (dispatch, getState) => {
    const fm = getState().fileManager[index];
    let potentialParents: string[] = [];
    dispatch(removeTreeCache({ index, value: deleted.map((f) => f.path) }));
    if (!fm.path) {
      return;
    }
    let newNavigatePath = fm.path;

    // Find all parent paths of current
    const currentUri = new CrUri(fm.path);
    const elements = currentUri.elements();
    const base = currentUri.base();
    potentialParents.push(base);

    elements.forEach((_p, index) => {
      potentialParents.push(new CrUri(base).join(...elements.slice(0, index + 1)).toString());
    });

    deleted.forEach((f) => {
      // Filter out potential parents
      const index = potentialParents.findIndex((p) => f.path == p);
      if (index > 0) {
        potentialParents.splice(index);
      }
    });
    if (potentialParents.length > 0) {
      newNavigatePath = potentialParents[potentialParents.length - 1];
    }

    const newNavigateUri = new CrUri(newNavigatePath);
    const isCursorPagination = fm.list?.pagination.is_cursor;
    if (isCursorPagination) {
      // remove deleted files from selected
      const newSelected = { ...fm.selected };
      Object.keys(newSelected).forEach((p) => {
        if (deleted.find((f) => f.path === p)) {
          delete newSelected[p];
        }
      });
      dispatch(setSelected({ index, value: Object.values(newSelected) }));

      // For cursor pagination, Remove file from file list
      const newList = fm.list?.files.filter((f) => !deleted.find((d) => d.path === f.path));
      if (newList) {
        dispatch(setFileList({ index, value: newList }));
      }
    } else {
      // For offset pagination, refresh file list
      if (refreshIfNeeded && newNavigateUri.pure_uri().toString() == currentUri.pure_uri().toString()) {
        dispatch(refreshFileList(index));
      }
    }

    // Retreat to parent if current path is deleted
    if (
      refreshIfNeeded &&
      newNavigateUri.pure_uri().toString() != currentUri.pure_uri().toString() &&
      newNavigatePath
    ) {
      dispatch(navigateToPath(index, newNavigatePath));
    }
  };
}

export function processDnd(index: number, src: DragItem, dst: DropResult): AppThunk {
  return async (dispatch, getState) => {
    if (!dst.uri) return;

    const fm = getState().fileManager[index];
    const srcFiles = [
      src.target,
      ...(src.includeSelected ? Object.values(fm.selected).filter((f) => f.path != src.target.path) : []),
    ];

    const isCopy = dst.dropEffect == DropEffect.copy;
    return dispatch(moveFiles(index, srcFiles, dst.uri, isCopy));
  };
}

// ignore if exist src file that is equal to dst file, or the parent of dst file
// TODO: validate bypass by share
function checkMovePrecondition(srcFiles: FileResponse[], dst: string) {
  return srcFiles.some((src) => {
    if (src.path === dst || getFileLinkedUri(src) == dst) return true;
    if (dst.startsWith(src.path + "/")) {
      enqueueSnackbar({
        message: i18next.t("application:modals.cannotMoveCopyToChild"),
        variant: "warning",
        action: DefaultCloseAction,
      });
      return true;
    }
    const srcUri = new CrUri(src.path);
    if (srcUri.parent().toString() == dst) {
      return true;
    }
  });
}

export function moveFiles(index: number, src: FileResponse[], dst: string, isCopy: boolean): AppThunk {
  return async (dispatch, getState) => {
    const dstUri = new CrUri(dst);
    if (
      !src.find((f) => f.metadata && f.metadata[Metadata.restore_uri]) &&
      dstUri.fs() == Filesystem.trash &&
      dstUri.elements().length == 0
    ) {
      return dispatch(deleteFile(index, src));
    }

    if (checkMovePrecondition(src, dst)) {
      return;
    }

    if (!canCopyMoveTo(src, dst, isCopy)) {
      enqueueSnackbar({
        message: i18next.t("application:modals.cannotPerformAction"),
        variant: "warning",
        action: DefaultCloseAction,
      });
      return;
    }

    let success = true;
    try {
      await longRunningTaskWithSnackbar(
        dispatch(sendMoveFile({ uris: src.map((f) => f.path), dst, copy: isCopy })),
        isCopy ? "application:modals.processingCopying" : "application:modals.processingMoving",
      );
    } catch (e) {
      success = false;
    }

    if (isCopy) {
      if (dst == getState().fileManager[index].path) dispatch(refreshFileList(index));
    } else if (success) {
      dispatch(processFileListDiff(index, src));
    }

    if (success) {
      enqueueSnackbar({
        message: i18next.t(isCopy ? "application:modals.copySuccess" : "application:modals.moveSuccess", {
          num: src.length,
        }),
        variant: "success",
        action: ViewDstAction(dst),
      });
    }
  };
}

export function restoreFile(index: number, src: FileResponse[]): AppThunk {
  return async (dispatch, _getState) => {
    dispatch(closeContextMenu({ index, value: undefined }));

    let success = true;
    try {
      await longRunningTaskWithSnackbar(
        dispatch(sendRestoreFile({ uris: src.map((f) => f.path) })),
        "application:modals.processingRestoring",
      );
    } catch (e) {
      success = false;
    }

    if (success) {
      dispatch(processFileListDiff(index, src));
    }

    if (success) {
      enqueueSnackbar({
        message: i18next.t("application:modals.fileRestored", {
          num: src.length,
        }),
        variant: "success",
      });
    }
  };
}

export function emptyTrash(index: number): AppThunk {
  return async (dispatch, _getState) => {
    try {
      await dispatch(confirmOperation(i18next.t("application:fileManager.emptyTrashConfirm")));
    } catch (e) {
      // user canceled
      return;
    }

    let success = true;
    try {
      await longRunningTaskWithSnackbar(dispatch(sendEmptyTrash()), "application:modals.processingDeletion");
    } catch (e) {
      success = false;
    }

    if (success) {
      await dispatch(refreshFileList(index));
      dispatch(updateUserCapacity(index));
      enqueueSnackbar({
        message: i18next.t("application:fileManager.trashEmptied"),
        variant: "success",
      });
    }
  };
}

export async function longRunningTaskWithSnackbar<T>(
  task: Promise<T>,
  textKey: string,
  action?: SnackbarAction,
  getProgress?: () => number,
): Promise<T> {
  let loadingId: SnackbarKey | undefined = undefined;
  const delayedLoading = setTimeout(() => {
    loadingId = enqueueSnackbar({
      message: i18next.t(textKey),
      variant: "loading",
      persist: true,
      action: action,
      getProgress: getProgress,
    });
  }, loadingDebounceMs);
  let result: T;
  try {
    result = await task;
    clearTimeout(delayedLoading);
    if (loadingId) {
      closeSnackbar(loadingId);
    }
  } catch (e) {
    throw e;
  } finally {
    clearTimeout(delayedLoading);
    if (loadingId) {
      closeSnackbar(loadingId);
    }
  }

  return result;
}

export function patchFileMetadata(index: number, files: FileResponse[], patches: MetadataPatch[]): AppThunk {
  return async (dispatch, _getState) => {
    await dispatch(sendMetadataPatch({ uris: files.map((f) => f.path), patches }));

    dispatch(
      fileUpdated({
        index,
        value: files.map((f) => {
          const newMetadata = { ...f.metadata };
          patches.forEach((patch) => {
            if (patch.remove) {
              delete newMetadata[patch.key];
            } else {
              newMetadata[patch.key] = patch.value ?? "";
            }
          });
          return {
            file: { ...f, metadata: newMetadata },
            oldPath: f.path,
            includeMetadata: true,
          };
        }),
      }),
    );
  };
}

export function patchCustomProp(
  index: number,
  file: FileResponse,
  propId: string,
  value: string,
  remove?: boolean,
): AppThunk<Promise<void>> {
  return async (dispatch, getState) => {
    const patches: MetadataPatch[] = [
      {
        key: `props:${propId}`,
        value,
        remove,
      },
    ];
    const selected = getState().fileManager[index]?.selected ?? {};
    const targets: { [path: string]: FileResponse } = { [file.path]: file };
    for (const path in selected) {
      targets[path] = selected[path];
    }
    await dispatch(patchFileMetadata(index, Object.values(targets), patches));
  };
}

export function applyIconColor(
  index: number,
  files: FileResponse[],
  color?: string,
  clearSelectedFiles?: boolean,
): AppThunk {
  return async (dispatch, _getState) => {
    const patches: MetadataPatch[] = [
      {
        key: Metadata.icon_color,
        value: color,
        remove: !color,
      },
    ];
    try {
      await dispatch(patchFileMetadata(index, files, patches));
    } catch (e) {
      return;
    }

    if (clearSelectedFiles) {
      dispatch(clearSelected({ index, value: undefined }));
    }

    addRecentUsedColor(color, UserSettings.UsedCustomizedIconColors);
  };
}

export function patchTags(index: number, files: FileResponse[], tags: Tag[]): AppThunk {
  return async (dispatch, _getState) => {
    const patches: MetadataPatch[] = tags.map((tag) => ({
      key: Metadata.tag_prefix + tag.key,
      value: tag.color,
    }));

    // find tags that are removed
    const allTags: { [key: string]: boolean } = {};
    const removedTags: { [key: string]: boolean } = {};
    tags.forEach((tag) => {
      allTags[tag.key] = true;
    });
    files.forEach((file) => {
      if (file.metadata) {
        Object.keys(file.metadata).forEach((key: string) => {
          if (key.startsWith(Metadata.tag_prefix)) {
            // trim prefix for key
            const tagKey = key.slice(Metadata.tag_prefix.length);
            // remove tag if not in allTags
            if (!allTags[tagKey]) {
              removedTags[tagKey] = true;
            }
          }
        });
      }
    });

    Object.keys(removedTags).forEach((key) => {
      patches.push({
        key: Metadata.tag_prefix + key,
        remove: true,
      });
    });

    try {
      await dispatch(patchFileMetadata(index, files, patches));
    } catch (e) {
      return;
    }

    addUsedTags(tags);
  };
}

export function applyIcon(index: number, files: FileResponse[], icon?: string): AppThunk {
  return async (dispatch, _getState) => {
    const patches: MetadataPatch[] = [
      {
        key: Metadata.emoji,
        value: icon,
        remove: !icon,
      },
    ];
    try {
      await dispatch(patchFileMetadata(index, files, patches));
    } catch (e) {
      return;
    }
  };
}

export function openShareDialog(index: number, src: FileResponse): AppThunk {
  return async (dispatch, _getState) => {
    dispatch(closeContextMenu({ index, value: undefined }));
    dispatch(setShareLinkDialog({ open: true, file: src }));
  };
}

export function dialogBasedMoveCopy(index: number, files: FileResponse[], isCopy: boolean): AppThunk {
  return async (dispatch, _getState) => {
    dispatch(closeContextMenu({ index, value: undefined }));
    const dst = await dispatch(
      selectPath(isCopy ? PathSelectionVariantOptions.copy : PathSelectionVariantOptions.move),
    );
    dispatch(moveFiles(index, files, dst, !!isCopy));
  };
}

export function createShareShortcut(index: number): AppThunk {
  return async (dispatch, getState) => {
    const fm = getState().fileManager[index];
    const base = fm?.path_root;
    const isSingleFile = fm?.list?.single_file_view;
    const files = fm?.list?.files;
    if (!base || !files) {
      return;
    }

    let shortcutPath = base;
    if (isSingleFile && files && files.length > 0) {
      shortcutPath = files[0].path;
    }

    const dst = await dispatch(selectPath(PathSelectionVariantOptions.shortcut, defaultPath));
    const dstCrUri = new CrUri(dst);
    const shareCrUri = new CrUri(shortcutPath);
    const shareInfo = await dispatch(queueLoadShareInfo(shareCrUri));
    if (!shareInfo.name || !shareInfo.owner) {
      return;
    }

    dstCrUri.join(shareInfo.name);

    await dispatch(
      sendCreateFile({
        type: shareInfo.source_type == FileType.file ? "file" : "folder",
        uri: dstCrUri.toString(),
        metadata: {
          [Metadata.share_redirect]: shortcutPath,
          [Metadata.share_owner]: shareInfo.owner.id,
        },
        err_on_conflict: true,
      }),
    );

    enqueueSnackbar({
      message: i18next.t("application:modals.shortcutCreated"),
      variant: "success",
      action: ViewDstAction(dst),
    });
  };
}

export function goToSharedLink(index: number, file: FileResponse): AppThunk {
  return async (dispatch, _getState) => {
    dispatch(closeContextMenu({ index, value: undefined }));
    const shareUri = file.metadata?.[Metadata.share_redirect];
    if (shareUri) {
      // Add a delay for animation to close the context menu
      const shareUriParsed = new CrUri(shareUri);
      shareUriParsed.setPath("");
      setTimeout(() => {
        dispatch(navigateToPath(index, shareUriParsed.toString()));
      }, contextMenuCloseAnimationDelay);
    }
  };
}
export function openSidebar(index: number, file: FileResponse): AppThunk {
  return async (dispatch, _getState) => {
    dispatch(closeContextMenu({ index, value: undefined }));
    dispatch(setSidebar({ open: true, target: file }));
  };
}

export function setFileVersion(index: number, file: FileResponse, version: string): AppThunk<Promise<void>> {
  return async (dispatch, _getState) => {
    await dispatch(
      setCurrentVersion({
        uri: file.path,
        version: version,
      }),
    );

    dispatch(
      fileUpdated({
        index,
        value: [
          {
            file: {
              ...file,
              primary_entity: version,
            },
            oldPath: file.path,
          },
        ],
      }),
    );
  };
}

export function getEntityContent(file: FileResponse, version?: string): AppThunk<Promise<ArrayBuffer>> {
  return async (dispatch, _getState) => {
    const urls = await dispatch(
      getFileEntityUrl({
        uris: [getFileLinkedUri(file)],
        entity: version,
      }),
    );
    try {
      const data = await fetch(urls.urls[0].url);
      if (!data.ok) {
        throw new Error(`Failed to load file, response code ${data.status}`);
      }

      return data.arrayBuffer();
    } catch (e) {
      enqueueSnackbar({
        message: i18next.t("application:fileManager.failedToLoadFile", {
          msg: (e as Error).message,
        }),
        variant: "error",
        action: DefaultCloseAction,
      });
      return new ArrayBuffer(0);
    }
  };
}

export function createNew(index: number, type: string, viewer?: Viewer, template?: NewFileTemplate): AppThunk {
  return async (dispatch, _getState) => {
    dispatch(closeContextMenu({ index, value: undefined }));
    let newFile: FileResponse | undefined = undefined;
    let newFileName = "";
    if (template) {
      newFileName = i18next.t("fileManager.newFileName", { ext: template.ext });
    }
    try {
      newFile = await dispatch(
        requestCreateNew(
          index,
          type,
          type == CreateNewDialogType.folder ? i18next.t("application:fileManager.newlyCreatedFolder") : newFileName,
        ),
      );
    } catch (e) {
      return;
    }

    if (viewer) {
      dispatch(openViewer(newFile, viewer, newFile.size));
    }
  };
}

export function submitCreateNew(index: number, name: string, type: number): AppThunk<Promise<FileResponse>> {
  return async (dispatch, getState): Promise<FileResponse> => {
    const fm = getState().fileManager[index];
    const uri = fm.pure_path;
    if (!uri) {
      return Promise.reject("No path to create file");
    }

    const crUri = new CrUri(uri);

    const newFile = await dispatch(
      sendCreateFile({
        type: type == FileType.file ? "file" : "folder",
        uri: crUri.join(name).toString(),
        err_on_conflict: true,
      }),
    );

    // if name does not contain "/", append cache
    if (name.includes("/")) {
      dispatch(refreshFileList(index));
      return newFile;
    }

    const newList = fm.list?.files ? [...fm.list.files, newFile] : [newFile];
    if (newList) {
      dispatch(setFileList({ index: FileManagerIndex.main, value: newList }));
      dispatch(setFileList({ index: FileManagerIndex.selector, value: newList }));
    }
    dispatch(appendTreeCache({ index: FileManagerIndex.main, value: [[newFile], uri] }));
    dispatch(appendTreeCache({ index: FileManagerIndex.selector, value: [[newFile], uri] }));
    dispatch(setSelected({ index, value: [newFile] }));
    return newFile;
  };
}

export function walk(
  files: FileResponse[],
  callback: (f: FileResponse[], relativePath: string) => Promise<any>,
  walkedSymbolicLinks: string[] = [],
  relativePath: string = "",
): AppThunk {
  return async (dispatch, _getState) => {
    await callback(files, relativePath);

    for (const f of files) {
      if (f.type == FileType.folder) {
        var uri = getFileLinkedUri(f);
        if (f.metadata && f.metadata[Metadata.share_redirect]) {
          if (!walkedSymbolicLinks.includes(uri)) {
            walkedSymbolicLinks.push(uri);
          } else {
            // Ignore symbolic link loop
            continue;
          }
        }
        try {
          const allChildren: FileResponse[] = [];
          let nextToken: string | undefined = undefined;
          let page: number | undefined = undefined;
          while (true) {
            const children: ListResponse = await dispatch(
              getFileList({
                next_page_token: nextToken,
                page,
                uri,
                page_size: 1000,
              }),
            );
            allChildren.push(...children.files);
            if (children.pagination.total_items) {
              page = (page ?? 0) + 1;
            } else if (children.pagination.next_token) {
              nextToken = children.pagination.next_token;
            }

            const pageSize = children.pagination?.page_size;
            const totalPages = Math.ceil(
              (children.pagination.total_items ?? 1) / (pageSize && pageSize > 0 ? pageSize : MinPageSize),
            );
            const usePagination = totalPages > 1;
            const loadMore = nextToken || (usePagination && (page ?? 0) < totalPages);

            if (!loadMore) {
              break;
            }
          }
          const path = (relativePath == "" ? "" : relativePath + "/") + f.name;
          await dispatch(walk(allChildren, callback, walkedSymbolicLinks, path));
        } catch (e) {
          console.warn("Failed to load children", e);
          if (e instanceof Error && e.name == "AbortError") {
            return;
          }
        }
      }
    }
  };
}

export interface FileResponseWalked extends FileResponse {
  relativePath: string;
}

export function walkAll(files: FileResponse[]): AppThunk<Promise<FileResponseWalked[]>> {
  return async (dispatch, _getState): Promise<FileResponseWalked[]> => {
    const allFiles: FileResponseWalked[] = [];
    await dispatch(
      walk(files, async (f, relativePath) =>
        allFiles.push(
          ...f.map(
            (file): FileResponseWalked => ({
              ...file,
              relativePath: (relativePath == "" ? "" : relativePath + "/") + file.name,
            }),
          ),
        ),
      ),
    );
    return allFiles;
  };
}

export function extractArchive(index: number, file: FileResponse): AppThunk {
  return async (dispatch, _getState) => {
    dispatch(closeContextMenu({ index, value: undefined }));
    dispatch(setExtractArchiveDialog({ open: true, file }));
  };
}

export function newRemoteDownload(index: number, file?: FileResponse): AppThunk {
  return async (dispatch, _getState) => {
    dispatch(closeContextMenu({ index, value: undefined }));
    dispatch(setRemoteDownloadDialog({ open: true, file }));
  };
}

export function batchGetDirectLinks(index: number, files: FileResponse[]): AppThunk {
  return async (dispatch, _getState) => {
    dispatch(closeContextMenu({ index, value: undefined }));
    const res = await longRunningTaskWithSnackbar(
      dispatch(startBatchGetDirectLinks(files)),
      "modals.generatingSourceLinks",
    );
    dispatch(setDirectLinkDialog({ open: true, res }));
  };
}

export function resetThumbnails(files: FileResponse[]): AppThunk {
  return async (dispatch, getState) => {
    const thumbConfigLoaded = getState().siteConfig.thumb.loaded;
    if (thumbConfigLoaded != ConfigLoadState.Loaded) {
      await dispatch(loadSiteConfig("thumb"));
    }

    const thumbExts = getState().siteConfig.thumb.config.thumb_exts ?? [];
    const targetFiles = files
      .filter((f) => f.type == FileType.file)
      .filter(
        (f) => f.metadata?.[Metadata.thumbDisabled] !== undefined && thumbExts.includes(fileExtension(f.name) ?? ""),
      );

    if (targetFiles.length === 0) {
      enqueueSnackbar({
        message: i18next.t("application:fileManager.noFileCanResetThumbnail"),
        preventDuplicate: true,
        variant: "warning",
        action: DefaultCloseAction,
      });
      return;
    }

    try {
      // Re-enable thumbnails by removing the disable mark, and update local metadata/cache.
      await dispatch(
        patchFileMetadata(FileManagerIndex.main, targetFiles, [
          {
            key: Metadata.thumbDisabled,
            remove: true,
          },
        ]),
      );

      dispatch(
        removeThumbCache({
          index: FileManagerIndex.main,
          value: targetFiles.map((file) => file.path),
        }),
      );
      // refresh file list
      dispatch(refreshFileList(FileManagerIndex.main));

      // 成功信息
      enqueueSnackbar({
        message: i18next.t("application:fileManager.resetThumbnailRequested"),
        variant: "success",
        action: DefaultCloseAction,
      });
    } catch (_e) {
      // Error snackbar is handled in send()
    }
  };
}

// Single file symbolic links might be invalid if original file is renamed by its owner,
// we need to refresh the symbolic links by getting the latest file list
export function refreshSingleFileSymbolicLinks(file: FileResponse): AppThunk<Promise<FileResponse>> {
  return async (dispatch, _getState) => {
    if (file.type != FileType.file || !file?.metadata?.[Metadata.share_redirect]) {
      return file;
    }
    const currentUrl = new CrUri(getFileLinkedUri(file));
    const latestList = await dispatch(
      getFileList(
        {
          uri: currentUrl.setPath("").toString(),
          page_size: 50,
        },
        false,
      ),
    );
    if (latestList.files.length != 1) {
      return file;
    }
    const latestFile = latestList.files[0];
    if (!latestFile) {
      return file;
    }

    if (latestFile.path != file?.metadata?.[Metadata.share_redirect]) {
      // File renamed, update file share_redirect
      dispatch(
        patchFileMetadata(FileManagerIndex.main, [file], [{ key: Metadata.share_redirect, value: latestFile.path }]),
      );
    }
    return latestFile;
  };
}

function startBatchGetDirectLinks(files: FileResponse[]): AppThunk<Promise<DirectLink[]>> {
  return async (dispatch, _getState): Promise<DirectLink[]> => {
    const allFiles: FileResponse[] = [];
    const currentUser = SessionManager.currentUserGroup();
    const batchLimit = currentUser?.direct_link_batch_size ?? 0;
    await dispatch(
      walk(files, async (children, relativePath) => {
        const childFiles = children.filter((f) => f.type == FileType.file);
        allFiles.push(...childFiles);
        if (allFiles.length > batchLimit) {
          enqueueSnackbar({
            message: i18next.t("modals.sourceBatchSizeExceeded", {
              limit: batchLimit,
            }),
            preventDuplicate: true,
            variant: "warning",
            action: DefaultCloseAction,
          });
          throw new Error("AbortError");
        }
      }),
    );

    // Check if there are any files to generate direct links for
    if (allFiles.length === 0) {
      enqueueSnackbar({
        message: i18next.t("modals.noFileCanGenerateSourceLink"),
        preventDuplicate: true,
        variant: "warning",
        action: DefaultCloseAction,
      });
      throw new Error("AbortError");
    }

    return await dispatch(
      getFileDirectLinks({
        uris: allFiles.map((f) => getFileLinkedUri(f)),
      }),
    );
  };
}
