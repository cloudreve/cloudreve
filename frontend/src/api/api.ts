import { AxiosProgressEvent, CancelToken } from "axios";
import { EncryptedBlob } from "../component/Uploader/core/uploader/encrypt/blob.ts";
import i18n from "../i18n.ts";
import {
  ActivityEventListResponse,
  AbuseReportListResponse,
  ReportAbuseService,
  UpdateAbuseReportService,
  AdjustCreditService,
  AdminListGroupResponse,
  AdminListService,
  ListShareResponse as AdminListShareResponse,
  StoragePolicy as AdminStoragePolicy,
  CreateGiftCodeService,
  GiftCode,
  GiftCodeListResponse,
  BatchIDService,
  CleanupTaskService,
  CreateStoragePolicyCorsService,
  Entity,
  FetchWOPIDiscoveryService,
  File as FileEnt,
  FinishOauthCallbackService,
  GetOAuthClientResponse,
  GetOauthRedirectService,
  GetSettingService,
  GroupEnt,
  HomepageSummary,
  ListEntityResponse,
  ListFileResponse,
  ListNodeResponse,
  ListOAuthClientResponse,
  ListStoragePolicyResponse,
  ListTaskResponse,
  ListUserResponse,
  Node,
  OauthCredentialStatus,
  QueueMetric,
  SetSettingService,
  Share as ShareEnt,
  Sku,
  Task,
  TestNodeDownloaderService,
  TestNodeService,
  TestSMTPService,
  ThumbGeneratorTestService,
  UpsertFileService,
  UpsertGroupService,
  UpsertNodeService,
  UpsertOAuthClientService,
  UpsertStoragePolicyService,
  UpsertUserService,
  User as UserEnt,
} from "./dashboard.ts";
import {
  AclEntry,
  AclSubject,
  AclUpsertService,
  ArchiveListFilesResponse,
  ArchiveListFilesService,
  CreateFileService,
  CreateViewerSessionService,
  DeleteFileService,
  DeleteUploadSessionService,
  DirectLink,
  FileRelocateResponse,
  FileRelocateService,
  FileResponse,
  FileThumbResponse,
  FileUpdateService,
  FileURLResponse,
  FileURLService,
  GetFileInfoService,
  ListFileService,
  ListResponse,
  MoveFileService,
  MultipleUriService,
  PatchMetadataService,
  PatchTagService,
  PatchViewSyncService,
  PinFileService,
  PreferredPolicyService,
  RenameFileService,
  Share,
  ShareCreateService,
  StoragePolicyBrief,
  UnlockFileService,
  UploadCredential,
  UploadSessionRequest,
  UserTag,
  VersionControlService,
  ViewerGroup,
  ViewerSessionResponse,
} from "./explorer.ts";
import { AppError, Code, CrHeaders, defaultOpts, isRequestAbortedError, send, ThunkResponse } from "./request.ts";
import { CreateDavAccountService, DavAccount, ListDavAccountsResponse, ListDavAccountsService } from "./setting.ts";
import { ListPublicShareService, ListShareResponse, ListShareService } from "./share.ts";
import { CaptchaResponse, SiteConfig } from "./site.ts";
import {
  AppRegistration,
  Capacity,
  FinishPasskeyLoginService,
  FinishPasskeyRegistrationService,
  CreditInfo,
  CreditTxnList,
  GrantResponse,
  GrantService,
  LoginResponse,
  Passkey,
  PasskeyCredentialOption,
  PasswordLoginRequest,
  PatchUserSetting,
  PrepareLoginResponse,
  PreparePasskeyLoginResponse,
  RefreshTokenRequest,
  ResetPasswordService,
  SendResetEmailService,
  ShopSku,
  SignUpService,
  SmsBindRequest,
  SmsLoginRequest,
  SmsResetRequest,
  SmsSendCodeRequest,
  Token,
  TwoFALoginRequest,
  User,
  UserSettings,
} from "./user.ts";
import CrUri, { Filesystem } from "../util/uri.ts";
import {
  ArchiveWorkflowService,
  DownloadWorkflowService,
  ImportWorkflowService,
  ListTaskService,
  BlobAuditWorkflowService,
  RebuildFTSIndexWorkflowService,
  RelocateEntityService,
  SetDownloadFilesService,
  TaskListResponse,
  TaskProgresses,
  TaskResponse,
} from "./workflow.ts";

export function getSiteConfig(section: string): ThunkResponse<SiteConfig> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/site/config/" + section,
        {
          method: "GET",
        },
        {
          ...defaultOpts,
          bypassSnackbar: (e) => isRequestAbortedError(e),
          errorSnackbarMsg: (e) => i18n.t("errLoadingSiteConfig", { ns: "common" }) + e.message,
        },
      ),
    );
  };
}

export function sendPrepareLogin(email: string): ThunkResponse<PrepareLoginResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/session/prepare",
        {
          params: {
            email: email,
          },
          method: "GET",
        },
        {
          ...defaultOpts,
          noCredential: true,
          bypassSnackbar: (e) => e instanceof AppError && e.code == Code.NodeFound,
        },
      ),
    );
  };
}

