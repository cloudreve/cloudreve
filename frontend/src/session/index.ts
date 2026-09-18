import dayjs from "dayjs";
import { Group, LoginResponse, Token, User } from "../api/user.ts";
import { ColumType } from "../component/FileManager/Explorer/ListView/Column.tsx";
import Boolset from "../util/boolset.ts";
import {
  ErrAccessTokenExpired,
  ErrNoAvailableSessionClass,
  ErrRefreshTokenExpired,
  NoSesssionSelected,
} from "./errors.ts";

export enum UserSettings {
  PreferredDarkMode = "preferred_dark_mode",
  DrawerWidth = "drawer_width",
  PageSize = "page_size",
  Layout = "layout",
  ShowThumb = "show_thumb",
  SortBy = "sort_by",
  SortDirection = "sort_direction",
  ListViewColumns = "list_view_columns",
  UsedCustomizedIconColors = "used_customized_icon_colors",
  UsedCustomizedTagColors = "used_customized_tag_colors",
  UsedTags = "used_tags",
  SubtitleStyle = "subtitleStyle",
  BookLocationPrefix = "book_location_",
  MusicVolume = "music_volume",
  OpenWithPrefix = "open_with_",
  ConcurrentLimit = "concurrent_limit",
  UseAvgSpeed = "use_avg_speed",
  TaskFilter = "task_filter",
  TaskSorter = "task_sorter",
  GalleryWidth = "gallery_width",
  UploadOverwrite = "upload_overwrite",
  TimeZone = "time_zone",
  TreeViewAutoExpand = "tree_view_auto_expand",
  FolderClickAction = "folder_click_action",
}

export const UserSettingsDefault: { [key: string]: any } = {
  [UserSettings.DrawerWidth]: 240,
  [UserSettings.PageSize]: 100,
  [UserSettings.Layout]: "grid",
  [UserSettings.ShowThumb]: true,
  [UserSettings.SortBy]: "created_at",
  [UserSettings.SortDirection]: "asc",
  [UserSettings.SubtitleStyle]: {},
  [UserSettings.MusicVolume]: 0.2,
  [UserSettings.ConcurrentLimit]: 5,
  [UserSettings.UseAvgSpeed]: true,
  [UserSettings.TaskFilter]: "default",
  [UserSettings.TaskSorter]: "default",
  [UserSettings.ListViewColumns]: [
    {
      type: ColumType.name,
    },
    {
      type: ColumType.size,
    },
    {
      type: ColumType.date_modified,
    },
  ],
  [UserSettings.GalleryWidth]: 220,
  [UserSettings.UploadOverwrite]: false,
  [UserSettings.TimeZone]: "Asia/Shanghai",
  [UserSettings.TreeViewAutoExpand]: true,
  [UserSettings.FolderClickAction]: "open",
};

export interface Session extends LoginResponse {
  settings: {
    [key: string]: any;
  };
  signedOut?: boolean;
}

export interface SessionState {
  current?: string;
  sessions: { [uid: string]: Session };
  anonymousSettings: { [key: string]: any };
  anonymousUser?: User;
}

const SESSION_KEY = "cloudreve_session";

export class Manager {
  private state: SessionState;
  constructor() {
    this.state = {
      sessions: {},
      anonymousSettings: {},
    };

    const cached = localStorage.getItem(SESSION_KEY);
    if (cached) {
      this.state = JSON.parse(cached);
    }

    // Get user ID `user_hint` from query, find the user and set it as current.
    // if not found, redirect to login page/session, set redirect query to current url without user_hint
    this.handleUserHint();
  }

  private handleUserHint = () => {
    const urlParams = new URLSearchParams(window.location.search);
    const userHint = urlParams.get("user_hint");

    if (!userHint) {
      return;
    }

    // Remove user_hint from URL to build clean path
    urlParams.delete("user_hint");
    const cleanPath = urlParams.toString()
      ? `${window.location.pathname}?${urlParams.toString()}${window.location.hash}`
      : `${window.location.pathname}${window.location.hash}`;

    // Check if we have a valid session for this user
    const session = this.state.sessions[userHint];
    const isValidSession = session && !session.signedOut && !dayjs(session.token.refresh_expires).isBefore(dayjs());

    if (isValidSession && this.state.current === userHint) {
      // Remove user_hint from URL without causing a page reload
      window.history.replaceState({}, "", cleanPath);
    } else {
      // No valid session found, redirect to login/session page
      // Skip if already on session page to avoid infinite redirect
      if (window.location.pathname.startsWith("/session")) {
        window.history.replaceState({}, "", cleanPath);
        return;
      }

      window.location.href = `/session?redirect=${encodeURIComponent(cleanPath)}`;
    }
  };

