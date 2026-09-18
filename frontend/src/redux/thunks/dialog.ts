import { ConflictDetail, FileResponse } from "../../api/explorer.ts";
import { Response } from "../../api/request.ts";
import { DeleteOption } from "../../component/FileManager/Dialogs/DeleteConfirmation.tsx";
import { setFileDeleteModal, setRenameFileModal } from "../fileManagerSlice.ts";
import {
  DialogSelectOption,
  setAggregatedErrorDialog,
  setConfirmDialog,
  setCreateNewDialog,
  setLockConflictDialog,
  setPathSelectionDialog,
  setSaveAsDialog,
  setSelectOptionDialog,
  setStaleVersionDialog,
} from "../globalStateSlice.ts";
import { AppThunk } from "../store.ts";

export const promiseId = () => new Date().getTime().toString();
export const deleteDialogPromisePool: {
  [key: string]: {
    resolve: (value: DeleteOption | PromiseLike<DeleteOption>) => void;
    reject: (reason?: any) => void;
  };
} = {};

export const generalDialogPromisePool: {
  [key: string]: {
    resolve: (value: void | PromiseLike<void>) => void;
    reject: (reason?: any) => void;
  };
} = {};

export const renameDialogPromisePool: {
  [key: string]: {
    resolve: (value: string | PromiseLike<string>) => void;
    reject: (reason?: any) => void;
  };
} = {};

export const pathSelectionDialogPromisePool: {
  [key: string]: {
    resolve: (value: string | PromiseLike<string>) => void;
    reject: (reason?: any) => void;
  };
} = {};

export const createNewDialogPromisePool: {
  [key: string]: {
    resolve: (value: FileResponse | PromiseLike<FileResponse>) => void;
    reject: (reason?: any) => void;
  };
} = {};

export const selectOptionDialogPromisePool: {
  [key: string]: {
    resolve: (value: any | PromiseLike<any>) => void;
    reject: (reason?: any) => void;
  };
} = {};

export interface StaleVersionAction {
  overwrite?: boolean;
  saveAs?: string;
}

export const staleVersionDialogPromisePool: {
  [key: string]: {
    resolve: (value: StaleVersionAction | PromiseLike<StaleVersionAction>) => void;
    reject: (reason?: any) => void;
  };
} = {};

export interface SaveAsAction {
  uri: string;
  name: string;
}

export const saveAsDialogPromisePool: {
  [key: string]: {
    resolve: (value: SaveAsAction | PromiseLike<SaveAsAction>) => void;
    reject: (reason?: any) => void;
  };
} = {};

export function deleteConfirmation(index: number, files: FileResponse[]): AppThunk<Promise<DeleteOption>> {
  return async (dispatch, _getState) => {
    const id = promiseId();
    return new Promise<DeleteOption>((resolve, reject) => {
      deleteDialogPromisePool[id] = { resolve, reject };
      dispatch(setFileDeleteModal({ index, value: [true, files, id, false] }));
    });
  };
}

export function renameForm(index: number, file: FileResponse): AppThunk<Promise<string>> {
  return async (dispatch, _getState) => {
    const id = promiseId();
    return new Promise<string>((resolve, reject) => {
      renameDialogPromisePool[id] = { resolve, reject };
      dispatch(
        setRenameFileModal({
          index,
          value: {
            open: true,
            selected: file,
            promiseId: id,
            loading: false,
          },
        }),
      );
    });
  };
}

export function showAggregatedErrorDialog(error: Response<any>): AppThunk {
  return (dispatch, getState) => {
    const tree = getState().fileManager[0].tree;
    const files: {
      [key: string]: FileResponse;
    } = {};
    Object.keys(error.aggregated_error ?? {}).forEach((path) => {
      const cached = tree[path]?.file;
      if (cached) {
        files[path] = cached;
      }
    });
    dispatch(
      setAggregatedErrorDialog({
        open: true,
        error: error,
        files: files,
      }),
    );
  };
}

export function openLockConflictDialog(error: Response<ConflictDetail[]>): AppThunk<Promise<void>> {
  return (dispatch, getState) => {
    const tree = getState().fileManager[0].tree;
    const files: {
      [key: string]: FileResponse;
    } = {};
    error.data.forEach((conflict) => {
      if (!conflict.path) {
        return;
      }

      const cached = tree[conflict.path]?.file;
      if (cached) {
        files[conflict.path] = cached;
      }
    });

    const id = promiseId();
    return new Promise<void>((resolve, reject) => {
      generalDialogPromisePool[id] = { resolve, reject };
      dispatch(
        setLockConflictDialog({
          open: true,
          error: error,
          files: files,
          promiseId: id,
        }),
      );
    });
  };
}

export function confirmOperation(message: string): AppThunk<Promise<void>> {
  return (dispatch) => {
    const id = promiseId();
    return new Promise<void>((resolve, reject) => {
      generalDialogPromisePool[id] = { resolve, reject };
      dispatch(
        setConfirmDialog({
          open: true,
          message: message,
          promiseId: id,
        }),
      );
    });
  };
}

export function selectPath(variant: string, initialPath?: string): AppThunk<Promise<string>> {
  return (dispatch) => {
    const id = promiseId();
    return new Promise<string>((resolve, reject) => {
      pathSelectionDialogPromisePool[id] = { resolve, reject };
      dispatch(
        setPathSelectionDialog({
          open: true,
          variant: variant,
          promiseId: id,
          initialPath: initialPath,
        }),
      );
    });
  };
}

export function askStaleVersionAction(uri: string): AppThunk<Promise<StaleVersionAction>> {
  return async (dispatch, _getState) => {
    const id = promiseId();
    return new Promise<StaleVersionAction>((resolve, reject) => {
      staleVersionDialogPromisePool[id] = { resolve, reject };
      dispatch(
        setStaleVersionDialog({
          open: true,
          uri,
          promiseId: id,
        }),
      );
    });
  };
}

export function askSaveAs(name: string): AppThunk<Promise<SaveAsAction>> {
  return async (dispatch, _getState) => {
    const id = promiseId();
    return new Promise<SaveAsAction>((resolve, reject) => {
      saveAsDialogPromisePool[id] = { resolve, reject };
      dispatch(
        setSaveAsDialog({
          open: true,
          name,
          promiseId: id,
        }),
      );
    });
  };
}

export function requestCreateNew(fmIndex: number, type: string, defaultName?: string): AppThunk<Promise<FileResponse>> {
  return async (dispatch, _getState) => {
    const id = promiseId();
    return new Promise<FileResponse>((resolve, reject) => {
      createNewDialogPromisePool[id] = { resolve, reject };
      dispatch(
        setCreateNewDialog({
          open: true,
          type,
          default: defaultName,
          promiseId: id,
          fmIndex,
        }),
      );
    });
  };
}

export function selectOption(options: DialogSelectOption[], title: string): AppThunk<Promise<any> | Promise<void>> {
  return async (dispatch) => {
    const id = promiseId();
    return new Promise<any>((resolve, reject) => {
      selectOptionDialogPromisePool[id] = { resolve, reject };
      dispatch(
        setSelectOptionDialog({
          open: true,
          title,
          options,
          promiseId: id,
        }),
      );
    });
  };
}
