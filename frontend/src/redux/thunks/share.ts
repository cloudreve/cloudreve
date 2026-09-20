import i18next from "i18next";
import { closeSnackbar, enqueueSnackbar, SnackbarKey } from "notistack";
import {
  getFileInfo,
  getShareInfo,
  sendCreateFile,
  sendCreateShare,
  sendUpdateShare,
} from "../../api/api.ts";
import { FileResponse, Share, ShareCreateService } from "../../api/explorer.ts";
import { DefaultCloseAction, OpenReadMeAction } from "../../component/Common/Snackbar/snackbar.tsx";
import { ShareSetting } from "../../component/FileManager/Dialogs/Share/ShareSetting.tsx";
import CrUri, { Filesystem } from "../../util/uri.ts";
import { fileUpdated } from "../fileManagerSlice.ts";
import {
  addShareInfo,
  closeShareReadme,
  setManageShareDialog,
  setShareLinkDialog,
  setShareReadmeOpen,
} from "../globalStateSlice.ts";
import { AppThunk } from "../store.ts";
import { longRunningTaskWithSnackbar } from "./file.ts";

export function createOrUpdateShareLink(
  index: number,
  file: FileResponse,
  setting: ShareSetting,
  existed?: string,
  files?: FileResponse[],
): AppThunk<Promise<string>> {
  return async (dispatch, getState) => {
    const req: ShareCreateService = {
      uri: file.path,
      uris: files && files.length > 1 ? files.map((f) => f.path) : undefined,
      is_private: setting.is_private,
      password: setting.password,
      share_view: setting.share_view,
      show_readme: setting.show_readme,
      hide_readme: setting.show_readme ? setting.hide_readme : false,
      allow_upload: setting.allow_upload || setting.allow_edit,
      allow_edit: setting.allow_edit,
      preview_only: setting.preview_only,
      upload_only: setting.upload_only,
      note: setting.note?.trim() || undefined,
      price_points: setting.price_points ?? undefined,
      listed_publicly: setting.listed_publicly,
      downloads: setting.downloads && setting.downloads_val.value > 0 ? setting.downloads_val.value : undefined,
      expire: setting.expires && setting.expires_val.value > 0 ? setting.expires_val.value : undefined,
    };

    const res = await dispatch(existed ? sendUpdateShare(req, existed) : sendCreateShare(req));
    const shared = files && files.length > 1 ? files : [file];
    dispatch(
      fileUpdated({
        index,
        value: shared.map((f) => ({
          file: { ...f, shared: true },
          oldPath: f.path,
        })),
      }),
    );

    if (existed) {
      const {
        globalState: { manageShareDialogOpen, manageShareDialogFile },
      } = getState();
      if (manageShareDialogOpen && manageShareDialogFile?.path === file.path) {
        dispatch(
          setManageShareDialog({
            open: true,
            file: {
              ...manageShareDialogFile,
              extended_info: undefined,
            },
          }),
        );
      }
    }

    return res;
  };
}

interface shareInfoQueueItem {
  resolve: (value: Share | PromiseLike<Share>) => void;
  reject: (reason?: any) => void;
}

const shareInfoLoadQueue: {
  [key: string]: shareInfoQueueItem[];
} = {};

export function queueLoadShareInfo(uri: CrUri, countViews: boolean = false): AppThunk<Promise<Share>> {
  return async (dispatch, getState) => {
    const id = `${uri.id()}/${uri.password()}/${countViews}`;
    const cached = getState().globalState.shareInfo[id];
    if (cached) {
      return cached;
    }
    if (!shareInfoLoadQueue[id]) {
      shareInfoLoadQueue[id] = [];
    }

    const p = new Promise<Share>((resolve, reject) => {
      shareInfoLoadQueue[id].push({ resolve, reject });
    });

    if (shareInfoLoadQueue[id].length === 1) {
      dispatch(getShareInfo(uri.id(), uri.password(), countViews))
        .then((res) => {
          shareInfoLoadQueue[id].forEach((item) => {
            item.resolve(res);
          });
          dispatch(addShareInfo({ id, info: res }));
        })
        .catch((e) => {
          shareInfoLoadQueue[id].forEach((item) => {
            item.reject(e);
          });
        })
        .finally(() => {
          delete shareInfoLoadQueue[id];
        });
    }

    return p;
  };
}

export interface ParsedShareLink {
  id: string;
  password?: string;
}

