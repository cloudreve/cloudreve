import dayjs from "dayjs";

export const ErrNames = {
  ErrNoAvailableSession: "ErrNoAvailableSession",
  ErrAccessTokenExpired: "ErrAccessTokenExpired",
  ErrRefreshTokenExpired: "ErrRefreshTokenExpired",
  ErrSessionChanged: "ErrSessionChanged",
  ErrNoSesssionSelected: "ErrNoSesssionSelected",
};

export class ErrNoAvailableSessionClass extends Error {
  constructor() {
    super("No available session");
    this.name = ErrNames.ErrNoAvailableSession;
  }
}

export class ErrAccessTokenExpired extends Error {
  constructor(expire: dayjs.Dayjs) {
    super(`Access token expired at ${expire.toISOString()}`);
    this.name = ErrNames.ErrAccessTokenExpired;
  }
}

export class ErrRefreshTokenExpired extends Error {
  constructor(expire: dayjs.Dayjs) {
    super(`Refresh token expired at ${expire.toISOString()}`);
    this.name = ErrNames.ErrRefreshTokenExpired;
  }
}

export class ErrSessionChanged extends Error {
  constructor() {
    super("Session changed, need to refresh page");
    this.name = ErrNames.ErrSessionChanged;
  }
}

export class NoSesssionSelected extends Error {
  constructor() {
    super("No session selected");
    this.name = ErrNames.ErrNoSesssionSelected;
  }
}
