import { createSlice, PayloadAction } from "@reduxjs/toolkit";
import {
  ConflictDetail,
  DirectLink,
  FileResponse,
  Share,
  StoragePolicy,
  Viewer,
  ViewerSession,
} from "../api/explorer.ts";
import { Response } from "../api/request.ts";
import { AppRegistration, User } from "../api/user.ts";
import { SelectType } from "../component/Uploader/core";
import SessionManager, { UserSettings } from "../session";

export interface BatchDownloadProgress {
  filesCompleted: number;
  totalFiles: number;
  totalBytes: number;
  totalExpectedBytes: number;
  currentFile: string;
  speed: number;
  isDownloading: boolean;
}

export interface DndState {
  dragging?: boolean;
  draggingWithSelected?: boolean;
}

export interface ImageViewerState extends GeneralViewerState {
  index?: number;
  exts: string[];
}

export interface ImageEditorState extends GeneralViewerState {}

export interface GeneralViewerState {
  open: boolean;
  file: FileResponse;
  version?: string;
}

export interface ViewerSelectorState extends GeneralViewerState {
  viewers: Viewer[];
  entitySize: number;
}

export interface WopiViewerState extends GeneralViewerState {
  src: string;
  session: ViewerSession;
}

export interface DrawIOViewerState extends GeneralViewerState {
  host?: string;
}

export interface CustomViewerState extends GeneralViewerState {
  url: string;
}

export interface MusicPlayerState {
  files: FileResponse[];
  startIndex: number;
  version?: string;
}

export const CreateNewDialogType = {
  folder: "folder",
  file: "file",
};

export interface UploadProgressTotal {
  processedSize: number;
  totalSize: number;
}

export interface DialogSelectOption {
  name: string;
  description: string;
  value: any;
}

export interface DesktopCallbackState {
  code: string;
  state: string;
}

export interface GlobalStateSlice {
  loading: {
    headlessFrame: boolean;
  };
  preferredTheme?: string;
  drawerOpen: boolean;
  mobileDrawerOpen?: boolean;
  darkMode?: boolean;
  drawerWidth: number;
  userInfoCache: {
    [key: string]: User;
  };

  // Others
  pinedGeneration: number;

  // Dialogs
  // Aggregated error dialog
  aggregatedErrorDialogOpen?: boolean;
  aggregatedError?: Response<any>;
  aggregatedErrorFile?: {
    [key: string]: FileResponse;
  };

  // Lock conflict dialog
  lockConflictDialogOpen?: boolean;
  lockConflictError?: Response<ConflictDetail[]>;
  lockConflictFile?: {
    [key: string]: FileResponse;
  };
  lockConflictPromiseId?: string;

  // Confirmation dialog
  confirmDialogOpen?: boolean;
  confirmDialogMessage?: string;
  confirmPromiseId?: string;

  // Pin file dialog
  pinFileDialogOpen?: boolean;
  pinFileUri?: string;

  // path selection dialog
  pathSelectDialogOpen?: boolean;
  pathSelectDialogVariant?: string;
  pathSelectAllowedFs?: string[];
  pathSelectPromiseId?: string;
  pathSelectInitialPath?: string;

  // Tags dialog
  tagsDialogOpen?: boolean;
  tagsDialogFile?: FileResponse[];

  // Change icon dialog
  changeIconDialogOpen?: boolean;
  changeIconDialogFile?: FileResponse[];

  // Share link dialog
  shareLinkDialogOpen?: boolean;
  shareLinkDialogFile?: FileResponse;
  shareLinkDialogShare?: Share;

  // Version control dialog
  versionControlDialogOpen?: boolean;
  versionControlDialogFile?: FileResponse;
  versionControlHighlight?: string;

  // Manage share link dialog
  manageShareDialogOpen?: boolean;
  manageShareDialogFile?: FileResponse;

  // Stale version action dialog
  staleVersionDialogOpen?: boolean;
  staleVersionUri?: string;
  staleVersionPromiseId?: string;