// parseShareLink accepts "https://host/s/<id>[/password]", "/s/<id>[/password]",
// "cloudreve://<id>[:<password>]@share", or a bare share id.
export function parseShareLink(input: string): ParsedShareLink | undefined {
  const trimmed = input.trim();
  if (!trimmed) {
    return undefined;
  }
  if (trimmed.startsWith("cloudreve://")) {
    const uri = new CrUri(trimmed);
    return uri.fs() == Filesystem.share && uri.id()
      ? { id: uri.id(), password: uri.password() || undefined }
      : undefined;
  }
  const match = trimmed.match(/\/s\/([A-Za-z0-9]+)(?:\/([^/?#]+))?/);
  if (match) {
    return { id: match[1], password: match[2] ? decodeURIComponent(match[2]) : undefined };
  }
  return /^[A-Za-z0-9]+$/.test(trimmed) ? { id: trimmed } : undefined;
}

export function saveShareToMyFiles(shareInfo: Share, password?: string, name?: string): AppThunk<Promise<void>> {
  return async (dispatch) => {
    const displayName =
      name?.trim() ||
      shareInfo.name ||
      i18next.t("application:share.somebodyShare", { name: shareInfo.owner.nickname });
    const uri = new CrUri("cloudreve://" + Filesystem.my).join(displayName);
    await dispatch(
      sendCreateFile({
        uri: uri.toString(),
        type: "share",
        share_id: shareInfo.id,
        share_password: password ?? shareInfo.password,
      }),
    );
    enqueueSnackbar({
      message: i18next.t("application:share.savedToMyFiles"),
      variant: "success",
      action: DefaultCloseAction,
    });
  };
}

export function openShareEditByID(shareId: string, password?: string, singleFile?: boolean): AppThunk {
  return async (dispatch) => {
    try {
      const { share, file } = await longRunningTaskWithSnackbar(
        dispatch(getFileAndShareById(shareId, password, singleFile)),
        "application:uploader.processing",
      );
      dispatch(
        setShareLinkDialog({
          open: true,
          file: file,
          share: share,
        }),
      );
    } catch (e) {
      console.log(e);
      return;
    }
  };
}

// Priority from high to low
const supportedReadMeFiles = ["README.md", "README.txt"];

export function detectReadMe(index: number, isTablet: boolean): AppThunk<Promise<void>> {
  return async (dispatch, getState) => {
    const { files: list } = getState().fileManager[index]?.list ?? {};
    if (list) {
      // Find readme file from highest to lowest priority
      for (const readmeFile of supportedReadMeFiles) {
        const found = list.find((file) => file.name === readmeFile);
        if (found) {
          dispatch(tryOpenReadMe(found, isTablet));
          return;
        }
      }
    }

    // Not found in current file list, try to get file directly. Always
    // probe: the readme may be filtered out of the listing entirely
    // (hide_readme) or live on a page we have not fetched yet.
    const path = getState().fileManager[index]?.pure_path;
    if (path) {
      const uri = new CrUri(path);
      for (const readmeFile of supportedReadMeFiles) {
        try {
          const file = await dispatch(getFileInfo({ uri: uri.join(readmeFile).toString() }, true));
          if (file) {
            dispatch(tryOpenReadMe(file, isTablet));
            return;
          }
        } catch (e) {}
      }
    }
    dispatch(closeShareReadme());
  };
}

let snackbarId: SnackbarKey | undefined = undefined;

function tryOpenReadMe(file: FileResponse, askForConfirmation?: boolean): AppThunk<Promise<void>> {
  return async (dispatch) => {
    if (askForConfirmation) {
      dispatch(setShareReadmeOpen({ open: false, target: file }));
      if (snackbarId) {
        closeSnackbar(snackbarId);
      }
      snackbarId = enqueueSnackbar({
        message: "README.md",
        variant: "file",
        file,
        action: OpenReadMeAction(file),
      });
    } else {
      dispatch(setShareReadmeOpen({ open: true, target: file }));
    }
  };
}

function getFileAndShareById(
  shareId: string,
  password?: string,
  singleFile?: boolean,
): AppThunk<
  Promise<{
    share: Share;
    file: FileResponse;
  }>
> {
  return async (dispatch) => {
    let share: Share | undefined;
    try {
      share = await dispatch(getShareInfo(shareId, password, false, true));
    } catch (e) {
      enqueueSnackbar({
        message: i18next.t("application:share.shareNotExist"),
        variant: "error",
        action: DefaultCloseAction,
      });
      throw e;
    }

    let file: FileResponse | undefined = undefined;
    if (singleFile) {
      const root = new CrUri(share.source_uri ?? "");
      file = await dispatch(getFileInfo({ uri: root.join(share.name ?? "").toString() }));
    } else {
      file = await dispatch(getFileInfo({ uri: share.source_uri ?? "" }));
    }
    return { share, file };
  };
}
