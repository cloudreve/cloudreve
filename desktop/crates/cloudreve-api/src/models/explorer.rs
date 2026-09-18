use crate::models::common::PaginationResults;
use crate::models::user::User;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// List file service parameters
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListFileService {
    pub uri: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub page: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub page_size: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub order_by: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub order_direction: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub next_page_token: Option<String>,
}

/// File type constants
pub mod file_type {
    pub const FILE: i32 = 0;
    pub const FOLDER: i32 = 1;
}

/// File response
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct FileResponse {
    #[serde(rename = "type")]
    pub file_type: i32,
    pub id: String,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub permission: Option<String>,
    pub created_at: String,
    pub updated_at: String,
    pub size: i64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub metadata: Option<HashMap<String, String>>,
    pub path: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub shared: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub capability: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub owned: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub folder_summary: Option<FolderSummary>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub extended_info: Option<ExtendedInfo>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub primary_entity: Option<String>,
}

/// Folder summary
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct FolderSummary {
    pub size: i64,
    pub files: i32,
    pub folders: i32,
    pub completed: bool,
    pub calculated_at: String,
}

/// Extended file information
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct ExtendedInfo {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub storage_policy: Option<StoragePolicy>,
    pub storage_policy_inherited: bool,
    pub storage_used: i64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub shares: Option<Vec<Share>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub entities: Option<Vec<Entity>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub permissions: Option<PermissionSettingReq>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub view: Option<ExplorerView>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub direct_links: Option<Vec<DirectLink>>,
}

/// Direct link
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct DirectLink {
    pub id: String,
    pub created_at: String,
    pub url: String,
    #[serde(default)]
    pub downloaded: i32,
}

/// Entity
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct Entity {
    pub id: String,
    #[serde(rename = "type")]
    pub entity_type: i32,
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub storage_policy: Option<StoragePolicy>,
    pub size: i64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub created_by: Option<User>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub encrypted_with: Option<EncryptionCipher>,
}

/// Share information
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct Share {
    pub id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub name: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub expires: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_private: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub share_view: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub remain_downloads: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub created_at: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub price: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub permission_setting: Option<PermissionSettingReq>,
    pub url: String,
    pub visited: i32,
    #[serde(default)]
    pub downloaded: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub expired: Option<bool>,
    pub unlocked: bool,
    #[serde(default)]
    pub password_protected: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub source_type: Option<i32>,
    pub owner: User,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub permissions: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub source_uri: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub password: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub show_readme: Option<bool>,
}

/// Storage policy type
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
#[serde(rename_all = "lowercase")]
pub enum PolicyType {
    #[default]
    Local,
    Remote,
    Oss,
    Qiniu,
    Onedrive,
    Cos,
    Upyun,
    S3,
    Ks3,
    Obs,
    #[serde(rename = "load_balance")]
    LoadBalance,
}

/// Storage policy
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct StoragePolicy {
    pub id: String,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub allowed_suffix: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub denied_suffix: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub allowed_name_regexp: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub denied_name_regexp: Option<String>,
    pub max_size: i64,
    #[serde(rename = "type")]
    pub policy_type: PolicyType,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub relay: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub weight: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub children: Option<Vec<StoragePolicy>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub chunk_concurrency: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub encryption: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub streaming_encryption: Option<bool>,
}

/// List response
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct ListResponse {
    pub files: Vec<FileResponse>,
    pub pagination: PaginationResults,
    pub props: NavigatorProps,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub context_hint: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub recursion_limit_reached: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mixed_type: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub single_file_view: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub parent: Option<FileResponse>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub storage_policy: Option<StoragePolicy>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub view: Option<ExplorerView>,
}

/// Navigator properties
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct NavigatorProps {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub capability: Option<String>,
    pub max_page_size: i32,
    pub order_by_options: Vec<String>,
    pub order_direction_options: Vec<String>,
}

/// Explorer view settings
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct ExplorerView {
    pub page_size: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub order: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub order_direction: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub view: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub thumbnail: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub gallery_width: Option<i32>,
}

/// File thumbnail response
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct FileThumbResponse {
    pub url: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub expires: Option<String>,
    pub obfuscated: bool,
}

/// Delete file service
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeleteFileService {
    pub uris: Vec<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub unlink: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub skip_soft_delete: Option<bool>,
}

/// Unlock file service
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UnlockFileService {
    pub tokens: Vec<String>,
}

/// Rename file service
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RenameFileService {
    pub uri: String,
    pub new_name: String,
}

/// Move file service
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MoveFileService {
    pub uris: Vec<String>,
    pub dst: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub copy: Option<bool>,
}

/// Metadata patch
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MetadataPatch {
    pub key: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub value: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub remove: Option<bool>,
}

/// Patch metadata service
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PatchMetadataService {
    pub uris: Vec<String>,
    pub patches: Vec<MetadataPatch>,
}

/// Permission setting request
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct PermissionSettingReq {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub same_group: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub everyone: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub other: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub anonymous: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub group_explicit: Option<HashMap<String, String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub user_explicit: Option<HashMap<String, String>>,
}

/// Create file service
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateFileService {
    pub uri: String,
    #[serde(rename = "type")]
    pub file_type: String, // "file" or "folder"
    #[serde(skip_serializing_if = "Option::is_none")]
    pub err_on_conflict: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub metadata: Option<HashMap<String, String>>,
}