  // Save as dialog
  saveAsDialogOpen?: boolean;
  saveAsInitialName?: string;
  saveAsPromiseId?: string;

  // Create new dialog
  createNewDialogOpen?: boolean;
  createNewDialogType?: string;
  createNewDialogDefault?: string;
  createNewPromiseId?: string;
  createNewDialogFmIndex?: number;

  // Select option dialog
  selectOptionDialogOpen?: boolean;
  selectOptionDialogOptions?: DialogSelectOption[];
  selectOptionPromiseId?: string;
  selectOptionTitle?: string;

  // Batch download log dialog
  batchDownloadLogDialogOpen?: boolean;
  batchDownloadLogDialogId?: string;
  batchDownloadLogDialogLogs?: {
    [key: string]: string;
  };

  // Batch download progress tracking
  batchDownloadProgress?: {
    [key: string]: BatchDownloadProgress;
  };

  // Create archive dialog
  createArchiveDialogOpen?: boolean;
  createArchiveDialogFiles?: FileResponse[];

  // Extract archive dialog
  extractArchiveDialogOpen?: boolean;
  extractArchiveDialogFile?: FileResponse;
  extractArchiveDialogMask?: string[];
  extractArchiveDialogEncoding?: string;

  // Remote download dialog
  remoteDownloadDialogOpen?: boolean;
  remoteDownloadDialogFile?: FileResponse;

  // List view column settings dialog
  listViewColumnSettingDialogOpen?: boolean;

  // Direct Link result dialog
  directLinkDialogOpen?: boolean;
  directLinkRes?: DirectLink[];

  // Direct Link management dialog
  directLinkManagementDialogOpen?: boolean;
  directLinkManagementDialogFile?: FileResponse;
  directLinkHighlight?: string;

  // Desktop mount setup dialog
  desktopMountSetupDialogOpen?: boolean;
  desktopMountSetupState?: DesktopCallbackState;

  // DnD
  dndState: DndState;

  // Share info cache
  shareInfo: {
    [key: string]: Share;
  };

  // Sidebar
  sidebarOpen?: boolean;
  sidebarTarget?: FileResponse | string;

  // Viewers
  imageViewer?: ImageViewerState;
  imageEditor?: ImageEditorState;
  photopeaViewer?: GeneralViewerState;
  wopiViewer?: WopiViewerState;
  codeViewer?: GeneralViewerState;
  drawIOViewer?: DrawIOViewerState;
  markdownViewer?: GeneralViewerState;
  videoViewer?: GeneralViewerState;
  pdfViewer?: GeneralViewerState;
  customViewer?: CustomViewerState;
  epubViewer?: GeneralViewerState;
  musicPlayer?: MusicPlayerState;
  excalidrawViewer?: GeneralViewerState;
  archiveViewer?: GeneralViewerState;

  // Viewer selector
  viewerSelector?: ViewerSelectorState;

  // Uploader
  uploadFileSignal?: number;
  uploadFolderSignal?: number;
  uploadProgress?: UploadProgressTotal;
  uploadTaskCount?: number;
  uploadTaskListOpen?: boolean;
  uploadFromClipboardDialogOpen?: boolean;
  uploadRawFiles?: File[];
  uploadRawPromiseId?: string[];

  policyOptionCache?: StoragePolicy[];

  // Search popup
  searchPopupOpen?: boolean;

  // Advance search
  advanceSearchOpen?: boolean;
  advanceSearchBasePath?: string;
  advanceSearchInitialNameCondition?: string[];

  // Share README
  shareReadmeDetect?: number;
  shareReadmeOpen?: boolean;
  shareReadmeTarget?: FileResponse;

  // OAuth consent flow
  oauthApp?: AppRegistration;
  oauthAppLoading?: boolean;
}

let preferred_theme: string | undefined = undefined;
try {
  preferred_theme = SessionManager.currentLogin().user.preferred_theme;
} catch (e) {}

