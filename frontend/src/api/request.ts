import axios, { AxiosRequestConfig } from "axios";
import { enqueueSnackbar, SnackbarAction } from "notistack";
import { DefaultCloseAction, ErrorListDetailAction } from "../component/Common/Snackbar/snackbar.tsx";
import i18n from "../i18n.ts";
import { AppThunk } from "../redux/store.ts";
import { openLockConflictDialog } from "../redux/thunks/dialog.ts";
import { router } from "../router";
import SessionManager from "../session";
import { ErrNames } from "../session/errors.ts";
import { sendRefreshToken } from "./api.ts";

export interface requestOpts {
  errorSnackbarMsg: (e: Error) => string;
  bypassSnackbar?: (e: Error) => boolean;
  noCredential: boolean;
  skipBatchError?: boolean;
  skipLockConflict?: boolean;
  withHost?: boolean;
  acceptBatchPartialSuccess?: boolean;
}

export const defaultOpts: requestOpts = {
  errorSnackbarMsg: (e) => e.message,
  noCredential: false,
  skipBatchError: false,
};

export const ApiPrefix = "/api/v4";

const instance = axios.create({
  baseURL: ApiPrefix,
});
export interface AggregatedError<T> {
  [key: string]: Response<T>;
}

export interface Response<T> {
  data: T;
  code: number;
  msg: string;
  error?: string;
  correlation_id?: string;
  aggregated_error?: AggregatedError<T>;
}

export class AppError extends Error {
  public code: any;
  public cid: string | undefined = undefined;
  public aggregatedError: AggregatedError<any> | undefined = undefined;
  public rawMessage: string = "";
  public error?: string = undefined;
  public response: Response<any>;

  constructor(resp: Response<any>) {
    const message = resp.msg;
    const code = resp.code;
    super(message);

    this.response = resp;
    this.code = code;
    const error = resp.error;
    this.cid = resp.correlation_id;
    this.aggregatedError = resp.aggregated_error;
    this.rawMessage = message;
    this.error = error;

    if (i18n.exists(`errors.${code}`, { ns: "common" })) {
      this.message = i18n.t(`errors.${code}`, {
        ns: "common",
        message,
      });
    } else if (i18n.exists(`errors.${code}`, { ns: "dashboard" })) {
      this.message = i18n.t(`errors.${code}`, {
        ns: "dashboard",
        message,
      });
    } else {
      this.message = message || i18n.t("unknownError", { ns: "common" });
    }

    this.message += error && !this.message.includes(error) ? ` (${error})` : "";
    this.stack = new Error().stack;
  }

  ErrorResponse = (): Response<any> => {
    return this.response;
  };
}
//
// instance.interceptors.response.use(
//     function (response: any) {
//         response.rawData = response.data;
//         response.data = response.data.data;
//         if (
//             response.rawData.code !== undefined &&
//             response.rawData.code !== 0 &&
//             response.rawData.code !== 203
//         ) {
//             // Login expired
//             if (response.rawData.code === 401) {
//                 Auth.signout();
//                 window.location.href =
//                     "/login?redirect=" +
//                     encodeURIComponent(
//                         window.location.pathname + window.location.search
//                     );
//             }
//
//             // Non-admin
//             if (response.rawData.code === 40008) {
//                 window.location.href = "/home";
//             }
//
//             // Not binding mobile phone
//             if (response.rawData.code === 40010) {
//                 window.location.href = "/setting?modal=phone";
//             }
//             throw new AppError(
//                 response.rawData.msg,
//                 response.rawData.code,
//                 response.rawData.error
//             );
//         }
//         return response;
//     },
//     function (error) {
//         return Promise.reject(error);
//     }
// );

export type ThunkResponse<T = any> = AppThunk<Promise<T>>;

export const Code = {
  Success: 0,
  Continue: 203,
  CredentialInvalid: 40020,
  IncorrectPassword: 40069,
  LockConflict: 40073,
  StaleVersion: 40076,
  BatchOperationNotFullyCompleted: 40081,
  DomainNotLicensed: 40087,
  AnonymouseAccessDenied: 40088,
  CodeLoginRequired: 401,
  PermissionDenied: 403,
  NodeFound: 404,
};