/// File URL service
#[derive(Debug, Default, Clone, Serialize, Deserialize)]
pub struct FileURLService {
    pub uris: Vec<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub download: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub redirect: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub entity: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub no_cache: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub skip_error: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub use_primary_site_url: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub archive: Option<bool>,
}

/// File URL response
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct FileURLResponse {
    pub urls: Vec<EntityURLResponse>,
    pub expires: String,
}

/// Entity URL response
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct EntityURLResponse {
    pub url: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub stream_saver_display_name: Option<String>,
}

/// Get file info service
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct GetFileInfoService {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub uri: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub extended: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub folder_summary: Option<bool>,
}

/// Version control service
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VersionControlService {
    pub uri: String,
    pub version: String,
}

/// File update service
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FileUpdateService {
    pub uri: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub previous: Option<String>,
}

/// Upload credential
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct UploadCredential {
    pub session_id: String,
    pub expires: i64,
    pub chunk_size: i64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub upload_urls: Option<Vec<String>>,
    #[serde(default)]
    pub credential: String,
    #[serde(default)]
    pub upload_id: String,
    pub callback_secret: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub ak: Option<String>,
    #[serde(rename = "keyTime")]
    #[serde(skip_serializing_if = "Option::is_none")]
    pub key_time: Option<String>,
    #[serde(rename = "completeURL")]
    #[serde(skip_serializing_if = "Option::is_none")]
    pub complete_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub storage_policy: Option<StoragePolicy>,
    pub uri: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mime_type: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub upload_policy: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub encrypt_metadata: Option<EncryptMetadata>,
}

/// Encryption metadata
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EncryptMetadata {
    pub algorithm: EncryptionCipher,
    pub key_plain_text: String,
    pub iv: String,
}

/// Encryption cipher
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum EncryptionCipher {
    #[serde(rename = "aes-256-ctr")]
    Aes256Ctr,
}

/// Delete upload session service
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeleteUploadSessionService {
    pub id: String,
    pub uri: String,
}

/// Mount policy service
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MountPolicyService {
    pub uri: String,
    pub policy_id: String,
}

/// Set permission service
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SetPermissionService {
    pub uris: Vec<String>,
    pub setting: PermissionSettingReq,
}

/// Upload session request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UploadSessionRequest {
    pub uri: String,
    pub size: i64,
    pub policy_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_modified: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub entity_type: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub metadata: Option<HashMap<String, String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mime_type: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub encryption_supported: Option<Vec<EncryptionCipher>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub previous: Option<String>,
}

/// Metadata key constants
pub mod metadata {
    pub const SHARE_REDIRECT: &str = "sys:shared_redirect";
    pub const SHARE_OWNER: &str = "sys:shared_owner";
    pub const UPLOAD_SESSION_ID: &str = "sys:upload_session_id";
    pub const ICON_COLOR: &str = "customize:icon_color";
    pub const EMOJI: &str = "customize:emoji";
    pub const LIVE_PHOTO: &str = "customize:live_photo";
    pub const TAG_PREFIX: &str = "tag:";
    pub const THUMBNAIL_DISABLED: &str = "thumb:disabled";
}

/// File permission constants
pub mod file_permission {
    pub const READ: i32 = 0;
    pub const UPDATE: i32 = 1;
    pub const CREATE: i32 = 2;
    pub const DELETE: i32 = 3;
}

/// Navigator capability constants
pub mod navigator_capability {
    pub const CREATE_FILE: i32 = 0;
    pub const RENAME_FILE: i32 = 1;
    pub const SET_PERMISSION: i32 = 2;
    pub const MOVE_TO_MY: i32 = 3;
    pub const MOVE_TO_SHARE: i32 = 4;
    pub const MOVE_TO_TRASH: i32 = 5;
    pub const UPLOAD_FILE: i32 = 6;
    pub const DOWNLOAD_FILE: i32 = 7;
    pub const UPDATE_METADATA: i32 = 8;
    pub const LIST_CHILDREN: i32 = 9;
    pub const GENERATE_THUMB: i32 = 10;
    pub const COPY_TO_MY: i32 = 11;
    pub const COPY_TO_SHARE: i32 = 12;
    pub const COPY_TO_TRASH: i32 = 13;
    pub const DELETE_FILE: i32 = 14;
    pub const LOCK_FILE: i32 = 15;
    pub const SOFT_DELETE: i32 = 16;
    pub const RESTORE: i32 = 17;
    pub const SHARE: i32 = 18;
    pub const INFO: i32 = 19;
    pub const VERSION_CONTROL: i32 = 20;
    pub const MOUNT: i32 = 21;
    pub const RELOCATE: i32 = 22;
    pub const ENTER_FOLDER: i32 = 23;
}

/// File event type for SSE events
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum FileEventType {
    Create,
    Modify,
    Rename,
    Delete,
}

/// File event data received from SSE
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FileEventData {
    #[serde(rename = "type")]
    pub event_type: FileEventType,
    pub file_id: String,
    pub from: String,
    #[serde(default)]
    pub to: String,
}

/// SSE event types from the file events endpoint
#[derive(Debug, Clone)]
pub enum FileEvent {
    /// Connection resumed
    Resumed,
    /// Subscription confirmed
    Subscribed,
    /// Keep-alive ping
    KeepAlive,
    /// Reconnect required
    ReconnectRequired,
    /// Batch of file events with data
    Event(Vec<FileEventData>),
}