const initialState: GlobalStateSlice = {
  loading: {
    headlessFrame: false,
  },
  pinedGeneration: 0,
  darkMode: SessionManager.get(UserSettings.PreferredDarkMode),
  preferredTheme: preferred_theme,
  drawerOpen: true,
  drawerWidth: SessionManager.getWithFallback(UserSettings.DrawerWidth),
  userInfoCache: {},
  dndState: {},
  shareInfo: {},
};

export const globalStateSlice = createSlice({
  name: "globalState",
  initialState,
  reducers: {
    setDesktopMountSetupDialog: (state, action: PayloadAction<{ open: boolean; state?: DesktopCallbackState }>) => {
      state.desktopMountSetupDialogOpen = action.payload.open;
      state.desktopMountSetupState = action.payload.state;
    },
    closeDesktopMountSetupDialog: (state) => {
      state.desktopMountSetupDialogOpen = false;
      state.desktopMountSetupState = undefined;
    },
    setUploadRawFiles: (
      state,
      action: PayloadAction<{
        files: File[];
        promiseId: string[];
      }>,
    ) => {
      state.uploadRawFiles = action.payload.files ?? [];
      state.uploadRawPromiseId = action.payload.promiseId ?? [];
    },
    setOAuthApp: (state, action: PayloadAction<AppRegistration>) => {
      state.oauthApp = action.payload;
    },
    setOAuthAppLoading: (state, action: PayloadAction<boolean>) => {
      state.oauthAppLoading = action.payload;
    },
    clearOAuthApp: (state) => {
      state.oauthApp = undefined;
      state.oauthAppLoading = undefined;
    },
    setShareReadmeDetect: (state, action: PayloadAction<boolean>) => {
      state.shareReadmeDetect = action.payload ? (state.shareReadmeDetect ?? 0) + 1 : 0;
    },
    setShareReadmeOpen: (state, action: PayloadAction<{ open: boolean; target?: FileResponse }>) => {
      state.shareReadmeOpen = action.payload.open;
      state.shareReadmeTarget = action.payload.target;
    },
    closeShareReadme: (state) => {
      state.shareReadmeOpen = false;
    },
    setDirectLinkManagementDialog: (
      state,
      action: PayloadAction<{ open: boolean; file?: FileResponse; highlight?: string }>,
    ) => {
      state.directLinkManagementDialogOpen = action.payload.open;
      state.directLinkManagementDialogFile = action.payload.file;
      state.directLinkHighlight = action.payload.highlight;
    },
    closeDirectLinkManagementDialog: (state) => {
      state.directLinkManagementDialogOpen = false;
      state.directLinkManagementDialogFile = undefined;
      state.directLinkHighlight = undefined;
    },
    setMobileDrawerOpen: (state, action: PayloadAction<boolean>) => {
      state.mobileDrawerOpen = action.payload;
    },
    setDirectLinkDialog: (state, action: PayloadAction<{ open: boolean; res?: DirectLink[] }>) => {
      state.directLinkDialogOpen = action.payload.open;
      state.directLinkRes = action.payload.res;
    },
    closeDirectLinkDialog: (state) => {
      state.directLinkDialogOpen = false;
    },
    setListViewColumnSettingDialog: (state, action: PayloadAction<boolean>) => {
      state.listViewColumnSettingDialogOpen = action.payload;
    },
    setAdvanceSearch: (
      state,
      action: PayloadAction<{
        open: boolean;
        basePath?: string;
        nameCondition?: string[];
      }>,
    ) => {
      state.advanceSearchOpen = action.payload.open;
      state.advanceSearchBasePath = action.payload.basePath;
      state.advanceSearchInitialNameCondition = action.payload.nameCondition;
    },
    closeAdvanceSearch: (state) => {
      state.advanceSearchOpen = false;
    },
    setSearchPopup: (state, action: PayloadAction<boolean>) => {
      state.searchPopupOpen = action.payload;
    },
    setRemoteDownloadDialog: (state, action: PayloadAction<{ open: boolean; file?: FileResponse }>) => {
      state.remoteDownloadDialogOpen = action.payload.open;
      state.remoteDownloadDialogFile = action.payload.file;
    },
    closeRemoteDownloadDialog: (state) => {
      state.remoteDownloadDialogOpen = false;
    },
    setPolicyOptionCache: (state, action: PayloadAction<StoragePolicy[] | undefined>) => {
      state.policyOptionCache = action.payload;
    },
    resetDialogs: (state) => {
      state.aggregatedErrorDialogOpen = state.aggregatedErrorDialogOpen ? false : undefined;
      state.pathSelectDialogOpen = state.pathSelectDialogOpen ? false : undefined;
      state.tagsDialogOpen = state.tagsDialogOpen ? false : undefined;
      state.changeIconDialogOpen = state.changeIconDialogOpen ? false : undefined;
      state.shareLinkDialogOpen = state.shareLinkDialogOpen ? false : undefined;
      state.versionControlDialogOpen = state.versionControlDialogOpen ? false : undefined;
      state.manageShareDialogOpen = state.manageShareDialogOpen ? false : undefined;
      state.createNewDialogOpen = state.createNewDialogOpen ? false : undefined;
      state.selectOptionDialogOpen = state.selectOptionDialogOpen ? false : undefined;
      state.batchDownloadLogDialogOpen = state.batchDownloadLogDialogOpen ? false : undefined;
      state.createArchiveDialogOpen = state.createArchiveDialogOpen ? false : undefined;
      state.extractArchiveDialogOpen = state.extractArchiveDialogOpen ? false : undefined;

      // reset all viewers
      state.imageViewer = undefined;
      state.imageEditor = undefined;
      state.photopeaViewer = undefined;
      state.wopiViewer = undefined;
      state.codeViewer = undefined;
      state.drawIOViewer = undefined;
      state.markdownViewer = undefined;
      state.videoViewer = undefined;
      state.pdfViewer = undefined;
      state.customViewer = undefined;
      state.epubViewer = undefined;
      state.excalidrawViewer = undefined;
      state.archiveViewer = undefined;
    },
    setExtractArchiveDialog: (
      state,
      action: PayloadAction<{ open: boolean; file?: FileResponse; mask?: string[]; encoding?: string }>,
    ) => {
      state.extractArchiveDialogOpen = action.payload.open;
      state.extractArchiveDialogFile = action.payload.file;
      state.extractArchiveDialogMask = action.payload.mask;
      state.extractArchiveDialogEncoding = action.payload.encoding;
    },
    closeExtractArchiveDialog: (state) => {
      state.extractArchiveDialogOpen = false;
      state.extractArchiveDialogMask = undefined;
      state.extractArchiveDialogEncoding = undefined;
    },
    setCreateArchiveDialog: (
      state,
      action: PayloadAction<{
        open: boolean;
        files: FileResponse[];
      }>,
    ) => {
      state.createArchiveDialogOpen = action.payload.open;
      state.createArchiveDialogFiles = action.payload.files;
    },
    closeCreateArchiveDialog: (state) => {
      state.createArchiveDialogOpen = false;
    },
    setBatchDownloadLogDialog: (
      state,
      action: PayloadAction<{
        open: boolean;
        id: string;
      }>,
    ) => {
      state.batchDownloadLogDialogOpen = action.payload.open;
      state.batchDownloadLogDialogId = action.payload.id;
    },
    closeBatchDownloadLogDialog: (state) => {
      state.batchDownloadLogDialogOpen = false;
    },
    setBatchDownloadLog: (
      state,
      action: PayloadAction<{
        id: string;
        logs: string;
      }>,
    ) => {
      if (!state.batchDownloadLogDialogLogs) {
        state.batchDownloadLogDialogLogs = {};
      }
      state.batchDownloadLogDialogLogs[action.payload.id] = action.payload.logs;
    },
    setBatchDownloadProgress: (
      state,
      action: PayloadAction<{
        id: string;
        progress: Partial<BatchDownloadProgress>;
      }>,
    ) => {
      if (!state.batchDownloadProgress) {
        state.batchDownloadProgress = {};
      }
      const existing: BatchDownloadProgress = state.batchDownloadProgress[action.payload.id] ?? {
        filesCompleted: 0,
        totalFiles: 0,
        totalBytes: 0,
        totalExpectedBytes: 0,
        currentFile: "",
        speed: 0,
        isDownloading: false,
      };
      state.batchDownloadProgress[action.payload.id] = {
        ...existing,
        ...action.payload.progress,
      };
    },
    clearBatchDownloadProgress: (state, action: PayloadAction<string>) => {
      if (state.batchDownloadProgress) {
        delete state.batchDownloadProgress[action.payload];
      }
    },
    setSelectOptionDialog: (
      state,
      action: PayloadAction<{
        open: boolean;
        options?: DialogSelectOption[];
        promiseId: string;
        title?: string;
      }>,
    ) => {
      state.selectOptionDialogOpen = action.payload.open;
      state.selectOptionDialogOptions = action.payload.options;
      state.selectOptionPromiseId = action.payload.promiseId;
      state.selectOptionTitle = action.payload.title;
    },
    closeSelectOptionDialog: (state) => {
      state.selectOptionDialogOpen = false;
    },
    setUploadFromClipboardDialog: (state, action: PayloadAction<boolean>) => {
      state.uploadFromClipboardDialogOpen = action.payload;
    },
    openUploadTaskList: (state) => {
      state.uploadTaskListOpen = true;
    },
    closeUploadTaskList: (state) => {
      state.uploadTaskListOpen = false;
    },
    setUploadProgress: (state, action: PayloadAction<{ progress: UploadProgressTotal; count: number }>) => {
      state.uploadProgress = action.payload.progress;
      state.uploadTaskCount = action.payload.count;
    },
    selectForUpload(state, action: PayloadAction<{ type: SelectType }>) {
      if (action.payload.type === SelectType.File) {
        state.uploadFileSignal = (state.uploadFileSignal ?? 0) + 1;
      } else {
        state.uploadFolderSignal = (state.uploadFolderSignal ?? 0) + 1;
      }
    },
    setCreateNewDialog: (
      state,
      action: PayloadAction<{
        open: boolean;
        type?: string;
        default?: string;
        promiseId?: string;
        fmIndex: number;
      }>,
    ) => {
      state.createNewDialogOpen = action.payload.open;
      state.createNewDialogType = action.payload.type;
      state.createNewDialogDefault = action.payload.default;
      state.createNewPromiseId = action.payload.promiseId;
      state.createNewDialogFmIndex = action.payload.fmIndex;
    },
    closeCreateNewDialog: (state) => {
      state.createNewDialogOpen = false;
    },
    setMusicPlayer: (state, action: PayloadAction<MusicPlayerState>) => {
      state.musicPlayer = action.payload;
    },
    closeMusicPlayer: (state) => {
      state.musicPlayer = undefined;
    },
    setEpubViewer: (state, action: PayloadAction<GeneralViewerState>) => {
      state.epubViewer = action.payload;
    },
    closeEpubViewer: (state) => {
      state.epubViewer && (state.epubViewer.open = false);
    },
    setCustomViewer: (state, action: PayloadAction<CustomViewerState>) => {
      state.customViewer = action.payload;
    },
    closeCustomViewer: (state) => {
      state.customViewer && (state.customViewer.open = false);
    },
    setPdfViewer: (state, action: PayloadAction<GeneralViewerState>) => {
      state.pdfViewer = action.payload;
    },
    closePdfViewer: (state) => {
      state.pdfViewer && (state.pdfViewer.open = false);
    },
    setVideoViewer: (state, action: PayloadAction<GeneralViewerState>) => {
      state.videoViewer = action.payload;
    },
    closeVideoViewer: (state) => {
      state.videoViewer && (state.videoViewer.open = false);
    },
    setMarkdownViewer: (state, action: PayloadAction<GeneralViewerState>) => {
      state.markdownViewer = action.payload;
    },
    closeMarkdownViewer: (state) => {
      state.markdownViewer && (state.markdownViewer.open = false);
    },
    setExcalidrawViewer: (state, action: PayloadAction<GeneralViewerState>) => {
      state.excalidrawViewer = action.payload;
    },
    closeExcalidrawViewer: (state) => {
      state.excalidrawViewer && (state.excalidrawViewer.open = false);
    },
    setArchiveViewer: (state, action: PayloadAction<GeneralViewerState>) => {
      state.archiveViewer = action.payload;
    },
    closeArchiveViewer: (state) => {
      state.archiveViewer && (state.archiveViewer.open = false);
    },
    addShareInfo: (state, action: PayloadAction<{ info: Share; id: string }>) => {
      state.shareInfo[action.payload.id] = action.payload.info;
    },
    setHeadlessFrameLoading: (state, action: PayloadAction<boolean>) => {
      state.loading.headlessFrame = action.payload;
    },
    setPreferredTheme: (state, action: PayloadAction<string>) => {
      state.preferredTheme = action.payload;
    },
    setDrawerOpen: (state, action: PayloadAction<boolean>) => {
      state.drawerOpen = action.payload;
    },
    setDrawerWidth: (state, action: PayloadAction<number>) => {
      state.drawerWidth = action.payload;
    },
    setDarkMode: (state, action: PayloadAction<boolean | undefined>) => {
      state.darkMode = action.payload;
    },
    setUserInfoCache: (state, action: PayloadAction<[string, User]>) => {
      state.userInfoCache[action.payload[0]] = action.payload[1];
    },
    setAggregatedErrorDialog: (
      state,
      action: PayloadAction<{
        open: boolean;
        error?: Response<any>;
        files?: { [key: string]: FileResponse };
      }>,
    ) => {
      state.aggregatedErrorDialogOpen = action.payload.open;
      state.aggregatedError = action.payload.error;
      state.aggregatedErrorFile = action.payload.files;
    },
    closeAggregatedErrorDialog: (state) => {
      state.aggregatedErrorDialogOpen = false;
    },
    setLockConflictDialog: (
      state,
      action: PayloadAction<{
        open: boolean;
        error?: Response<ConflictDetail[]>;
        files?: { [key: string]: FileResponse };
        promiseId?: string;
      }>,
    ) => {
      state.lockConflictDialogOpen = action.payload.open;
      state.lockConflictError = action.payload.error;
      state.lockConflictFile = action.payload.files;
      state.lockConflictPromiseId = action.payload.promiseId;
    },
    closeLockConflictDialog: (state) => {
      state.lockConflictDialogOpen = false;
    },
    updateLockConflicts: (state, action: PayloadAction<Response<ConflictDetail[]>>) => {
      state.lockConflictError = action.payload;
    },
    setConfirmDialog: (
      state,
      action: PayloadAction<{
        open: boolean;
        message?: string;
        promiseId?: string;
      }>,
    ) => {
      state.confirmDialogOpen = action.payload.open;
      state.confirmDialogMessage = action.payload.message;
      state.confirmPromiseId = action.payload.promiseId;
    },
    closeConfirmDialog: (state) => {
      state.confirmDialogOpen = false;
    },
    increasePinedGeneration: (state) => {
      state.pinedGeneration += 1;
    },
    setPinFileDialog: (state, action: PayloadAction<{ open: boolean; uri?: string }>) => {
      state.pinFileDialogOpen = action.payload.open;
      state.pinFileUri = action.payload.uri;
    },
    closePinFileDialog: (state) => {
      state.pinFileDialogOpen = false;
    },
    setDragging: (state, action: PayloadAction<DndState>) => {
      state.dndState.dragging = action.payload.dragging;
      state.dndState.draggingWithSelected = action.payload.draggingWithSelected;
    },
    setPathSelectionDialog: (
      state,
      action: PayloadAction<{
        open: boolean;
        variant: string;
        promiseId: string;
        initialPath?: string;
      }>,
    ) => {
      state.pathSelectDialogOpen = action.payload.open;
      state.pathSelectDialogVariant = action.payload.variant;
      state.pathSelectPromiseId = action.payload.promiseId;
      state.pathSelectInitialPath = action.payload.initialPath;
    },
    closePathSelectionDialog: (state) => {
      state.pathSelectDialogOpen = false;
    },
    setTagsDialog: (state, action: PayloadAction<{ open: boolean; file?: FileResponse[] }>) => {
      state.tagsDialogOpen = action.payload.open;
      state.tagsDialogFile = action.payload.file;
    },
    closeTagsDialog: (state) => {
      state.tagsDialogOpen = false;
    },
    setChangeIconDialog: (state, action: PayloadAction<{ open: boolean; file?: FileResponse[] }>) => {
      state.changeIconDialogOpen = action.payload.open;
      state.changeIconDialogFile = action.payload.file;
    },
    closeChangeIconDialog: (state) => {
      state.changeIconDialogOpen = false;
    },
    setShareLinkDialog: (
      state,
      action: PayloadAction<{
        open: boolean;
        file?: FileResponse;
        share?: Share;
      }>,
    ) => {
      state.shareLinkDialogOpen = action.payload.open;
      state.shareLinkDialogFile = action.payload.file;
      state.shareLinkDialogShare = action.payload.share;
    },
    closeShareLinkDialog: (state) => {
      state.shareLinkDialogOpen = false;
    },
    setSidebar: (state, action: PayloadAction<{ open: boolean; target?: FileResponse | string }>) => {
      state.sidebarOpen = action.payload.open;
      state.sidebarTarget = action.payload.target;
    },
    closeSidebar: (state) => {
      state.sidebarOpen = false;
    },
    setVersionControlDialog: (
      state,
      action: PayloadAction<{
        open: boolean;
        file?: FileResponse;
        highlight?: string;
      }>,
    ) => {
      state.versionControlDialogOpen = action.payload.open;
      state.versionControlDialogFile = action.payload.file;
      state.versionControlHighlight = action.payload.highlight;
    },
    closeVersionControlDialog: (state) => {
      state.versionControlDialogOpen = false;
    },
    setManageShareDialog: (state, action: PayloadAction<{ open: boolean; file: FileResponse }>) => {
      state.manageShareDialogOpen = action.payload.open;
      state.manageShareDialogFile = action.payload.file;
    },
    closeManageShareDialog: (state) => {
      state.manageShareDialogOpen = false;
    },
    setImageViewer: (state, action: PayloadAction<ImageViewerState>) => {
      state.imageViewer = action.payload;
    },
    closeImageViewer: (state) => {
      state.imageViewer = undefined;
    },
    setImageEditor: (state, action: PayloadAction<ImageEditorState>) => {
      state.imageEditor = action.payload;
    },
    closeImageEditor: (state) => {
      state.imageEditor = undefined;
    },
    setStaleVersionDialog: (state, action: PayloadAction<{ open: boolean; promiseId: string; uri: string }>) => {
      state.staleVersionDialogOpen = action.payload.open;
      state.staleVersionPromiseId = action.payload.promiseId;
      state.staleVersionUri = action.payload.uri;
    },
    closeStaleVersionDialog: (state) => {
      state.staleVersionDialogOpen = false;
    },
    setSaveAsDialog: (
      state,
      action: PayloadAction<{
        open: boolean;
        name?: string;
        promiseId: string;
      }>,
    ) => {
      state.saveAsDialogOpen = action.payload.open;
      state.saveAsInitialName = action.payload.name;
      state.saveAsPromiseId = action.payload.promiseId;
    },
    closeSaveAsDialog: (state) => {
      state.saveAsDialogOpen = false;
    },
    setPhotopeaViewer: (state, action: PayloadAction<GeneralViewerState>) => {
      state.photopeaViewer = action.payload;
    },
    closePhotopeaViewer: (state) => {
      state.photopeaViewer && (state.photopeaViewer.open = false);
    },
    setViewerSelector: (state, action: PayloadAction<ViewerSelectorState>) => {
      state.viewerSelector = action.payload;
    },
    closeViewerSelector: (state) => {
      state.viewerSelector && (state.viewerSelector.open = false);
    },
    setWopiViewer: (state, action: PayloadAction<WopiViewerState>) => {
      state.wopiViewer = action.payload;
    },
    closeWopiViewer: (state) => {
      state.wopiViewer && (state.wopiViewer.open = false);
    },
    setCodeViewer: (state, action: PayloadAction<GeneralViewerState>) => {
      state.codeViewer = action.payload;
    },
    closeCodeViewer: (state) => {
      state.codeViewer && (state.codeViewer.open = false);
    },
    setDrawIOViewer: (state, action: PayloadAction<DrawIOViewerState>) => {
      state.drawIOViewer = action.payload;
    },
    closeDrawIOViewer: (state) => {
      state.drawIOViewer && (state.drawIOViewer.open = false);
    },
  },
});