export function getCaptcha(): ThunkResponse<CaptchaResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/site/captcha",
        {
          method: "GET",
        },
        {
          ...defaultOpts,
          noCredential: true,
          errorSnackbarMsg: (e) => i18n.t("login.captchaError", { ns: "application" }) + e.message,
        },
      ),
    );
  };
}

export function sendLogin(req: PasswordLoginRequest): ThunkResponse<LoginResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/session/token",
        {
          data: req,
          method: "POST",
        },
        {
          ...defaultOpts,
          noCredential: true,
          bypassSnackbar: (e) => e instanceof AppError && e.code == Code.Continue,
        },
      ),
    );
  };
}

export function sendSSOExchange(ticket: string): ThunkResponse<LoginResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/session/sso/exchange",
        {
          data: { ticket },
          method: "POST",
        },
        {
          ...defaultOpts,
          noCredential: true,
        },
      ),
    );
  };
}

export function send2FALogin(req: TwoFALoginRequest): ThunkResponse<LoginResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/session/token/2fa",
        {
          data: req,
          method: "POST",
        },
        {
          ...defaultOpts,
          noCredential: true,
        },
      ),
    );
  };
}

export function getUserMe(): ThunkResponse<User> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/user/me",
        {
          method: "GET",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendRefreshToken(req: RefreshTokenRequest): ThunkResponse<Token> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/session/token/refresh",
        {
          data: req,
          method: "POST",
        },
        {
          ...defaultOpts,
          bypassSnackbar: (_e) => true,
          noCredential: true,
        },
      ),
    );
  };
}

export function sendSignout(req: RefreshTokenRequest): ThunkResponse<string> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/session/token",
        {
          data: req,
          method: "DELETE",
        },
        {
          ...defaultOpts,
          noCredential: true,
        },
      ),
    );
  };
}

export function getFileList(req: ListFileService, skipSnackbar = true): ThunkResponse<ListResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/file",
        {
          params: req,
          method: "GET",
        },
        {
          ...defaultOpts,
          bypassSnackbar: (_e) => skipSnackbar,
        },
      ),
    );
  };
}

export function getFileThumb(path: string, contextHint?: string): ThunkResponse<FileThumbResponse> {
  return async (dispatch, _getState) => {
    const params: Record<string, string> = { uri: path };
    try {
      const uri = new CrUri(path);
      if (uri.fs() == Filesystem.share) {
        const ticket = getSharePurchaseTicket(uri.id());
        if (ticket) {
          params.purchase_ticket = ticket;
        }
      }
    } catch {
      // non-CrUri inputs fall through unchanged
    }
    return await dispatch(
      send(
        "/file/thumb",
        {
          params,
          method: "GET",
          headers: contextHint
            ? {
                [CrHeaders.context_hint]: contextHint,
              }
            : {},
        },
        {
          ...defaultOpts,
          bypassSnackbar: (_e) => true,
        },
      ),
    );
  };
}

export function getUserInfo(uid: string): ThunkResponse<User> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/user/info/" + uid,
        {
          method: "GET",
        },
        {
          ...defaultOpts,
          bypassSnackbar: (_e) => true,
        },
      ),
    );
  };
}

export function getUserCapacity(): ThunkResponse<Capacity> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/user/capacity",
        {
          method: "GET",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendDeleteFiles(req: DeleteFileService): ThunkResponse {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/file",
        {
          data: req,
          method: "DELETE",
        },
        {
          ...defaultOpts,
          skipBatchError: req.uris.length == 1,
        },
      ),
    );
  };
}

export function sendEmptyTrash(): ThunkResponse {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/file/trash",
        {
          method: "DELETE",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendUnlockFiles(req: UnlockFileService): ThunkResponse {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/file/lock",
        {
          data: req,
          method: "DELETE",
        },
        {
          ...defaultOpts,
          skipLockConflict: true,
        },
      ),
    );
  };
}

export function sendRenameFile(req: RenameFileService): ThunkResponse<FileResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/file/rename",
        {
          data: req,
          method: "POST",
        },
        {
          ...defaultOpts,
          bypassSnackbar: (e) => isRequestAbortedError(e),
        },
      ),
    );
  };
}

export function sendPinFile(req: PinFileService): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/file/pin",
        {
          data: req,
          method: "PUT",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendUnpinFile(req: PinFileService): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/file/pin",
        {
          data: req,
          method: "DELETE",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendMoveFile(req: MoveFileService): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/file/move",
        {
          data: req,
          method: "POST",
        },
        {
          ...defaultOpts,
          skipBatchError: req.uris.length == 1,
          // Leave conflict failures silent: the caller prompts for
          // overwrite/skip first (#3159).
          bypassSnackbar: isNameConflictBatchError,
        },
      ),
    );
  };
}

export function isNameConflictBatchError(e: Error): boolean {
  return (
    e instanceof AppError &&
    e.code == Code.BatchOperationNotFullyCompleted &&
    Object.values(e.aggregatedError ?? {}).some((r) => r.code == Code.ObjectExist)
  );
}

export function sendRestoreFile(req: DeleteFileService): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/file/restore",
        {
          data: req,
          method: "POST",
        },
        {
          ...defaultOpts,
          skipBatchError: req.uris.length == 1,
        },
      ),
    );
  };
}

