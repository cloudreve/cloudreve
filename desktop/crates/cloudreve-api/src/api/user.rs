use crate::client::{Client, RequestOptions};
use crate::error::ApiResult;
use crate::models::explorer::StoragePolicy;
use crate::models::user::*;
use async_trait::async_trait;

/// User and authentication API methods
#[async_trait]
pub trait UserApi {
    /// Login with email and password
    async fn login(&self, email: &str, password: &str) -> ApiResult<LoginResponse>;
    
    /// Login with 2FA code
    async fn login_2fa(&self, otp: &str, session_id: &str) -> ApiResult<LoginResponse>;
    
    /// Get current user information
    async fn get_user_me(&self) -> ApiResult<User>;
    
    /// Get user capacity information
    async fn get_user_capacity(&self) -> ApiResult<Capacity>;
    
    /// Get user settings
    async fn get_user_settings(&self) -> ApiResult<UserSettings>;
    
    /// Update user settings
    async fn patch_user_settings(&self, settings: &PatchUserSetting) -> ApiResult<()>;
    
    /// Get credit change log
    async fn get_credit_log(&self, params: &GetCreditLogService) -> ApiResult<CreditChangeLogResponse>;
    
    /// Sign up a new user
    async fn sign_up(&self, request: &SignUpService) -> ApiResult<()>;
    
    /// Send password reset email
    async fn send_reset_email(&self, request: &SendResetEmailService) -> ApiResult<()>;
    
    /// Reset password with secret
    async fn reset_password(&self, request: &ResetPasswordService) -> ApiResult<()>;

    /// Get user storage policies
    async fn get_user_storage_policies(&self) -> ApiResult<Vec<StoragePolicy>>;
}

#[async_trait]
impl UserApi for Client {
    async fn login(&self, email: &str, password: &str) -> ApiResult<LoginResponse> {
        let request = PasswordLoginRequest {
            email: email.to_string(),
            password: password.to_string(),
            captcha: None,
        };
        
        self.post(
            "/session/login",
            &request,
            RequestOptions::new().no_credential(),
        ).await
    }
    
    async fn login_2fa(&self, otp: &str, session_id: &str) -> ApiResult<LoginResponse> {
        let request = TwoFALoginRequest {
            otp: otp.to_string(),
            session_id: session_id.to_string(),
        };
        
        self.post(
            "/session/2fa",
            &request,
            RequestOptions::new().no_credential(),
        ).await
    }
    
    async fn get_user_me(&self) -> ApiResult<User> {
        self.get("/user/me", RequestOptions::new()).await
    }
    
    async fn get_user_capacity(&self) -> ApiResult<Capacity> {
        self.get("/user/capacity", RequestOptions::new()).await
    }

    async fn get_user_storage_policies(&self) -> ApiResult<Vec<StoragePolicy>> {
        self.get("/user/setting/policies", RequestOptions::new()).await
    }
    
    async fn get_user_settings(&self) -> ApiResult<UserSettings> {
        self.get("/user/setting", RequestOptions::new()).await
    }
    
    async fn patch_user_settings(&self, settings: &PatchUserSetting) -> ApiResult<()> {
        self.patch("/user/setting", settings, RequestOptions::new()).await
    }
    
    async fn get_credit_log(&self, params: &GetCreditLogService) -> ApiResult<CreditChangeLogResponse> {
        // Build query string from params
        let mut query_params = vec![];
        if let Some(page_size) = params.page_size {
            query_params.push(format!("page_size={}", page_size));
        }
        if let Some(order_by) = &params.order_by {
            query_params.push(format!("order_by={}", order_by));
        }
        if let Some(order_direction) = &params.order_direction {
            query_params.push(format!("order_direction={}", order_direction));
        }
        if let Some(next_page_token) = &params.next_page_token {
            query_params.push(format!("next_page_token={}", next_page_token));
        }
        
        let query = if query_params.is_empty() {
            String::new()
        } else {
            format!("?{}", query_params.join("&"))
        };
        
        self.get(&format!("/user/credit/log{}", query), RequestOptions::new()).await
    }
    
    async fn sign_up(&self, request: &SignUpService) -> ApiResult<()> {
        self.post(
            "/user/register",
            request,
            RequestOptions::new().no_credential(),
        ).await
    }
    
    async fn send_reset_email(&self, request: &SendResetEmailService) -> ApiResult<()> {
        self.post(
            "/user/reset",
            request,
            RequestOptions::new().no_credential(),
        ).await
    }
    
    async fn reset_password(&self, request: &ResetPasswordService) -> ApiResult<()> {
        self.post(
            "/user/reset/confirm",
            request,
            RequestOptions::new().no_credential(),
        ).await
    }
}

