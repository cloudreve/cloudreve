use crate::models::common::PaginationResults;
use crate::models::explorer::Share;
use serde::{Deserialize, Serialize};

/// List share service
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListShareService {
    pub page_size: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub order_by: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub order_direction: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub next_page_token: Option<String>,
}

/// List share response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListShareResponse {
    pub shares: Vec<Share>,
    pub pagination: PaginationResults,
}

/// Share create service. Only the target URI is required; omitted fields fall
/// back to the server's public-link defaults.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct ShareCreateService {
    pub uri: String,
    /// Password-protected share; requires `password`.
    #[serde(default)]
    pub is_private: bool,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub password: Option<String>,
    /// Remaining download quota; 0/absent = unlimited.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub downloads: Option<i32>,
    /// Seconds until the share expires; 0/absent = never.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub expire: Option<i32>,
    /// Visitors can preview but not download.
    #[serde(default)]
    pub preview_only: bool,
    /// Visitors can edit files in place (WOPI-capable types).
    #[serde(default)]
    pub allow_edit: bool,
    /// Visitors can upload into the shared folder.
    #[serde(default)]
    pub allow_upload: bool,
    /// Drop-box share: upload-only, contents hidden from visitors.
    #[serde(default)]
    pub upload_only: bool,
}