export function sendMetadataPatch(req: PatchMetadataService): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/file/metadata",
        {
          data: req,
          method: "PATCH",
        },
        {
          ...defaultOpts,
          skipBatchError: req.uris.length == 1,
        },
      ),
    );
  };
}

export function getUserTags(): ThunkResponse<UserTag[]> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/file/tag",
        {
          method: "GET",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendPatchTag(req: PatchTagService): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/file/tag",
        {
          data: req,
          method: "PATCH",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendDeleteTag(name: string): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/file/tag",
        {
          data: { name },
          method: "DELETE",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getAllowedPolicies(): ThunkResponse<StoragePolicyBrief[]> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/file/policy",
        {
          method: "GET",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function setPreferredPolicy(req: PreferredPolicyService): ThunkResponse<StoragePolicyBrief | undefined> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/file/policy",
        {
          data: req,
          method: "PUT",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function relocateToPolicy(req: FileRelocateService): ThunkResponse<FileRelocateResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/file/relocate",
        {
          data: req,
          method: "POST",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getSearchUser(keyword: string): ThunkResponse<User[]> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/user/search?keyword=" + encodeURIComponent(keyword),
        {
          method: "GET",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendCreateShare(req: ShareCreateService): ThunkResponse<string> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/share",
        {
          data: req,
          method: "PUT",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendUpdateShare(req: ShareCreateService, id: string): ThunkResponse<string> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/share/" + id,
        {
          data: req,
          method: "POST",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendDeleteShare(id: string): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/share/" + id,
        {
          method: "DELETE",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendDeleteShares(ids: string[]): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/share",
        {
          method: "DELETE",
          data: { ids },
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

const shareTicketKey = (shareId: string) => `cloudreve.share_ticket.${shareId}`;

export const getSharePurchaseTicket = (shareId: string): string | undefined => {
  try {
    return localStorage.getItem(shareTicketKey(shareId)) ?? undefined;
  } catch {
    return undefined;
  }
};

export const setSharePurchaseTicket = (shareId: string, ticket: string) => {
  try {
    localStorage.setItem(shareTicketKey(shareId), ticket);
  } catch {
    // storage unavailable; resume simply won't survive a reload
  }
};

export interface SharePurchaseResponse {
  ticket: string;
}

export function purchaseShare(id: string): ThunkResponse<SharePurchaseResponse> {
  return async (dispatch, _getState) => {
    const res = await dispatch(
      send(
        "/share/purchase/" + id,
        {
          method: "POST",
        },
        {
          ...defaultOpts,
        },
      ),
    );
    if (res?.ticket) {
      setSharePurchaseTicket(id, res.ticket);
    }
    return res;
  };
}

export function getShareInfo(
  id: string,
  password?: string,
  count_views?: boolean,
  owner_extended?: boolean,
): ThunkResponse<Share> {
  return async (dispatch, _getState) => {
    let uri = "/share/info/" + id;
    const query = new URLSearchParams();
    if (password && password != "") {
      query.set("password", password);
    }
    const purchaseTicket = getSharePurchaseTicket(id);
    if (purchaseTicket) {
      query.set("purchase_ticket", purchaseTicket);
    }
    if (count_views) {
      query.set("count_views", "true");
    }
    if (owner_extended) {
      query.set("owner_extended", "true");
    }
    if (query.toString() != "") {
      uri += "?" + query.toString();
    }
    const res = await dispatch(
      send(
        uri,
        {
          method: "GET",
        },
        {
          ...defaultOpts,
          bypassSnackbar: (_e) => true,
        },
      ),
    );
    // Persist the server-issued resume ticket so entity/thumb downloads keep
    // working for buyers who purchased on another device or browser.
    if (res?.purchase_ticket) {
      setSharePurchaseTicket(id, res.purchase_ticket);
    }
    return res;
  };
}

export function sendCreateFile(req: CreateFileService): ThunkResponse<FileResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/file/create",
        {
          data: req,
          method: "POST",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getFileEntityUrl(req: FileURLService): ThunkResponse<FileURLResponse> {
  return async (dispatch, _getState) => {
    // Attach the stored purchase ticket for share URIs so paid-share
    // downloads resume without a session.
    if (!req.purchase_ticket && req.uris.length > 0) {
      try {
        const uri = new CrUri(req.uris[0]);
        if (uri.fs() == Filesystem.share) {
          const ticket = getSharePurchaseTicket(uri.id());
          if (ticket) {
            req = { ...req, purchase_ticket: ticket };
          }
        }
      } catch {
        // non-CrUri inputs fall through unchanged
      }
    }
    return await dispatch(
      send(
        "/file/url",
        {
          data: req,
          method: "POST",
        },
        {
          ...defaultOpts,
          skipBatchError: req.uris.length == 1,
        },
      ),
    );
  };
}

export function getAclEntries(uri: string): ThunkResponse<AclEntry[]> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/file/acl",
        {
          method: "GET",
          params: { uri },
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function upsertAclEntry(req: AclUpsertService): ThunkResponse<AclEntry> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/file/acl",
        {
          method: "PUT",
          data: req,
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function deleteAclEntry(uri: string, id: number): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/file/acl",
        {
          method: "DELETE",
          params: { uri, id },
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function searchAclSubjects(keyword: string): ThunkResponse<AclSubject[]> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/file/acl/subjects",
        {
          method: "GET",
          params: { keyword },
        },
        {
          ...defaultOpts,
          bypassSnackbar: () => true,
        },
      ),
    );
  };
}

export function getFileInfo(req: GetFileInfoService, skipError = false): ThunkResponse<FileResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/file/info",
        {
          method: "GET",
          params: req,
        },
        {
          ...defaultOpts,
          bypassSnackbar: () => skipError,
        },
      ),
    );
  };
}

export function setCurrentVersion(req: VersionControlService): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/file/version/current",
        {
          method: "POST",
          data: req,
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function deleteVersion(req: VersionControlService): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/file/version",
        {
          method: "DELETE",
          data: req,
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendUpdateFile(req: FileUpdateService, data: any): ThunkResponse<FileResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/file/content",
        {
          data,
          params: req,
          method: "PUT",
          headers: {
            "Content-Type": "application/octet-stream",
          },
        },
        {
          bypassSnackbar: (e) => e instanceof AppError && e.code == Code.StaleVersion,
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendCreateViewerSession(req: CreateViewerSessionService): ThunkResponse<ViewerSessionResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/file/viewerSession",
        {
          data: req,
          method: "PUT",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendCreateUploadSession(req: UploadSessionRequest): ThunkResponse<UploadCredential> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/file/upload",
        {
          data: req,
          method: "PUT",
        },
        {
          ...defaultOpts,
          bypassSnackbar: (_e) => true,
        },
      ),
    );
  };
}

export function sendUploadChunk(
  sessionID: string,
  chunk: Blob,
  index: number,
  cancel?: CancelToken,
  onProgress?: (progressEvent: AxiosProgressEvent) => void,
): ThunkResponse<UploadCredential> {
  return async (dispatch, _getState) => {
    const streaming = chunk instanceof EncryptedBlob;
    return await dispatch(
      send(
        `/file/upload/${sessionID}/${index}`,
        {
          adapter: streaming ? "fetch" : "xhr",
          data: streaming ? chunk.stream() : chunk,
          cancelToken: cancel,
          onUploadProgress: onProgress,
          method: "POST",
          headers: {
            "Content-Type": "application/octet-stream",
            ...(streaming && { "X-Expected-Entity-Length": chunk.size?.toString() ?? "0" }),
          },
        },
        {
          ...defaultOpts,
          bypassSnackbar: (_e) => true,
        },
      ),
    );
  };
}

export function sendDeleteUploadSession(req: DeleteUploadSessionService): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/file/upload`,
        {
          data: req,
          method: "DELETE",
        },
        {
          ...defaultOpts,
          bypassSnackbar: (_e) => true,
        },
      ),
    );
  };
}

export function sendS3LikeCompleteUpload(policyType: string, sessionId: string, sessionKey: string): ThunkResponse {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/callback/${policyType}/${sessionId}/${sessionKey}`,
        {
          method: "GET",
        },
        {
          ...defaultOpts,
          bypassSnackbar: (_e) => true,
        },
      ),
    );
  };
}

export function sendOneDriveCompleteUpload(sessionId: string, sessionKey: string): ThunkResponse {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/callback/onedrive/${sessionId}/${sessionKey}`,
        {
          method: "POST",
        },
        {
          ...defaultOpts,
          bypassSnackbar: (_e) => true,
        },
      ),
    );
  };
}

export function sendCreateArchive(req: ArchiveWorkflowService): ThunkResponse<TaskResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/workflow/archive",
        {
          data: req,
          method: "POST",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendExtractArchive(req: ArchiveWorkflowService): ThunkResponse<TaskResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/workflow/extract",
        {
          data: req,
          method: "POST",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getTasks(req: ListTaskService): ThunkResponse<TaskListResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/workflow",
        {
          params: req,
          method: "GET",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendCancelTask(id: string): ThunkResponse<undefined> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/workflow/${id}/cancel`,
        {
          method: "POST",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendDeleteTask(id: string): ThunkResponse<undefined> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/workflow/${id}`,
        {
          method: "DELETE",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendRetryTask(id: string): ThunkResponse<undefined> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/workflow/${id}/retry`,
        {
          method: "POST",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getTasksPhaseProgress(id: string): ThunkResponse<TaskProgresses> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/workflow/progress/" + id,
        {
          method: "GET",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendCreateRemoteDownload(req: DownloadWorkflowService): ThunkResponse<TaskResponse[]> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/workflow/download",
        {
          data: req,
          method: "POST",
        },
        {
          ...defaultOpts,
          skipBatchError: (req.src?.length ?? 0) <= 1,
        },
      ),
    );
  };
}

export function sendSetDownloadTarget(id: string, req: SetDownloadFilesService): ThunkResponse {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/workflow/download/" + id,
        {
          data: req,
          method: "PATCH",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendCancelDownloadTask(id: string): ThunkResponse {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/workflow/download/" + id,
        {
          method: "DELETE",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getShares(req: ListShareService): ThunkResponse<ListShareResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/share",
        {
          method: "GET",
          params: req,
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

// getPublicShares lists shares opted into the public directory. No auth
// required; anonymous visitors get the same response shape.
export function getPublicShares(req: ListPublicShareService): ThunkResponse<ListShareResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/share/listed",
        {
          method: "GET",
          params: req,
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getDavAccounts(req: ListDavAccountsService): ThunkResponse<ListDavAccountsResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/devices/dav",
        {
          method: "GET",
          params: req,
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendCreateDavAccounts(req: CreateDavAccountService): ThunkResponse<DavAccount> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/devices/dav",
        {
          method: "PUT",
          data: req,
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendUpdateDavAccounts(id: string, req: CreateDavAccountService): ThunkResponse<DavAccount> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/devices/dav/${id}`,
        {
          method: "PATCH",
          data: req,
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendDeleteDavAccount(id: string): ThunkResponse {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/devices/dav/${id}`,
        {
          method: "DELETE",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getFileDirectLinks(req: MultipleUriService): ThunkResponse<DirectLink[]> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/file/source",
        {
          data: req,
          method: "PUT",
        },
        {
          ...defaultOpts,
          skipBatchError: req.uris.length == 1,
          acceptBatchPartialSuccess: true,
        },
      ),
    );
  };
}

export function sendDeleteDirectLink(id: string): ThunkResponse {
  return async (dispatch, _getState) => {
    return await dispatch(send(`/file/source/${id}`, { method: "DELETE" }, { ...defaultOpts }));
  };
}

export function getUserShares(req: ListShareService, uid: string): ThunkResponse<ListShareResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/user/shares/${uid}`,
        {
          method: "GET",
          params: req,
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getUserSettings(): ThunkResponse<UserSettings> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/user/setting`,
        {
          method: "GET",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendUploadAvatar(avatar?: Blob, contentType?: string): ThunkResponse {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/user/setting/avatar`,
        {
          method: "PUT",
          data: avatar,
          headers: contentType
            ? {
                "Content-Type": contentType,
              }
            : undefined,
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getAnnouncement(): ThunkResponse<{ content?: string }> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/user/setting/announcement`,
        {
          method: "GET",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendUpdateUserSetting(settings: PatchUserSetting): ThunkResponse {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/user/setting`,
        {
          method: "PATCH",
          data: settings,
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function get2FAInitSecret(): ThunkResponse<string> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/user/setting/2fa`,
        {
          method: "GET",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function regenerate2FABackupCodes(two_fa_code: string): ThunkResponse<string[]> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/user/setting/2fa/backup`,
        {
          method: "PUT",
          data: { two_fa_code },
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendPreparePasskeyRegistration(): ThunkResponse<PasskeyCredentialOption> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/user/authn`,
        {
          method: "PUT",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendFinishPasskeyRegistration(req: FinishPasskeyRegistrationService): ThunkResponse<Passkey> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/user/authn`,
        {
          method: "POST",
          data: req,
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendDeletePasskey(id: string): ThunkResponse {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/user/authn?id=${encodeURIComponent(id)}`,
        {
          method: "DELETE",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendFinishPasskeyLogin(req: FinishPasskeyLoginService): ThunkResponse<LoginResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/session/authn`,
        {
          method: "POST",
          data: req,
        },
        {
          ...defaultOpts,
          noCredential: true,
        },
      ),
    );
  };
}

export function sendPreparePasskeyLogin(): ThunkResponse<PreparePasskeyLoginResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/session/authn`,
        {
          method: "PUT",
        },
        {
          ...defaultOpts,
          noCredential: true,
        },
      ),
    );
  };
}

export function sendSinUp(req: SignUpService): ThunkResponse<User> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/user",
        {
          data: req,
          method: "POST",
        },
        {
          ...defaultOpts,
          noCredential: true,
          bypassSnackbar: (e) => e instanceof AppError && e.code == Code.Continue,
        },
      ),
    );
  };
}

export function sendEmailActivate(id: string, sign: string): ThunkResponse<User> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/user/activate/${id}?sign=${encodeURIComponent(sign)}`,
        {
          method: "GET",
        },
        {
          ...defaultOpts,
          noCredential: true,
        },
      ),
    );
  };
}

export function sendRequestEmailChange(newEmail: string, password: string): ThunkResponse {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/user/setting/email`,
        {
          method: "POST",
          data: { new_email: newEmail, password },
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendEmailChangeActivate(id: string, sign: string): ThunkResponse {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/user/activate_email/${id}?sign=${encodeURIComponent(sign)}`,
        {
          method: "GET",
        },
        {
          ...defaultOpts,
          noCredential: true,
        },
      ),
    );
  };
}

export function sendResetEmail(req: SendResetEmailService): ThunkResponse<User> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/user/reset`,
        {
          method: "POST",
          data: req,
        },
        {
          ...defaultOpts,
          noCredential: true,
        },
      ),
    );
  };
}

export function sendSmsCode(req: SmsSendCodeRequest): ThunkResponse<null> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/session/sms/send",
        {
          method: "POST",
          data: req,
        },
        {
          ...defaultOpts,
          noCredential: true,
        },
      ),
    );
  };
}

export function sendSmsLogin(req: SmsLoginRequest): ThunkResponse<LoginResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/session/sms/login",
        {
          method: "POST",
          data: req,
        },
        {
          ...defaultOpts,
          noCredential: true,
          bypassSnackbar: (e) => e instanceof AppError && e.code == Code.Continue,
        },
      ),
    );
  };
}

