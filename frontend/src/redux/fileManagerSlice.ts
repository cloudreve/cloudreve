import { createSlice, PayloadAction } from "@reduxjs/toolkit";
import { FileResponse, FileThumbResponse, FileType, ListResponse } from "../api/explorer.ts";
import { AppError, Response } from "../api/request.ts";
import { Capacity } from "../api/user.ts";
import { ListViewColumnSetting } from "../component/FileManager/Explorer/ListView/Column.tsx";
import SessionManager, { UserSettings } from "../session";
import { SearchParam } from "../util/uri.ts";

export const ContextMenuTypes = {
  empty: "empty",
  new: "new",
  file: "file",
  searchResult: "searchResult",
};

export interface SingleManager {
  path?: string;
  previous_path?: string;
  pure_path?: string;
  pure_path_with_category?: string;
  path_root?: string;
  path_root_with_category?: string;
  path_elements?: string[];
  search_params?: SearchParam;
  showError: boolean;
  error?: Response<any>;
  loading: boolean;
  list?: ListResponse;
  tree: {
    [key: string]: FmTreeItem;
  };
  current_fs?: string;
  capacity?: Capacity;
  selected: {
    [key: string]: FileResponse;
  };

  // View
  layout?: string;
  showThumb: boolean;
  pageSize: number;
  sortBy?: string;
  sortDirection?: string;
  multiSelectHovered: {
    [key: string]: boolean;
  };
  galleryWidth: number;

  // Context Menu
  contextMenuOpen?: boolean;
  contextMenuPos?: { x: number; y: number };
  contextMenuType?: string;
  contextMenuTargets?: {
    [key: string]: FileResponse;
  };
  contextMenuTargetFm?: number;

  // Dialogs

  // Delete file dialog
  deleteFileModalOpen?: boolean;
  deleteFileModalSelected?: FileResponse[];
  deleteFileModalPromiseId?: string;
  deleteFileModalLoading?: boolean;

  // Rename file dialog
  renameFileModalOpen?: boolean;
  renameFileModalSelected?: FileResponse;
  renameFileModalPromiseId?: string;
  renameFileModalLoading?: boolean;
  renameFileModalError?: string;

  // List view
  listViewColumns: ListViewColumnSetting[];
}

export const Layouts = {
  grid: "grid",
  list: "list",
  gallery: "gallery",
};

export interface FmTreeItem {
  file?: FileResponse;
  thumb?: ThumbCache;
  children?: string[];
}

export interface ThumbCache {
  src: string;
  expires?: string;
}

export interface FileManagerArgsBase<T> {
  index: number;
  value: T;
}

const defaultManagerValue = {
  loading: false,
  showError: false,
  tree: {},
  root_meta: {},
  selected: {},
  pageSize: SessionManager.getWithFallback(UserSettings.PageSize),
  layout: SessionManager.getWithFallback(UserSettings.Layout),
  showThumb: SessionManager.getWithFallback(UserSettings.ShowThumb),
  sortBy: SessionManager.getWithFallback(UserSettings.SortBy),
  sortDirection: SessionManager.getWithFallback(UserSettings.SortDirection),
  listViewColumns: SessionManager.getWithFallback(UserSettings.ListViewColumns),
  multiSelectHovered: {},
  galleryWidth: window.matchMedia("(max-width: 600px)")?.matches
    ? 110
    : SessionManager.getWithFallback(UserSettings.GalleryWidth),
};
const initialState: [SingleManager, SingleManager] = [defaultManagerValue, defaultManagerValue];

