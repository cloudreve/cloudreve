use crate::client::{Client, RequestOptions};
use crate::error::ApiResult;
use crate::models::site::*;
use async_trait::async_trait;

/// Site configuration API methods
#[async_trait]
pub trait SiteApi {
    /// Get site configuration
    async fn get_site_config(&self, section: &str) -> ApiResult<SiteConfig>;
    
    /// Get captcha
    async fn get_captcha(&self) -> ApiResult<CaptchaResponse>;
    
    /// Create abuse report
    async fn create_abuse_report(&self, request: &CreateAbuseReportService) -> ApiResult<()>;
}

#[async_trait]
impl SiteApi for Client {
    async fn get_site_config(&self, section: &str) -> ApiResult<SiteConfig> {
        self.get(
            &format!("/site/config/{}", section),
            RequestOptions::new().no_credential(),
        ).await
    }
    
    async fn get_captcha(&self) -> ApiResult<CaptchaResponse> {
        self.get("/site/captcha", RequestOptions::new().no_credential()).await
    }
    
    async fn create_abuse_report(&self, request: &CreateAbuseReportService) -> ApiResult<()> {
        self.post(
            "/site/abuse",
            request,
            RequestOptions::new().no_credential(),
        ).await
    }
}