export default globalStateSlice.reducer;
export const {
  setArchiveViewer,
  closeArchiveViewer,
  setUploadRawFiles,
  setMobileDrawerOpen,
  setDirectLinkDialog,
  closeDirectLinkDialog,
  setListViewColumnSettingDialog,
  closeAdvanceSearch,
  setAdvanceSearch,
  setRemoteDownloadDialog,
  closeRemoteDownloadDialog,
  setExtractArchiveDialog,
  closeExtractArchiveDialog,
  setCreateArchiveDialog,
  closeCreateArchiveDialog,
  setBatchDownloadLogDialog,
  closeBatchDownloadLogDialog,
  setBatchDownloadLog,
  setBatchDownloadProgress,
  clearBatchDownloadProgress,
  setSelectOptionDialog,
  closeSelectOptionDialog,
  setUploadFromClipboardDialog,
  openUploadTaskList,
  closeUploadTaskList,
  setUploadProgress,
  selectForUpload,
  setCreateNewDialog,
  closeCreateNewDialog,
  setMusicPlayer,
  closeMusicPlayer,
  setEpubViewer,
  closeEpubViewer,
  setCustomViewer,
  closeCustomViewer,
  setPdfViewer,
  closePdfViewer,
  setVideoViewer,
  closeVideoViewer,
  setMarkdownViewer,
  closeMarkdownViewer,
  setDrawIOViewer,
  closeDrawIOViewer,
  setCodeViewer,
  closeCodeViewer,
  setWopiViewer,
  closeWopiViewer,
  setViewerSelector,
  closeViewerSelector,
  setPhotopeaViewer,
  closePhotopeaViewer,
  setStaleVersionDialog,
  closeStaleVersionDialog,
  setImageViewer,
  closeImageViewer,
  setManageShareDialog,
  closeManageShareDialog,
  setVersionControlDialog,
  closeVersionControlDialog,
  closeSidebar,
  setSidebar,
  addShareInfo,
  setShareLinkDialog,
  closeShareLinkDialog,
  setChangeIconDialog,
  closeChangeIconDialog,
  setTagsDialog,
  closeTagsDialog,
  setPathSelectionDialog,
  closePathSelectionDialog,
  setDragging,
  setPinFileDialog,
  closePinFileDialog,
  increasePinedGeneration,
  setConfirmDialog,
  closeConfirmDialog,
  updateLockConflicts,
  closeLockConflictDialog,
  setLockConflictDialog,
  closeAggregatedErrorDialog,
  setAggregatedErrorDialog,
  setDarkMode,
  setDrawerOpen,
  setDrawerWidth,
  setHeadlessFrameLoading,
  setPreferredTheme,
  setUserInfoCache,
  setImageEditor,
  closeImageEditor,
  setSaveAsDialog,
  closeSaveAsDialog,
  resetDialogs,
  setPolicyOptionCache,
  setSearchPopup,
  setExcalidrawViewer,
  closeExcalidrawViewer,
  setDirectLinkManagementDialog,
  closeDirectLinkManagementDialog,
  setShareReadmeDetect,
  closeShareReadme,
  setShareReadmeOpen,
  setOAuthApp,
  setOAuthAppLoading,
  clearOAuthApp,
  setDesktopMountSetupDialog,
  closeDesktopMountSetupDialog,
} = globalStateSlice.actions;
