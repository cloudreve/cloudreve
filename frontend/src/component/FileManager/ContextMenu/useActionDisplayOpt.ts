import { useMemo } from "react";
import { FileResponse, FileType, Metadata, NavigatorCapability } from "../../../api/explorer.ts";
import { GroupPermission } from "../../../api/user.ts";
import { defaultPath } from "../../../hooks/useNavigation.tsx";
import { ContextMenuTypes } from "../../../redux/fileManagerSlice.ts";
import { Viewers, ViewersByID } from "../../../redux/siteConfigSlice.ts";
import { ExpandedViewerSetting } from "../../../redux/thunks/viewer.ts";
import SessionManager from "../../../session";
import { fileExtension } from "../../../util";
import Boolset from "../../../util/boolset.ts";
import CrUri, { Filesystem } from "../../../util/uri.ts";
import { FileManagerIndex } from "../FileManager.tsx";

const supportedArchiveTypes = ["zip", "gz", "xz", "tar", "rar", "7z", "bz2"];

// archiveVolumeSetKey returns the volume-set key of a file name following the
// same conventions as the backend (partN.rar, .rNN), or null when the name is
// not a recognizable multi-volume member.
const archiveVolumeSetKey = (name: string): string | null => {
  const l = name.toLowerCase();
  let m = /^(.+)\.part\d+\.rar$/.exec(l);
  if (m) return `rar:${m[1]}`;
  m = /^(.+)\.r\d+$/.exec(l);
  if (m) return `rarold:${m[1]}`;
  if (l.endsWith(".rar")) return `rarold:${l.slice(0, -4)}`;
  return null;
};

// isSingleVolumeSet is true when all files are volumes of the same archive set.
export const isSingleVolumeSet = (files: FileResponse[]): boolean => {
  if (files.length < 2 || files.some((f) => f.type != FileType.file)) {
    return false;
  }
  const key = archiveVolumeSetKey(files[0].name);
  return key != null && files.every((f) => archiveVolumeSetKey(f.name) === key);
};

export const canManageVersion = (file: FileResponse, bs: Boolset) => {
  return (
    file.type == FileType.file &&
    (!file.metadata || !file.metadata[Metadata.share_redirect]) &&
    bs.enabled(NavigatorCapability.version_control)
  );
};

export const canShowInfo = (cap: Boolset) => {
  return cap.enabled(NavigatorCapability.info);
};

export const canUpdate = (opt: DisplayOption) => {
  // Overwriting an existing file requires the rename capability server-side.
  return !!(
    opt.allUpdatable &&
    opt.hasFile &&
    opt.orCapability?.enabled(NavigatorCapability.upload_file) &&
    opt.orCapability?.enabled(NavigatorCapability.rename_file)
  );
};

export interface DisplayOption {
  allReadable: boolean;
  allUpdatable: boolean;

  hasReadable?: boolean;
  hasUpdatable?: boolean;

  hasTrashFile?: boolean;
  hasFile?: boolean;
  hasFolder?: boolean;
  hasOwned?: boolean;
  hasFailedThumb?: boolean;

  showEnter?: boolean;
  showOpen?: boolean;
  showOpenWithCascading?: () => boolean;
  showOpenWith?: () => boolean;
  showDownload?: boolean;
  showGoToSharedLink?: boolean;
  showExtractArchive?: boolean;
  showTorrentRemoteDownload?: boolean;
  showGoToParent?: boolean;

  showDelete?: boolean;
  showRestore?: boolean;
  showRename?: boolean;
  showPin?: boolean;
  showOrganize?: boolean;
  showCopy?: boolean;
  showShare?: boolean;
  showInfo?: boolean;
  showDirectLink?: boolean;

  showMove?: boolean;
  showTags?: boolean;
  showChangeFolderColor?: boolean;
  showChangeIcon?: boolean;
  showCustomProps?: boolean;

  showMore?: boolean;
  showVersionControl?: boolean;
  showDirectLinkManagement?: boolean;
  showManageShares?: boolean;
  showAcl?: boolean;
  showCreateArchive?: boolean;
  showResetThumb?: boolean;

  andCapability?: Boolset;
  orCapability?: Boolset;

  showCreateFolder?: boolean;
  showCreateFile?: boolean;
  showRefresh?: boolean;
  showNewFileFromTemplate?: boolean;
  showUpload?: boolean;
  showRemoteDownload?: boolean;
}

const capabilityMap: { [key: string]: Boolset } = {};

