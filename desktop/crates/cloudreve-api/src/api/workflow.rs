use crate::client::{Client, RequestOptions};
use crate::error::ApiResult;
use crate::models::workflow::*;
use async_trait::async_trait;

/// Workflow and task API methods
#[async_trait]
pub trait WorkflowApi {
    /// Create archive task
    async fn create_archive(&self, request: &ArchiveWorkflowService) -> ApiResult<TaskResponse>;
    
    /// Extract archive task
    async fn extract_archive(&self, request: &ArchiveWorkflowService) -> ApiResult<TaskResponse>;
    
    /// Relocate files task
    async fn relocate(&self, request: &RelocateWorkflowService) -> ApiResult<TaskResponse>;
    
    /// Remote download task
    async fn remote_download(&self, request: &DownloadWorkflowService) -> ApiResult<TaskResponse>;
    
    /// Import files task
    async fn import(&self, request: &ImportWorkflowService) -> ApiResult<TaskResponse>;
    
    /// List tasks
    async fn list_tasks(&self, params: &ListTaskService) -> ApiResult<TaskListResponse>;
    
    /// Get task by ID
    async fn get_task(&self, task_id: &str) -> ApiResult<TaskResponse>;
    
    /// Cancel task
    async fn cancel_task(&self, task_id: &str) -> ApiResult<()>;
    
    /// Delete task
    async fn delete_task(&self, task_id: &str) -> ApiResult<()>;
    
    /// Set download files for a task
    async fn set_download_files(&self, task_id: &str, request: &SetDownloadFilesService) -> ApiResult<()>;
}

#[async_trait]
impl WorkflowApi for Client {
    async fn create_archive(&self, request: &ArchiveWorkflowService) -> ApiResult<TaskResponse> {
        self.post(
            "/task/archive/create",
            request,
            RequestOptions::new().with_purchase_ticket(),
        ).await
    }
    
    async fn extract_archive(&self, request: &ArchiveWorkflowService) -> ApiResult<TaskResponse> {
        self.post(
            "/task/archive/extract",
            request,
            RequestOptions::new().with_purchase_ticket(),
        ).await
    }
    
    async fn relocate(&self, request: &RelocateWorkflowService) -> ApiResult<TaskResponse> {
        self.post(
            "/task/relocate",
            request,
            RequestOptions::new().with_purchase_ticket(),
        ).await
    }
    
    async fn remote_download(&self, request: &DownloadWorkflowService) -> ApiResult<TaskResponse> {
        self.post(
            "/task/download",
            request,
            RequestOptions::new().with_purchase_ticket(),
        ).await
    }
    
    async fn import(&self, request: &ImportWorkflowService) -> ApiResult<TaskResponse> {
        self.post("/task/import", request, RequestOptions::new()).await
    }
    
    async fn list_tasks(&self, params: &ListTaskService) -> ApiResult<TaskListResponse> {
        let mut query_params = vec![
            format!("page_size={}", params.page_size),
            format!("category={}", serde_json::to_string(&params.category).unwrap_or_default().trim_matches('"')),
        ];
        
        if let Some(next_page_token) = &params.next_page_token {
            query_params.push(format!("next_page_token={}", next_page_token));
        }
        
        let query = format!("?{}", query_params.join("&"));
        
        self.get(&format!("/task{}", query), RequestOptions::new()).await
    }
    
    async fn get_task(&self, task_id: &str) -> ApiResult<TaskResponse> {
        self.get(&format!("/task/{}", task_id), RequestOptions::new()).await
    }
    
    async fn cancel_task(&self, task_id: &str) -> ApiResult<()> {
        self.post::<(), ()>(
            &format!("/task/{}/cancel", task_id),
            &(),
            RequestOptions::new(),
        ).await
    }
    
    async fn delete_task(&self, task_id: &str) -> ApiResult<()> {
        self.delete(&format!("/task/{}", task_id), RequestOptions::new()).await
    }
    
    async fn set_download_files(&self, task_id: &str, request: &SetDownloadFilesService) -> ApiResult<()> {
        self.post(
            &format!("/task/{}/files", task_id),
            request,
            RequestOptions::new(),
        ).await
    }
}