export const CrHeaderPrefix = "X-Cr-";
export const CrHeaders = {
  context_hint: CrHeaderPrefix + "Context-Hint",
};

function getAccessToken(): AppThunk<Promise<string | undefined>> {
  return async (dispatch, _getState) => {
    try {
      const accessToken = await SessionManager.getAccessToken();
      return `Bearer ${accessToken}`;
    } catch (e) {
      if (!(e instanceof Error)) {
        throw e;
      }
      switch (e.name) {
        case ErrNames.ErrNoAvailableSession:
        case ErrNames.ErrRefreshTokenExpired:
        case ErrNames.ErrNoSesssionSelected:
          return undefined;
        case ErrNames.ErrAccessTokenExpired:
          // try to refresh token
          console.log("refresh access token");
          const newToken = await dispatch(refreshToken());
          return `Bearer ${newToken}`;
      }

      throw e;
    }
  };
}

function refreshToken(): AppThunk<Promise<string>> {
  return async (dispatch, _getState) => {
    const user = SessionManager.currentLogin();
    const token = await dispatch(sendRefreshToken({ refresh_token: user.token.refresh_token }));
    SessionManager.refreshToken(user.user.id, token);
    return token.access_token;
  };
}

let signOutLock = false;

export function send<T = any>(
  url: string,
  config?: AxiosRequestConfig,
  opts: requestOpts = defaultOpts,
): ThunkResponse<T> {
  return async (dispatch, _getState) => {
    try {
      let axiosConf: AxiosRequestConfig = {
        ...config,
        headers: {
          ...config?.headers,
        },
        url,
      };

      if (!opts.noCredential) {
        const token = await dispatch(getAccessToken());
        if (token && axiosConf.headers) {
          axiosConf.headers["Authorization"] = token;
        }
      }

      const resp = await instance.request<Response<T>>(axiosConf);

      if (resp.data.code !== undefined && resp.data.code !== Code.Success) {
        switch (resp.data.code) {
          case Code.CredentialInvalid:
          case Code.CodeLoginRequired:
            if (!signOutLock) {
              SessionManager.signOutCurrent();
              router.navigate(
                "/session?redirect=" + encodeURIComponent(window.location.pathname + window.location.search),
              );
            }
            signOutLock = true;
        }

        throw new AppError(resp.data);
      }
      return resp.data.data;
    } catch (e) {
      let partialSuccessResponse: any = undefined;
      if (e instanceof Error) {
        // Handle lock conflict error
        if (e instanceof AppError && e.code == Code.LockConflict && !opts.skipLockConflict) {
          let rejected = false;
          try {
            await dispatch(openLockConflictDialog(e.response));
          } catch (e) {
            rejected = true;
          }
          if (!rejected) {
            return await dispatch(send(url, config, opts));
          }
        }

        if (opts.bypassSnackbar && opts.bypassSnackbar(e)) {
          throw e;
        }

        let action: SnackbarAction = DefaultCloseAction;
        // Handle aggregated error
        if (e instanceof AppError && e.code == Code.BatchOperationNotFullyCompleted) {
          if (!opts.skipBatchError) {
            action = ErrorListDetailAction(e.ErrorResponse());
            if (opts.acceptBatchPartialSuccess) {
              partialSuccessResponse = e.response.data;
            }
          } else {
            const inner = e.aggregatedError?.[Object.keys(e.aggregatedError)[0]];
            e = inner?.code ? new AppError(inner) : e;
          }
        }

        if (e instanceof Error) {
          enqueueSnackbar({
            message: opts.errorSnackbarMsg(e),
            variant: "error",
            action,
          });
        }
      }

      if (partialSuccessResponse) {
        return partialSuccessResponse;
      }
      throw e;
    }
  };
}

export const isRequestAbortedError = (e: Error) => {
  return e.message == "Request aborted";
};