export const fileManagerSlice = createSlice({
  name: "fileManagerSlice",
  initialState,
  reducers: {
    setGalleryWidth: (state, action: PayloadAction<FileManagerArgsBase<number>>) => {
      state[action.payload.index].galleryWidth = action.payload.value;
    },
    setListViewColumns: (state, action: PayloadAction<ListViewColumnSetting[]>) => {
      state.forEach((_fm, index) => {
        state[index].listViewColumns = action.payload;
      });
    },
    removeThumbCache: (state, action: PayloadAction<FileManagerArgsBase<string[]>>) => {
      action.payload.value.forEach((path) => {
        state[action.payload.index].tree[path].thumb = undefined;
      });
    },
    resetFileManager: (state, action: PayloadAction<number>) => {
      state[action.payload].path = undefined;
      state[action.payload].previous_path = undefined;
      state[action.payload].pure_path = undefined;
      state[action.payload].pure_path_with_category = undefined;
      state[action.payload].search_params = undefined;
      state[action.payload].path_root = undefined;
      state[action.payload].path_root_with_category = undefined;
      state[action.payload].path_elements = undefined;
      state[action.payload].showError = false;
      state[action.payload].error = undefined;
      state[action.payload].list = undefined;
      state[action.payload].selected = {};
      state[action.payload].contextMenuOpen = false;
      state[action.payload].deleteFileModalOpen = state[action.payload].deleteFileModalOpen ? false : undefined;
      state[action.payload].renameFileModalOpen = state[action.payload].renameFileModalOpen ? false : undefined;
    },
    setFmError: (state, action: PayloadAction<FileManagerArgsBase<any>>) => {
      const e = action.payload.value;
      state[action.payload.index].showError = !!e;
      if (e instanceof AppError) {
        state[action.payload.index].error = e.ErrorResponse();
      } else if (e instanceof Error) {
        state[action.payload.index].error = {
          msg: e.message,
          code: -1,
          data: undefined,
        };
      }
    },
    setFmLoading: (state, action: PayloadAction<FileManagerArgsBase<boolean>>) => {
      state[action.payload.index].loading = action.payload.value;
    },
    applyListResponse: (state, action: PayloadAction<FileManagerArgsBase<ListResponse>>) => {
      state[action.payload.index].list = action.payload.value;
    },
    appendListResponse: (state, action: PayloadAction<FileManagerArgsBase<ListResponse>>) => {
      const newList = [...(state[action.payload.index].list?.files || []), ...action.payload.value.files];
      state[action.payload.index].list = {
        ...action.payload.value,
        files: newList,
      };
    },
    setFileList: (state, action: PayloadAction<FileManagerArgsBase<FileResponse[]>>) => {
      const fm = state[action.payload.index];
      if (fm.list) {
        fm.list.files = action.payload.value;
      }
    },
    appendTreeCache: (state, action: PayloadAction<FileManagerArgsBase<[FileResponse[], string | undefined]>>) => {
      const path = action.payload.value[1];
      const fm = state[action.payload.index];
      action.payload.value[0].forEach((file) => {
        if (!fm.tree[file.path]) {
          fm.tree[file.path] = {};
        }
        fm.tree[file.path].file = file;
      });

      if (!path) return;

      if (!fm.tree[path]) {
        fm.tree[path] = {};
      }
      fm.tree[path].children = [
        ...new Set([
          ...(fm.tree[path].children ?? []),
          ...action.payload.value[0].filter((t) => t.type == FileType.folder).map((file) => file.path),
        ]),
      ];
    },
    removeTreeCache: (state, action: PayloadAction<FileManagerArgsBase<string[]>>) => {
      const path = action.payload.value;
      const fm = state[action.payload.index];
      // Recursively delete children
      const deleteChildren = (p: string) => {
        if (fm.tree[p]?.children) {
          fm.tree[p].children?.forEach((c) => {
            deleteChildren(c);
            delete state[action.payload.index].tree[c];
          });
        }
      };
      path.forEach((p) => {
        deleteChildren(p);
        delete state[action.payload.index].tree[p];
        // Delete path from parent's child
        const parentPath = p.substring(0, p.lastIndexOf("/"));
        if (parentPath && state[action.payload.index].tree[parentPath]) {
          state[action.payload.index].tree[parentPath].children = state[action.payload.index].tree[
            parentPath
          ].children?.filter((pa) => pa != p);
        }
      });
    },
    setPathProps: (
      state,
      action: PayloadAction<
        FileManagerArgsBase<{
          path: string;
          path_elements: string[];
          path_root: string;
          current_fs: string;
          pure_path: string;
          pure_path_with_category: string;
          path_root_with_category: string;
          search_params?: SearchParam;
        }>
      >,
    ) => {
      if (state[action.payload.index].path != action.payload.value.path) {
        state[action.payload.index].previous_path = state[action.payload.index].path;
      }
      state[action.payload.index].path = action.payload.value.path;
      state[action.payload.index].path_elements = action.payload.value.path_elements;
      state[action.payload.index].path_root = action.payload.value.path_root;
      state[action.payload.index].current_fs = action.payload.value.current_fs;
      state[action.payload.index].pure_path_with_category = action.payload.value.pure_path_with_category;
      state[action.payload.index].pure_path = action.payload.value.pure_path;
      state[action.payload.index].path_root_with_category = action.payload.value.path_root_with_category;
      state[action.payload.index].search_params = action.payload.value.search_params;
    },
    setLayout: (state, action: PayloadAction<FileManagerArgsBase<string>>) => {
      state[action.payload.index].layout = action.payload.value;
    },
    setShowThumb: (state, action: PayloadAction<FileManagerArgsBase<boolean>>) => {
      state[action.payload.index].showThumb = action.payload.value;
    },
    setPageSize: (state, action: PayloadAction<FileManagerArgsBase<number>>) => {
      const index = action.payload.index;
      state[index].pageSize = action.payload.value;
      if (state[index].list?.pagination.next_token) {
        // @ts-ignore
        state[index].list.pagination.next_token = undefined;
      }
    },
    setSortOption: (state, action: PayloadAction<FileManagerArgsBase<[string, string]>>) => {
      const index = action.payload.index;
      state[index].sortBy = action.payload.value[0];
      state[index].sortDirection = action.payload.value[1];
      if (state[index].list?.pagination.next_token) {
        // @ts-ignore
        state[index].list.pagination.next_token = undefined;
      }
    },
    setThumbCache: (state, action: PayloadAction<FileManagerArgsBase<[string, FileThumbResponse]>>) => {
      const index = action.payload.index;
      const path = action.payload.value[0];
      const thumb = action.payload.value[1];
      if (!state[index].tree[path]) {
        state[index].tree[path] = {};
      }
      state[index].tree[path].thumb = {
        src: thumb?.url,
        expires: thumb?.expires,
      };
    },
    clearSessionCache: (state, action: PayloadAction<FileManagerArgsBase<any>>) => {
      const index = action.payload.index;
      state[index].tree = {};
      state[index].capacity = undefined;
    },
    setCapacity: (state, action: PayloadAction<FileManagerArgsBase<Capacity>>) => {
      const index = action.payload.index;
      state[index].capacity = action.payload.value;
    },
    setPage: (state, action: PayloadAction<FileManagerArgsBase<number>>) => {
      const s = state[action.payload.index];
      if (s.list?.pagination) {
        s.list.pagination.page = action.payload.value;
      }
    },
    addSelected: (state, action: PayloadAction<FileManagerArgsBase<FileResponse[]>>) => {
      const index = action.payload.index;
      action.payload.value.forEach((file) => {
        state[index].selected[file.path] = file;
      });
    },
    removeSelected: (state, action: PayloadAction<FileManagerArgsBase<string>>) => {
      const index = action.payload.index;
      delete state[index].selected[action.payload.value];
    },
    clearSelected: (state, action: PayloadAction<FileManagerArgsBase<any>>) => {
      const index = action.payload.index;
      state[index].selected = {};
    },
    setSelected: (state, action: PayloadAction<FileManagerArgsBase<FileResponse[]>>) => {
      const index = action.payload.index;
      state[index].selected = {};
      action.payload.value.forEach((file) => {
        state[index].selected[file.path] = file;
      });
    },
    setMultiSelectHovered: (state, action: PayloadAction<FileManagerArgsBase<[string, boolean]>>) => {
      const index = action.payload.index;
      const [path, hovered] = action.payload.value;
      if (!hovered) {
        delete state[index].multiSelectHovered[path];
      } else {
        state[index].multiSelectHovered = {};
        state[index].multiSelectHovered[path] = hovered;
      }
    },
    clearMultiSelectHovered: (state, action: PayloadAction<FileManagerArgsBase<any>>) => {
      const index = action.payload.index;
      state[index].multiSelectHovered = {};
    },
    setContextMenu: (
      state,
      action: PayloadAction<
        FileManagerArgsBase<{
          open: boolean;
          pos: { x: number; y: number };
          type: string;
          targets: { [key: string]: FileResponse } | undefined;
          fmIndex: number;
        }>
      >,
    ) => {
      const index = action.payload.index;
      const { open, pos, type, targets, fmIndex } = action.payload.value;
      state[index].contextMenuOpen = open;
      state[index].contextMenuPos = pos;
      state[index].contextMenuType = type;
      state[index].contextMenuTargets = targets;
      state[index].contextMenuTargetFm = fmIndex;
    },
    closeContextMenu: (state, action: PayloadAction<FileManagerArgsBase<any>>) => {
      const index = action.payload.index;
      state[index].contextMenuOpen = false;
    },
    setFileDeleteModal: (
      state,
      action: PayloadAction<FileManagerArgsBase<[boolean, FileResponse[] | undefined, string | undefined, boolean]>>,
    ) => {
      const index = action.payload.index;
      const [open, selected, promiseId, loading] = action.payload.value;
      state[index].deleteFileModalOpen = open;
      state[index].deleteFileModalSelected = selected;
      state[index].deleteFileModalPromiseId = promiseId;
      state[index].deleteFileModalLoading = loading;
    },
    setFileDeleteModalLoading: (state, action: PayloadAction<FileManagerArgsBase<boolean>>) => {
      const index = action.payload.index;
      state[index].deleteFileModalLoading = action.payload.value;
    },
    setRenameFileModal: (
      state,
      action: PayloadAction<
        FileManagerArgsBase<{
          open: boolean;
          selected: FileResponse | undefined;
          promiseId: string | undefined;
          loading: boolean | undefined;
          error?: string;
        }>
      >,
    ) => {
      const index = action.payload.index;
      state[index].renameFileModalOpen = action.payload.value.open;
      state[index].renameFileModalSelected = action.payload.value.selected;
      state[index].renameFileModalPromiseId = action.payload.value.promiseId;
      state[index].renameFileModalLoading = action.payload.value.loading;
      state[index].renameFileModalError = action.payload.value.error;
    },
    closeRenameFileModal: (state, action: PayloadAction<FileManagerArgsBase<any>>) => {
      const index = action.payload.index;
      state[index].renameFileModalOpen = false;
    },
    setRenameFileModalLoading: (state, action: PayloadAction<FileManagerArgsBase<boolean>>) => {
      const index = action.payload.index;
      state[index].renameFileModalLoading = action.payload.value;
    },
    setRenameFileModalError: (state, action: PayloadAction<FileManagerArgsBase<string | undefined>>) => {
      const index = action.payload.index;
      state[index].renameFileModalError = action.payload.value;
    },
    fileUpdated: (
      state,
      action: PayloadAction<
        FileManagerArgsBase<
          {
            oldPath: string;
            file: FileResponse;
            includeMetadata?: boolean;
          }[]
        >
      >,
    ) => {
      for (const fmIndex of [0, 1]) {
        const fm = state[fmIndex];
        action.payload.value.forEach((v) => {
          const file = v.file;
          const oldPath = v.oldPath;
          const pathChanged = oldPath != file.path;
          let interestFields: {
            [key: string]: any;
          } = {
            name: file.name,
            updated_at: file.updated_at,
            size: file.size,
            created_at: file.created_at,
            id: file.id,
            path: file.path,
            shared: file.shared,
            primary_entity: file.primary_entity,
          };

          if (v.includeMetadata) {
            interestFields = { ...interestFields, metadata: file.metadata };
          }

          if (fm.list) {
            const list = fm.list.files;
            const idx = list.findIndex((f) => f.path == oldPath);
            if (idx >= 0) {
              list[idx] = {
                ...list[idx],
                ...interestFields,
              };
            }
          }

          if (fm.selected[oldPath]) {
            fm.selected[file.path] = {
              ...fm.selected[oldPath],
              ...interestFields,
            };
            fm.selected[file.path].path = file.path;
            if (pathChanged) delete fm.selected[oldPath];
          }

          if (fm.contextMenuTargets?.[oldPath]) {
            fm.contextMenuTargets[file.path] = {
              ...fm.contextMenuTargets[oldPath],
              ...interestFields,
            };
            fm.contextMenuTargets[file.path].path = file.path;
            if (pathChanged) delete fm.contextMenuTargets[oldPath];
          }

          const treeCache = fm.tree[oldPath];
          if (treeCache && treeCache.file) {
            if (pathChanged) treeCache.children = [];
            treeCache.thumb = undefined;
            treeCache.file = {
              ...treeCache.file,
              ...interestFields,
            };
          }

          fm.tree[file.path] = treeCache;
          if (pathChanged) delete fm.tree[oldPath];

          // change path from parent's child
          if (pathChanged) {
            const parentPath = oldPath.substring(0, oldPath.lastIndexOf("/"));
            if (parentPath && fm.tree[parentPath]) {
              fm.tree[parentPath].children = state[fmIndex].tree[parentPath].children?.map((pa) =>
                pa == oldPath ? file.path : pa,
              );
            }
          }
        });
      }
    },
  },
});

export default fileManagerSlice.reducer;
export const {
  removeThumbCache,
  clearMultiSelectHovered,
  setGalleryWidth,
  setListViewColumns,
  resetFileManager,
  fileUpdated,
  setRenameFileModalError,
  setRenameFileModal,
  closeRenameFileModal,
  setRenameFileModalLoading,
  removeTreeCache,
  setFileList,
  setFileDeleteModalLoading,
  setFileDeleteModal,
  closeContextMenu,
  setContextMenu,
  setMultiSelectHovered,
  setSelected,
  addSelected,
  removeSelected,
  clearSelected,
  setPage,
  setCapacity,
  clearSessionCache,
  setPathProps,
  appendTreeCache,
  appendListResponse,
  applyListResponse,
  setFmError,
  setFmLoading,
  setLayout,
  setShowThumb,
  setPageSize,
  setSortOption,
  setThumbCache,
} = fileManagerSlice.actions;
