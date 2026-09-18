use serde::{Deserialize, Serialize};

/// License information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct License {
    pub expired_at: String,
    pub signed_at: String,
    pub root_domains: Vec<String>,
    pub domains: Vec<String>,
    pub vol_domains: Vec<String>,
}

/// Metrics summary
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MetricsSummary {
    pub dates: Vec<String>,
    pub files: Vec<i32>,
    pub users: Vec<i32>,
    pub shares: Vec<i32>,
    pub file_total: i32,
    pub user_total: i32,
    pub share_total: i32,
    pub entities_total: i32,
    pub generated_at: String,
}

/// Version information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Version {
    pub version: String,
    pub pro: bool,
    pub commit: String,
}

/// Homepage summary
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HomepageSummary {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub metrics_summary: Option<MetricsSummary>,
    pub site_urls: Vec<String>,
    pub license: License,
    pub version: Version,
}

