use serde::{Deserialize, Serialize};

/// Pagination results
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct PaginationResults {
    pub page: i32,
    pub page_size: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub total_items: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub next_token: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub is_cursor: Option<bool>,
}

/// Pagination arguments for list requests
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct PaginationArgs {
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

pub struct ListAllRes<T> {
    pub res: T,
    pub more: bool,
}
