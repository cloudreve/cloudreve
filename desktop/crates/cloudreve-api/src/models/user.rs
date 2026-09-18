use crate::models::common::PaginationResults;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// User model
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct User {
    pub id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub email: Option<String>,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub status: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar: Option<String>,
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub preferred_theme: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub credit: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub anonymous: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub group: Option<Group>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub pined: Option<Vec<PinedFile>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub language: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub disable_view_sync: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub share_links_in_profile: Option<ShareLinksInProfileLevel>,
}

/// User group
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Group {
    pub id: String,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub permission: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub direct_link_batch_size: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub trash_retention: Option<i32>,
}

/// Pinned file
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PinedFile {
    pub uri: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub name: Option<String>,
}

/// Share links visibility level
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
#[serde(rename_all = "snake_case")]
pub enum ShareLinksInProfileLevel {
    #[serde(rename = "")]
    #[default]
    PublicShareOnly,
    AllShare,
    HideShare,
}

/// Authentication token
#[derive(Clone, Serialize, Deserialize, Default)]
pub struct Token {
    pub access_token: String,
    pub refresh_token: String,
    pub access_expires: String,
    pub refresh_expires: String,
}

impl std::fmt::Debug for Token {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("Token")
            .field("access_token", &"[REDACTED]")
            .field("refresh_token", &"[REDACTED]")
            .field("access_expires", &self.access_expires)
            .field("refresh_expires", &self.refresh_expires)
            .finish()
    }
}

/// Login response
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct LoginResponse {
    pub user: User,
    pub token: Token,
}

/// Password login request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PasswordLoginRequest {
    pub email: String,
    pub password: String,
    #[serde(flatten)]
    pub captcha: Option<HashMap<String, serde_json::Value>>,
}

/// 2FA login request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TwoFALoginRequest {
    pub otp: String,
    pub session_id: String,
}

/// Refresh token request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RefreshTokenRequest {
    pub refresh_token: String,
}

/// User capacity information
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct Capacity {
    pub total: i64,
    pub used: i64,
    #[serde(default)]
    pub storage_pack_total: i64,
}

/// User settings
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct UserSettings {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub group_expires: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub open_id: Option<Vec<OpenID>>,
    pub version_retention_enabled: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub version_retention_ext: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub version_retention_max: Option<i32>,
    pub passwordless: bool,
    pub two_fa_enabled: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub passkeys: Option<Vec<Passkey>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub login_activity: Option<Vec<LoginActivity>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub storage_packs: Option<Vec<StoragePack>>,
    pub credit: i32,
    pub disable_view_sync: bool,
    pub share_links_in_profile: ShareLinksInProfileLevel,
}

/// Patch user settings request
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct PatchUserSetting {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub nick: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub group_expires: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub language: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub preferred_theme: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub version_retention_enabled: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub version_retention_ext: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub version_retention_max: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub current_password: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub new_password: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub two_fa_enabled: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub two_fa_code: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub disable_view_sync: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub share_links_in_profile: Option<ShareLinksInProfileLevel>,
}

/// OpenID provider
#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
pub enum OpenIDProvider {
    #[serde(rename = "0")]
    Logto = 0,
    #[serde(rename = "1")]
    QQ = 1,
    #[serde(rename = "2")]
    OIDC = 2,
}

/// OpenID information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OpenID {
    pub provider: OpenIDProvider,
    pub linked_at: String,
}

/// Passkey information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Passkey {
    pub id: String,
    pub name: String,
    pub created_at: String,
    pub used_at: String,
}

/// Login activity
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LoginActivity {
    pub created_at: String,
    pub ip: String,
    pub browser: String,
    pub device: String,
    pub os: String,
    pub login_with: String,
    pub open_id_provider: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub passkey: Option<String>,
    pub success: bool,
    pub webdav: bool,
}

/// Storage pack information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StoragePack {
    pub name: String,
    pub active_since: String,
    pub expire_at: String,
    pub size: i64,
}

/// Credit change log entry
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct CreditChangeLog {
    pub changed_at: String,
    pub diff: i32,
    pub reason: String,
}

/// Credit change log response
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct CreditChangeLogResponse {
    pub changes: Vec<CreditChangeLog>,
    pub pagination: PaginationResults,
}

/// Get credit log service parameters
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct GetCreditLogService {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub page_size: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub order_by: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub order_direction: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub next_page_token: Option<String>,
}

/// Sign up request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SignUpService {
    pub email: String,
    pub password: String,
    pub language: String,
    #[serde(flatten)]
    pub captcha: Option<HashMap<String, serde_json::Value>>,
}

/// Send reset email request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SendResetEmailService {
    pub email: String,
    #[serde(flatten)]
    pub captcha: Option<HashMap<String, serde_json::Value>>,
}

/// Reset password request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResetPasswordService {
    pub password: String,
    pub secret: String,
}

/// Group permissions constants
pub mod group_permission {
    pub const IS_ADMIN: i32 = 0;
    pub const IS_ANONYMOUS: i32 = 1;
    pub const SHARE: i32 = 2;
    pub const WEBDAV: i32 = 3;
    pub const ARCHIVE_DOWNLOAD: i32 = 4;
    pub const ARCHIVE_TASK: i32 = 5;
    pub const WEBDAV_PROXY: i32 = 6;
    pub const SHARE_DOWNLOAD: i32 = 7;
    pub const SHARE_FREE: i32 = 8;
    pub const REMOTE_DOWNLOAD: i32 = 9;
    pub const RELOCATE: i32 = 10;
    pub const REDIRECTED_SOURCE: i32 = 11;
    pub const ADVANCE_DELETE: i32 = 12;
    pub const SELECT_NODE: i32 = 13;
    pub const SET_ANONYMOUS_PERMISSION: i32 = 14;
    pub const SET_EXPLICIT_USER_PERMISSION: i32 = 15;
    pub const IGNORE_FILE_PERMISSION: i32 = 16;
    pub const UNIQUE_DIRECT_LINK: i32 = 17;
}
