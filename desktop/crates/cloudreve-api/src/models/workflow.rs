use crate::models::common::PaginationResults;
use serde::{Deserialize, Serialize};

/// Task status
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum TaskStatus {
    Queued,
    Processing,
    Suspending,
    Error,
    Canceled,
    Completed,
}

/// Task type
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum TaskType {
    CreateArchive,
    ExtractArchive,
    Relocate,
    RemoteDownload,
    #[serde(rename = "media_meta")]
    MediaMetadata,
    EntityRecycleRoutine,
    ExplicitEntityRecycle,
    UploadSentinelCheck,
    Import,
}

/// Task response
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct TaskResponse {
    pub created_at: String,
    pub updated_at: String,
    pub id: String,
    pub status: String,
    #[serde(rename = "type")]
    pub task_type: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub node: Option<NodeSummary>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub summary: Option<TaskSummary>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub error: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub error_history: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub duration: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub resume_time: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub retry_count: Option<i32>,
}

/// Task summary
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct TaskSummary {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub phase: Option<String>,
    pub props: TaskProps,
}

/// Task properties
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct TaskProps {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub src: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub src_str: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub dst: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub src_multiple: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub dst_policy_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub failed: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub download: Option<DownloadTaskStatus>,
}

/// Download task state
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
#[serde(rename_all = "lowercase")]
pub enum DownloadTaskState {
    #[default]
    Seeding,
    Downloading,
    Error,
    Completed,
    Unknown,
}

/// Download task status
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct DownloadTaskStatus {
    pub name: String,
    pub state: DownloadTaskState,
    pub total: i64,
    pub downloaded: i64,
    pub download_speed: i64,
    pub upload_speed: i64,
    pub uploaded: i64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub files: Option<Vec<DownloadTaskFile>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub hash: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub pieces: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub num_pieces: Option<i32>,
}

/// Download task file
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct DownloadTaskFile {
    pub index: i32,
    pub name: String,
    pub size: i64,
    pub progress: f64,
    pub selected: bool,
}

/// Node summary
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NodeSummary {
    pub id: String,
    pub name: String,
    #[serde(rename = "type")]
    pub node_type: NodeTypes,
    pub capabilities: String,
}

/// Node types
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum NodeTypes {
    Master,
    Slave,
}

/// Archive workflow service
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ArchiveWorkflowService {
    pub src: Vec<String>,
    pub dst: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub preferred_node_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub encoding: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub password: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub file_mask: Option<Vec<String>>,
}

/// Relocate workflow service
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RelocateWorkflowService {
    pub src: Vec<String>,
    pub dst_policy_id: String,
}

/// Download workflow service
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DownloadWorkflowService {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub src: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub src_file: Option<String>,
    pub dst: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub preferred_node_id: Option<String>,
}

/// Import workflow service
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ImportWorkflowService {
    pub src: String,
    pub dst: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub extract_media_meta: Option<bool>,
    pub user_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub recursive: Option<bool>,
    pub policy_id: i32,
}

/// List task service
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListTaskService {
    pub page_size: i32,
    pub category: ListTaskCategory,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub next_page_token: Option<String>,
}

/// List task category
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum ListTaskCategory {
    General,
    Downloading,
    Downloaded,
}

/// Task list response
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct TaskListResponse {
    pub tasks: Vec<TaskResponse>,
    pub pagination: PaginationResults,
}

/// Set download files service
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SetDownloadFilesService {
    pub files: Vec<SetFileToDownloadArgs>,
}

/// Set file to download args
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SetFileToDownloadArgs {
    pub index: i32,
    pub download: bool,
}

/// Node capability constants
pub mod node_capability {
    pub const NONE: i32 = 0;
    pub const CREATE_ARCHIVE: i32 = 1;
    pub const EXTRACT_ARCHIVE: i32 = 2;
    pub const REMOTE_DOWNLOAD: i32 = 3;
}
