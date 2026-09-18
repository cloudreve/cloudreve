/**
 * UserLoginService 管理用户登录的服务
 */
export interface UserLoginService {
  email: string;
  password: string;
  otp?: string;
}

/**
 * User 用户序列化器
 */
export interface User {
  id: string;
  email?: string;
  nickname: string;
  status?: any /* user.Status */;
  avatar?: string;
  created_at: any /* time.Time */;
  preferred_theme?: string;
  anonymous?: boolean;
  group?: Group;
  pined?: PinedFile[];
  language?: string;
  disable_view_sync?: boolean;
  share_links_in_profile?: ShareLinksInProfileLevel;
}
export interface Group {
  id: string;
  name: string;
  permission?: string;
  direct_link_batch_size?: number;
  trash_retention?: number;
}

export interface PinedFile {
  uri: string;
  name?: string;
}

export interface PrepareLoginResponse {
  webauthn_enabled: boolean;
  sso_enabled: boolean;
  password_enabled: boolean;
  qq_enabled: boolean;
}

export interface CaptchaRequest {
  [key: string]: any;
}

export interface PasswordLoginRequest extends CaptchaRequest {
  email: string;
  password: string;
}

export interface Token {
  access_token: string;
  refresh_token: string;
  access_expires: string;
  refresh_expires: string;
}

export interface LoginResponse {
  user: User;
  token: Token;
}

export interface TwoFALoginRequest {
  otp: string;
  session_id: string;
}

export interface RefreshTokenRequest {
  refresh_token: string;
}

export interface Capacity {
  total: number;
  used: number;
}

export const GroupPermission = {
  is_admin: 0,
  is_anonymous: 1,
  share: 2,
  webdav: 3,
  archive_download: 4,
  archive_task: 5,
  webdav_proxy: 6,
  share_download: 7,
  remote_download: 9,
  redirected_source: 11,
  advance_delete: 12,
  unique_direct_link: 17,
};

export interface UserSettings {
  version_retention_enabled: boolean;
  version_retention_ext?: string[];
  version_retention_max?: number;
  passwordless: boolean;
  two_fa_enabled: boolean;
  passkeys?: Passkey[];
  disable_view_sync: boolean;
  share_links_in_profile: ShareLinksInProfileLevel;
  oauth_grants?: OAuthGrant[];
}

export interface OAuthGrant {
  client_id: string;
  client_name: string;
  client_logo: string;
  scopes?: string[];
  last_used_at?: string;
}

export interface PatchUserSetting {
  nick?: string;
  language?: string;
  preferred_theme?: string;
  version_retention_enabled?: boolean;
  version_retention_ext?: string[];
  version_retention_max?: number;
  current_password?: string;
  new_password?: string;
  two_fa_enabled?: boolean;
  two_fa_code?: string;
  disable_view_sync?: boolean;
  share_links_in_profile?: ShareLinksInProfileLevel;
}

export interface PasskeyCredentialOption {
  publicKey: {
    rp: {
      name: string;
      id: string;
    };
    user: {
      name: string;
      displayName: string;
      id: string;
    };
    challenge: string;
    pubKeyCredParams: {
      type: "public-key";
      alg: number;
    }[];
    timeout: number;
    excludeCredentials: {
      type: "public-key";
      id: string;
    }[];
    authenticatorSelection: {
      requireResidentKey: boolean;
      userVerification: UserVerificationRequirement;
    };
  };
}

export interface PasskeyCredentialLoginOption {
  publicKey: {
    challenge: string;
    timeout: number;
    rpId: string;
  };
}

export interface PreparePasskeyLoginResponse {
  options: PasskeyCredentialLoginOption;
  session_id: string;
}

export interface FinishPasskeyRegistrationService {
  response: string;
  name: string;
  ua: string;
}

export interface Passkey {
  id: string;
  name: string;
  created_at: string;
  used_at: string;
}

export interface FinishPasskeyLoginService {
  response: string;
  session_id: string;
}

export interface SignUpService extends CaptchaRequest {
  email: string;
  password: string;
  language: string;
}

export interface SendResetEmailService extends CaptchaRequest {
  email: string;
}

export interface ResetPasswordService {
  password: string;
  secret: string;
}

export enum ShareLinksInProfileLevel {
  public_share_only = "",
  all_share = "all_share",
  hide_share = "hide_share",
}

export interface AppRegistration {
  id: string;
  name: string;
  homepage_url?: string;
  description?: string;
  consented_scopes?: string[];
  icon?: string;
}

export interface GrantService {
  client_id: string;
  response_type: string;
  redirect_uri: string;
  state: string;
  scope: string;
  code_challenge?: string;
  code_challenge_method?: string;
}

export interface GrantResponse {
  code: string;
  state: string;
}