export function sendSmsReset(req: SmsResetRequest): ThunkResponse<User> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/user/reset_sms",
        {
          method: "POST",
          data: req,
        },
        {
          ...defaultOpts,
          noCredential: true,
        },
      ),
    );
  };
}

export function bindPhone(req: SmsBindRequest): ThunkResponse<null> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/user/setting/phone",
        {
          method: "PUT",
          data: req,
        },
        defaultOpts,
      ),
    );
  };
}

export function unbindPhone(): ThunkResponse<null> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/user/setting/phone",
        {
          method: "DELETE",
          data: {},
        },
        defaultOpts,
      ),
    );
  };
}

export function sendReset(uid: string, req: ResetPasswordService): ThunkResponse<User> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/user/reset/${uid}`,
        {
          method: "PATCH",
          data: req,
        },
        {
          ...defaultOpts,
          noCredential: true,
        },
      ),
    );
  };
}

export function getDashboardSummary(generateCharts?: boolean): ThunkResponse<HomepageSummary> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/summary?generate=${!!generateCharts}`,
        {
          method: "GET",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getSettings(keys: GetSettingService): ThunkResponse<{
  [key: string]: string;
}> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/settings`,
        {
          method: "POST",
          data: keys,
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendSetSetting(keys: SetSettingService): ThunkResponse<{
  [key: string]: string;
}> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/settings`,
        {
          method: "PATCH",
          data: keys,
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getGroupList(args: AdminListService): ThunkResponse<AdminListGroupResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/group`,
        {
          method: "POST",
          data: args,
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getWopiDiscovery(args: FetchWOPIDiscoveryService): ThunkResponse<ViewerGroup> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/tool/wopi`,
        {
          method: "GET",
          params: args,
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendTestThumbGeneratorExecutable(args: ThumbGeneratorTestService): ThunkResponse<string> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/tool/thumbExecutable`,
        {
          method: "POST",
          data: args,
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendTestSMTP(args: TestSMTPService): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/tool/mail`,
        { method: "POST", data: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getQueueMetrics(): ThunkResponse<QueueMetric[]> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/queue/metrics`,
        { method: "GET" },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getStoragePolicyList(args: AdminListService): ThunkResponse<ListStoragePolicyResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/policy`,
        { method: "POST", data: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getStoragePolicyDetail(id: number, countEntity?: boolean): ThunkResponse<AdminStoragePolicy> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/policy/${id}`,
        { method: "GET", params: { countEntity: countEntity ? true : undefined } },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function upsertStoragePolicy(args: UpsertStoragePolicyService): ThunkResponse<AdminStoragePolicy> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/policy${args.policy.id ? `/${args.policy.id}` : ""}`,
        { method: "PUT", data: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getNodeList(args: AdminListService): ThunkResponse<ListNodeResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/node`,
        { method: "POST", data: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getNodeDetail(id: number): ThunkResponse<Node> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/node/${id}`,
        {
          method: "GET",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function upsertNode(args: UpsertNodeService): ThunkResponse<Node> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/node${args.node.id ? `/${args.node.id}` : ""}`,
        {
          method: "PUT",
          data: args,
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendClearBlobUrlCache(): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/tool/entityUrlCache`,
        { method: "DELETE" },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function createStoragePolicyCors(args: CreateStoragePolicyCorsService): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/policy/cors`,
        { method: "POST", data: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getPolicyOauthRedirectUrl(): ThunkResponse<string> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/policy/oauth/redirect`,
        { method: "GET" },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getPolicyOauthCredentialRefreshTime(id: string): ThunkResponse<OauthCredentialStatus> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/policy/oauth/status/${id}`,
        { method: "GET" },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getPolicyOauthUrl(args: GetOauthRedirectService): ThunkResponse<string> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/policy/oauth/signin`,
        { method: "POST", data: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function finishOauthCallback(args: FinishOauthCallbackService): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/policy/oauth/callback`,
        { method: "POST", data: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getOneDriveDriverRoot(id: number, url: string): ThunkResponse<string> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/policy/oauth/root/${id}`,
        { method: "GET", params: { url } },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function deleteStoragePolicy(id: number): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/policy/${id}`,
        { method: "DELETE" },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getGroupDetail(id: number, countUser?: boolean): ThunkResponse<GroupEnt> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/group/${id}`,
        { method: "GET", params: { countUser: countUser ? true : undefined } },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function upsertGroup(args: UpsertGroupService): ThunkResponse<GroupEnt> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/group${args.group.id ? `/${args.group.id}` : ""}`,
        { method: "PUT", data: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function deleteGroup(id: number): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/group/${id}`,
        { method: "DELETE" },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function deleteNode(id: number): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/node/${id}`,
        { method: "DELETE" },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function testNode(args: TestNodeService): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/node/test`,
        { method: "POST", data: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function testNodeDownloader(args: TestNodeDownloaderService): ThunkResponse<string> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/node/test/downloader`,
        { method: "POST", data: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getUserList(args: AdminListService): ThunkResponse<ListUserResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/user`,
        { method: "POST", data: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getUserDetail(id: number): ThunkResponse<UserEnt> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/user/${id}`,
        { method: "GET" },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function upsertUser(args: UpsertUserService): ThunkResponse<UserEnt> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/user${args.user.id ? `/${args.user.id}` : ""}`,
        { method: "PUT", data: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function batchDeleteUser(args: BatchIDService): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/user/batch/delete`,
        { method: "POST", data: args },
        {
          ...defaultOpts,
          skipBatchError: args.ids.length === 1,
        },
      ),
    );
  };
}

export interface BatchUserUpdateService {
  ids: number[];
  status?: "active" | "inactive" | "manual_banned";
  group_id?: number;
}

export function batchUpdateUser(args: BatchUserUpdateService): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/user/batch/update`,
        { method: "POST", data: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export interface InvitationCode {
  id: number;
  code: string;
  group_id: number;
  max_uses: number;
  used_count: number;
  expires_at?: string;
  created_at: string;
  hash_id: string;
}

export interface ListInvitationCodeResponse {
  pagination: PaginationResults;
  codes: InvitationCode[];
}

export interface ListInvitationCodeService {
  keyword?: string;
  page_size: number;
  page_token?: string;
}

export interface CreateInvitationCodeService {
  code?: string;
  group_id?: number;
  max_uses?: number;
  expires_at?: string;
}

export function listInvitationCodes(args: ListInvitationCodeService): ThunkResponse<ListInvitationCodeResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/user/invitation/list`,
        { method: "POST", data: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function createInvitationCode(args: CreateInvitationCodeService): ThunkResponse<InvitationCode> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/user/invitation`,
        { method: "PUT", data: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function deleteInvitationCode(id: number): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/user/invitation/${id}`,
        { method: "DELETE" },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getFlattenFileList(args: AdminListService): ThunkResponse<ListFileResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/file`,
        { method: "POST", data: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getFileDetail(id: number): ThunkResponse<FileEnt> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/file/${id}`,
        { method: "GET" },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function upsertFile(args: UpsertFileService): ThunkResponse<FileEnt> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/file${args.file.id ? `/${args.file.id}` : ""}`,
        { method: "PUT", data: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getFileUrl(id: number): ThunkResponse<string> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/file/url/${id}`,
        { method: "GET" },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function batchDeleteFiles(args: BatchIDService): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/file/batch/delete`,
        { method: "POST", data: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getEntityList(args: AdminListService): ThunkResponse<ListEntityResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/entity`,
        { method: "POST", data: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getEntityDetail(id: number): ThunkResponse<Entity> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/entity/${id}`,
        { method: "GET" },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getEntityUrl(id: number): ThunkResponse<string> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/entity/url/${id}`,
        { method: "GET" },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function batchDeleteEntities(args: BatchIDService): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/entity/batch/delete`,
        { method: "POST", data: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getTaskList(args: AdminListService): ThunkResponse<ListTaskResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/queue`,
        { method: "POST", data: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getTaskDetail(id: number): ThunkResponse<Task> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/queue/${id}`,
        { method: "GET" },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function batchDeleteTasks(args: BatchIDService): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/queue/batch/delete`,
        { method: "POST", data: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getShareList(args: AdminListService): ThunkResponse<AdminListShareResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/share`,
        { method: "POST", data: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getShareDetail(id: number): ThunkResponse<ShareEnt> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/share/${id}`,
        { method: "GET" },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function batchDeleteShares(args: BatchIDService): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/share/batch/delete`,
        { method: "POST", data: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendCalibrateUserStorage(id: number): ThunkResponse<UserEnt> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/user/${id}/calibrate`,
        { method: "POST" },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendImport(req: ImportWorkflowService): ThunkResponse<TaskResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/workflow/import",
        {
          data: req,
          method: "POST",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendPatchViewSync(args: PatchViewSyncService): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/file/view`,
        { method: "PATCH", data: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendCleanupTask(args: CleanupTaskService): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/queue/cleanup`,
        { method: "POST", data: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getArchiveListFiles(args: ArchiveListFilesService): ThunkResponse<ArchiveListFilesResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/file/archive`,
        { method: "GET", params: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getOauthAppRegistration(app_id: string): ThunkResponse<AppRegistration> {
  return async (dispatch, _getState) => {
    return await dispatch(send(`/session/oauth/app/${app_id}`, { method: "GET" }, { ...defaultOpts }));
  };
}

export function sendConsentOauthApp(args: GrantService): ThunkResponse<GrantResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(`/session/oauth/consent`, { method: "POST", data: args }, { bypassSnackbar: (e) => true, ...defaultOpts }),
    );
  };
}

export function getOAuthClientList(args: AdminListService): ThunkResponse<ListOAuthClientResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/oauthClient`,
        { method: "POST", data: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getOAuthClientDetail(id: number): ThunkResponse<GetOAuthClientResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/oauthClient/${id}`,
        { method: "GET" },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function upsertOAuthClient(args: UpsertOAuthClientService): ThunkResponse<GetOAuthClientResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/oauthClient${args.client.id ? `/${args.client.id}` : ""}`,
        { method: "PUT", data: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function deleteOAuthClient(id: number): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/oauthClient/${id}`,
        { method: "DELETE" },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function batchDeleteOAuthClients(args: BatchIDService): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/oauthClient/batch/delete`,
        { method: "POST", data: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendRevokeOAuthGrant(grant_id: string): ThunkResponse {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/session/oauth/grant/${grant_id}`,
        {
          method: "DELETE",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendUnbindSso(provider: string): ThunkResponse {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/user/setting/sso_binding/${provider}`,
        {
          method: "DELETE",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendVaultSetup(password: string): ThunkResponse {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/user/vault",
        {
          method: "POST",
          data: { password },
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendVaultUnlock(password: string): ThunkResponse {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/user/vault/unlock",
        {
          method: "PUT",
          data: { password },
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendVaultLock(): ThunkResponse {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/user/vault/unlock",
        {
          method: "DELETE",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendVaultDisable(password: string): ThunkResponse {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/user/vault",
        {
          method: "DELETE",
          data: { password },
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendFullTextSearch(query: string, offset?: number): ThunkResponse {
  const params = new URLSearchParams();
  params.set("query", query);
  if (offset) {
    params.set("offset", offset.toString());
  }
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/file/search`,
        {
          method: "GET",
          params,
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendBlobAuditTask(req: BlobAuditWorkflowService): ThunkResponse<TaskResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/workflow/blobAudit",
        {
          data: req,
          method: "POST",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function relocateEntities(req: RelocateEntityService): ThunkResponse<TaskResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/admin/entity/relocate",
        {
          data: req,
          method: "POST",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendRebuildFTSIndex(req: RebuildFTSIndexWorkflowService): ThunkResponse<TaskResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/workflow/rebuildFtsIndex",
        {
          data: req,
          method: "POST",
        },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getCredit(): ThunkResponse<CreditInfo> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/user/credit",
        { method: "GET" },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getCreditTxns(page: number, pageSize: number): ThunkResponse<CreditTxnList> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/user/credit/txns?page=${page}&page_size=${pageSize}`,
        { method: "GET" },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function redeemGiftCode(code: string): ThunkResponse<GiftCode> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/user/credit/redeem",
        { method: "POST", data: { code } },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function adminListGiftCodes(page: number, pageSize: number): ThunkResponse<GiftCodeListResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/vas/giftcode?page=${page}&page_size=${pageSize}`,
        { method: "GET" },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function adminCreateGiftCode(args: CreateGiftCodeService): ThunkResponse<GiftCode[]> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/admin/vas/giftcode",
        { method: "PUT", data: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function adminDeleteGiftCode(id: number): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/vas/giftcode/${id}`,
        { method: "DELETE" },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getShopSkus(): ThunkResponse<ShopSku[]> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/user/shop/skus",
        { method: "GET" },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function purchaseSku(sku: string): ThunkResponse<CreditInfo> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/user/shop/purchase",
        { method: "POST", data: { sku } },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function adminListSkus(): ThunkResponse<Sku[]> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/admin/vas/sku",
        { method: "GET" },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function adminUpsertSku(sku: Sku): ThunkResponse<Sku> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        sku.id > 0 ? `/admin/vas/sku/${sku.id}` : "/admin/vas/sku",
        { method: "PUT", data: { sku } },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function adminDeleteSku(id: number): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/vas/sku/${id}`,
        { method: "DELETE" },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function adminListEvents(args: {
  page: number;
  pageSize: number;
  type?: number;
  actor_id?: string;
  file_id?: string;
  share_id?: string;
}): ThunkResponse<ActivityEventListResponse> {
  return async (dispatch, _getState) => {
    const params = new URLSearchParams({ page: String(args.page), page_size: String(args.pageSize) });
    if (args.type !== undefined && args.type > 0) {
      params.set("type", String(args.type));
    }
    if (args.actor_id) {
      params.set("actor_id", args.actor_id);
    }
    if (args.file_id) {
      params.set("file_id", args.file_id);
    }
    if (args.share_id) {
      params.set("share_id", args.share_id);
    }
    return await dispatch(
      send(
        `/admin/event?${params.toString()}`,
        { method: "GET" },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function getFileActivity(uri: string, page: number, pageSize: number): ThunkResponse<ActivityEventListResponse> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/file/activity?uri=${encodeURIComponent(uri)}&page=${page}&page_size=${pageSize}`,
        { method: "GET" },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function sendReportAbuse(args: ReportAbuseService): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/abuse/report",
        { method: "POST", data: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function adminListAbuseReports(args: {
  page: number;
  pageSize: number;
  status?: string;
}): ThunkResponse<AbuseReportListResponse> {
  return async (dispatch, _getState) => {
    const params = new URLSearchParams({ page: String(args.page), page_size: String(args.pageSize) });
    if (args.status) {
      params.set("status", args.status);
    }
    return await dispatch(
      send(
        `/admin/abuse?${params.toString()}`,
        { method: "GET" },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function adminUpdateAbuseReport(id: string, args: UpdateAbuseReportService): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        `/admin/abuse/${id}`,
        { method: "PATCH", data: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}

export function adminAdjustCredit(args: AdjustCreditService): ThunkResponse<void> {
  return async (dispatch, _getState) => {
    return await dispatch(
      send(
        "/admin/vas/credit",
        { method: "POST", data: args },
        {
          ...defaultOpts,
        },
      ),
    );
  };
}
