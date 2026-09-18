use serde::{Deserialize, Serialize};
use crate::models::common::PaginationResults;
use crate::models::explorer::Share;

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