export const getActionOpt = (
  targets: FileResponse[],
  viewerSetting?: ExpandedViewerSetting,
  type?: string,
  parent?: FileResponse,
  fmIndex: number = 0,
): DisplayOption => {
  const currentUser = SessionManager.currentLoginOrNull();
  const currentUserAnonymous = SessionManager.currentUser();
  const groupBs = SessionManager.currentUserGroupPermission();
  const display: DisplayOption = {
    allReadable: true,
    allUpdatable: true,
  };
  if (type == ContextMenuTypes.empty || type == ContextMenuTypes.new) {
    display.showRefresh = type == ContextMenuTypes.empty;
    display.showRemoteDownload = groupBs.enabled(GroupPermission.remote_download) && !!currentUser;

    if (!parent || parent.type != FileType.folder) {
      display.showRemoteDownload = display.showRemoteDownload && type == ContextMenuTypes.new;
      return display;
    }

    const parentCap = new Boolset(parent.capability);
    const parentInShare = new CrUri(parent.path).fs() == Filesystem.share;
    display.showCreateFolder =
      parentCap.enabled(NavigatorCapability.create_file) && (parent.owned || parentInShare);
    display.showCreateFile = display.showCreateFolder && fmIndex == FileManagerIndex.main;
    display.showUpload = display.showCreateFile;
    if (display.showCreateFile) {
      const allViewers = Object.entries(ViewersByID);
      for (let i = 0; i < allViewers.length; i++) {
        if (allViewers[i][1] && allViewers[i][1].templates) {
          display.showNewFileFromTemplate = true;
          break;
        }
      }
    }

    return display;
  }

  if (type == ContextMenuTypes.searchResult) {
    display.showGoToParent = true;
  }

  const parentUrl = new CrUri(targets?.[0]?.path ?? defaultPath);
  const inShare = parentUrl.fs() == Filesystem.share;
  targets.forEach((target) => {
    const readable = true;
    // Share visitors are never owners; their writable actions are gated by
    // the per-file capability boolset stamped from the share's props.
    const updatable = target.owned || inShare;

    if (display.allReadable && !readable) {
      display.allReadable = false;
    }
    if (display.allUpdatable && !updatable) {
      display.allUpdatable = false;
    }

    if (!display.hasReadable && readable) {
      display.hasReadable = true;
    }
    if (!display.hasUpdatable && updatable) {
      display.hasUpdatable = true;
    }

    if (target.metadata) {
      if (target.metadata[Metadata.restore_uri]) {
        display.hasTrashFile = true;
      }
      if (target.metadata[Metadata.thumbDisabled] !== undefined) {
        display.hasFailedThumb = true;
      }
    }

    if (target.type == FileType.file) {
      display.hasFile = true;
    }

    if (target.type == FileType.folder) {
      display.hasFolder = true;
    }

    if (target.owned) {
      display.hasOwned = true;
    }

    if (target.capability) {
      let bs = capabilityMap[target.capability];
      if (!bs) {
        bs = new Boolset(target.capability);
        capabilityMap[target.capability] = bs;
      }

      if (!display.andCapability) {
        display.andCapability = bs;
      }

      display.andCapability = display.andCapability.and(bs);

      if (!display.orCapability) {
        display.orCapability = bs;
      }
      display.orCapability = display.orCapability.or(bs);
    }
  });

  const firstFileSuffix = fileExtension(targets[0]?.name ?? "");
  display.showPin = !display.hasTrashFile && targets.length == 1 && display.hasFolder;
  display.showDelete =
    display.hasUpdatable &&
    display.orCapability &&
    (display.orCapability.enabled(NavigatorCapability.soft_delete) ||
      display.orCapability.enabled(NavigatorCapability.delete_file));
  display.showRestore = display.andCapability?.enabled(NavigatorCapability.restore);
  display.showRename =
    targets.length == 1 &&
    display.allUpdatable &&
    display.orCapability &&
    display.orCapability.enabled(NavigatorCapability.rename_file);
  display.showCopy =
    display.hasUpdatable &&
    display.orCapability &&
    display.orCapability.enabled(NavigatorCapability.download_file) &&
    (!inShare || display.orCapability.enabled(NavigatorCapability.create_file));
  display.showShare =
    targets.length == 1 &&
    !!currentUser &&
    groupBs.enabled(GroupPermission.share) &&
    display.allUpdatable &&
    (targets[0].owned || groupBs.enabled(GroupPermission.is_admin)) &&
    display.orCapability &&
    display.orCapability.enabled(NavigatorCapability.share) &&
    (!targets[0].metadata ||
      (!targets[0].metadata[Metadata.share_redirect] && !targets[0].metadata[Metadata.restore_uri]));
  display.showMove =
    display.hasUpdatable &&
    display.orCapability &&
    display.orCapability.enabled(NavigatorCapability.delete_file) &&
    (!inShare || display.orCapability.enabled(NavigatorCapability.create_file));
  display.showTags =
    display.hasUpdatable && display.orCapability && display.orCapability.enabled(NavigatorCapability.update_metadata);
  display.showChangeFolderColor =
    display.hasUpdatable &&
    !display.hasFile &&
    display.orCapability &&
    display.orCapability.enabled(NavigatorCapability.update_metadata);
  display.showChangeIcon =
    display.hasUpdatable && display.orCapability && display.orCapability.enabled(NavigatorCapability.update_metadata);
  display.showCustomProps = display.showChangeIcon;
  display.showDownload =
    display.hasReadable && display.orCapability && display.orCapability.enabled(NavigatorCapability.download_file);
  display.showDirectLink =
    (display.hasOwned || groupBs.enabled(GroupPermission.is_admin)) &&
    display.orCapability &&
    (currentUserAnonymous?.group?.direct_link_batch_size ?? 0) >= targets.length &&
    display.orCapability.enabled(NavigatorCapability.download_file);
  display.showDirectLinkManagement = display.showDirectLink && targets.length == 1 && display.hasFile;
  display.showOpen =
    targets.length == 1 &&
    display.hasFile &&
    display.showDownload &&
    !!viewerSetting &&
    !!firstFileSuffix &&
    !!viewerSetting?.[firstFileSuffix];
  display.showEnter =
    targets.length == 1 &&
    display.hasFolder &&
    display.orCapability?.enabled(NavigatorCapability.enter_folder) &&
    display.allReadable;
  display.showExtractArchive =
    display.hasFile &&
    display.showDownload &&
    !!currentUser &&
    groupBs.enabled(GroupPermission.archive_task) &&
    ((targets.length == 1 && supportedArchiveTypes.includes(firstFileSuffix ?? "")) || isSingleVolumeSet(targets));
  display.showTorrentRemoteDownload =
    targets.length == 1 &&
    display.hasFile &&
    display.showDownload &&
    !!currentUser &&
    groupBs.enabled(GroupPermission.remote_download) &&
    firstFileSuffix == "torrent";

  display.showOpenWithCascading = () => false;
  display.showOpenWith = () => targets.length == 1 && !!display.hasFile && !!display.showDownload;
  if (display.showOpen) {
    display.showOpenWithCascading = () =>
      !!(display.showOpen && viewerSetting && viewerSetting[firstFileSuffix ?? ""]?.length >= 1);
    display.showOpenWith = () =>
      !!(display.showOpen && viewerSetting && viewerSetting[firstFileSuffix ?? ""]?.length < 1);
  }
  display.showOrganize = display.showPin || display.showMove || display.showChangeFolderColor || display.showChangeIcon;
  display.showGoToSharedLink =
    targets.length == 1 && display.hasFile && targets[0].metadata && !!targets[0].metadata[Metadata.share_redirect];
  display.showInfo = targets.length == 1 && display.orCapability && canShowInfo(display.orCapability);
  display.showVersionControl =
    targets.length == 1 &&
    display.orCapability &&
    display.hasReadable &&
    canManageVersion(targets[0], display.orCapability);
  display.showManageShares =
    targets.length == 1 &&
    targets[0].shared &&
    display.orCapability &&
    !!currentUser &&
    groupBs.enabled(GroupPermission.share) &&
    display.orCapability.enabled(NavigatorCapability.share);
  display.showCreateArchive =
    display.hasReadable &&
    !!currentUser &&
    groupBs.enabled(GroupPermission.archive_task) &&
    display.orCapability &&
    display.orCapability.enabled(NavigatorCapability.download_file);
  display.showResetThumb =
    display.hasFile &&
    !display.hasFolder &&
    display.hasFailedThumb &&
    display.allUpdatable &&
    display.orCapability &&
    display.orCapability.enabled(NavigatorCapability.update_metadata);

  display.showAcl =
    targets.length == 1 &&
    !!targets[0].owned &&
    !!currentUser &&
    groupBs.enabled(GroupPermission.set_explicit_user);

  display.showMore =
    display.showVersionControl ||
    display.showManageShares ||
    display.showAcl ||
    display.showCreateArchive ||
    display.showDirectLinkManagement ||
    display.showResetThumb;
  return display;
};

const useActionDisplayOpt = (targets: FileResponse[], type?: string, parent?: FileResponse, fmIndex: number = 0) => {
  const opt = useMemo(() => {
    return getActionOpt(targets, Viewers, type, parent, fmIndex);
  }, [targets, type, parent, fmIndex]);

  return opt;
};

export default useActionDisplayOpt;