  public set(key: string, value: any) {
    try {
      this.currentLogin().settings[key] = value;
    } catch (e) {
      if (e instanceof ErrNoAvailableSessionClass || e instanceof NoSesssionSelected) {
        this.state.anonymousSettings[key] = value;
      }
    }
    this.syncToStore();
  }

  public getWithFallback(key: string): any {
    const res = this.get(key);
    if (res === undefined && UserSettingsDefault[key] !== undefined) {
      return UserSettingsDefault[key];
    }

    return res;
  }

  public get(key: string): any {
    try {
      return this.currentLogin().settings[key];
    } catch (e) {
      if (e instanceof ErrNoAvailableSessionClass || e instanceof NoSesssionSelected) {
        return this.state.anonymousSettings[key];
      }
    }
  }

  // Return current login user or null
  public currentLoginOrNull = (): Session | null => {
    try {
      return this.currentLogin();
    } catch (e) {
      return null;
    }
  };

  // similar to currentLoginOrNull, but anonymous user will be returned if no login user
  public currentUser = (): User | undefined => {
    const login = this.currentLoginOrNull();
    return login ? login.user : this.state.anonymousUser;
  };

  public currentUserGroup = (): Group | undefined => {
    const login = this.currentLoginOrNull();
    return login ? login.user.group : this.state.anonymousUser?.group;
  };

  public currentUserGroupPermission = (): Boolset => {
    return new Boolset(this.currentUserGroup()?.permission);
  };

  public currentLogin = (): Session => {
    if (!this.state.current || this.state.sessions[this.state.current] === undefined) {
      throw new NoSesssionSelected();
    }

    return this.state.sessions[this.state.current];
  };

  public upsert = (session: LoginResponse) => {
    const settings = this.state.sessions[session.user.id] ? this.state.sessions[session.user.id].settings : {};
    this.state.sessions[session.user.id] = { ...session, settings };
    this.state.current = session.user.id;
    this.syncToStore();
  };

  public updateUserIfExist = (u: User) => {
    if (this.state.sessions[u.id]) {
      this.state.sessions[u.id].user = u;
      this.syncToStore();
    } else if (u.anonymous) {
      this.state.anonymousUser = u;
      this.syncToStore();
    }
  };

  public getAccessToken = async (): Promise<string> => {
    const user = this.currentLogin();
    if (!user) {
      throw new ErrNoAvailableSessionClass();
    }

    // check if token is expired
    const expires = dayjs(user.token.access_expires);
    // add a 10 minutes safe margin
    const ddl = dayjs().add(10, "minute");
    if (expires.isBefore(ddl)) {
      // current access token expired, check if refresh token is expired
      const refreshExpires = dayjs(user.token.refresh_expires);
      if (refreshExpires.isBefore(ddl)) {
        // This session is no longer valid
        this.signOutCurrent();
        throw new ErrRefreshTokenExpired(refreshExpires);
      }

      throw new ErrAccessTokenExpired(expires);
    }

    return user.token.access_token;
  };

  public refreshToken(uid: string, token: Token) {
    const user = this.state.sessions[uid];
    if (!user) {
      throw new ErrNoAvailableSessionClass();
    }

    this.state.sessions[uid].token = token;
    this.syncToStore();
  }

  public signOutCurrent = () => {
    const current = this.state.current;
    this.state.current = undefined;
    if (current && this.state.sessions[current]) {
      this.state.sessions[current].signedOut = true;
      this.state.sessions[current].token = {
        access_token: "",
        refresh_token: "",
        access_expires: "",
        refresh_expires: "",
      };
    }

    this.syncToStore();
  };

  public removeSession(uid: string) {
    delete this.state.sessions[uid];
    this.syncToStore();
  }

  private syncToStore = () => {
    localStorage.setItem(SESSION_KEY, JSON.stringify(this.state));
  };
}

const SessionManager = new Manager();
export default SessionManager;
