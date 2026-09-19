use crate::client::{Client, RequestOptions};
use crate::error::ApiResult;
use crate::models::share::*;
use async_trait::async_trait;

/// Share API methods
#[async_trait]
pub trait ShareApi {
    /// Create a share link for a file or folder, returns the share URL.
    async fn create_share(&self, request: &ShareCreateService) -> ApiResult<String>;
}

#[async_trait]
impl ShareApi for Client {
    async fn create_share(&self, request: &ShareCreateService) -> ApiResult<String> {
        self.put("/share", request, RequestOptions::new()).await
    }
}
