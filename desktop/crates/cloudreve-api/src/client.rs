use crate::error::{ApiError, ApiResponse, ApiResult, ErrorCode, LockConflictDetail};
use crate::models::user::{RefreshTokenRequest, Token};
use chrono::{DateTime, Duration, Utc};
use reqwest::{Client as HttpClient, Method};
use serde::de::DeserializeOwned;
use serde::Serialize;
use std::future::Future;
use std::pin::Pin;
use std::sync::Arc;
use tokio::sync::RwLock;

const API_PREFIX: &str = "/api/v4";
pub const CR_HEADER_PREFIX: &str = "X-Cr-";

/// Client configuration
#[derive(Debug, Clone)]
pub struct ClientConfig {
    /// Base URL of the Cloudreve instance (e.g., "https://example.com")
    pub base_url: String,
    /// Timeout for requests in seconds
    pub timeout_seconds: u64,
    /// Client ID
    pub client_id: String,
    /// User agent string for HTTP requests
    pub user_agent: Option<String>,
}

impl ClientConfig {
    /// Create a new configuration with the given base URL
    pub fn new(base_url: impl Into<String>) -> Self {
        Self {
            base_url: base_url.into(),
            timeout_seconds: 60,
            client_id: "".to_string(),
            user_agent: None,
        }
    }

    /// Set the request timeout
    pub fn with_timeout(mut self, timeout_seconds: u64) -> Self {
        self.timeout_seconds = timeout_seconds;
        self
    }

    /// Set the client ID
    pub fn with_client_id(mut self, client_id: impl Into<String>) -> Self {
        self.client_id = client_id.into();
        self
    }

    /// Set the user agent string
    pub fn with_user_agent(mut self, user_agent: impl Into<String>) -> Self {
        self.user_agent = Some(user_agent.into());
        self
    }
}

/// Token storage with expiration tracking
#[derive(Debug, Clone)]
pub(crate) struct TokenStore {
    access_token: Option<String>,
    refresh_token: Option<String>,
    access_token_expires: Option<DateTime<Utc>>,
    refresh_token_expires: Option<DateTime<Utc>>,
}

impl TokenStore {
    fn new() -> Self {
        Self {
            access_token: None,
            refresh_token: None,
            access_token_expires: None,
            refresh_token_expires: None,
        }
    }

    fn is_access_token_expired(&self) -> bool {
        self.access_token_expires
            .map(|exp| Utc::now() >= exp)
            .unwrap_or(true)
    }

    fn is_refresh_token_expired(&self) -> bool {
        self.refresh_token_expires
            .map(|exp| Utc::now() >= exp)
            .unwrap_or(true)
    }

    fn has_tokens(&self) -> bool {
        self.access_token.is_some() && self.refresh_token.is_some()
    }
}

/// Request options for customizing API calls
#[derive(Debug, Clone, Default)]
pub struct RequestOptions {
    /// Don't include authentication credentials
    pub no_credential: bool,
    /// Include purchase ticket header
    pub with_purchase_ticket: bool,
    /// Skip batch error handling (return first error)
    pub skip_batch_error: bool,
    /// Skip lock conflict handling
    pub skip_lock_conflict: bool,
}

impl RequestOptions {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn no_credential(mut self) -> Self {
        self.no_credential = true;
        self
    }

    pub fn with_purchase_ticket(mut self) -> Self {
        self.with_purchase_ticket = true;
        self
    }

    pub fn skip_batch_error(mut self) -> Self {
        self.skip_batch_error = true;
        self
    }

    pub fn skip_lock_conflict(mut self) -> Self {
        self.skip_lock_conflict = true;
        self
    }
}

/// Callback type for credential refresh events
pub type OnCredentialRefreshed =
    Arc<dyn Fn(Token) -> Pin<Box<dyn Future<Output = ()> + Send>> + Send + Sync>;

/// Callback type for credential invalid/expired events (401, 40020, 40089)
pub type OnCredentialInvalid =
    Arc<dyn Fn() -> Pin<Box<dyn Future<Output = ()> + Send>> + Send + Sync>;

/// Main Cloudreve API client
pub struct Client {
    pub(crate) config: ClientConfig,
    pub(crate) http_client: HttpClient,
    pub(crate) tokens: Arc<RwLock<TokenStore>>,
    pub(crate) purchase_ticket: Arc<RwLock<Option<String>>>,
    on_credential_refreshed: Option<OnCredentialRefreshed>,
    on_credential_invalid: Option<OnCredentialInvalid>,
}

impl Client {
    /// Create a new API client
    pub fn new(config: ClientConfig) -> Self {
        let mut builder = HttpClient::builder()
            .connect_timeout(std::time::Duration::from_secs(config.timeout_seconds));

        if let Some(ref user_agent) = config.user_agent {
            builder = builder.user_agent(user_agent);
        }

        let http_client = builder.build().expect("Failed to create HTTP client");

        Self {
            config,
            http_client,
            tokens: Arc::new(RwLock::new(TokenStore::new())),
            purchase_ticket: Arc::new(RwLock::new(None)),
            on_credential_refreshed: None,
            on_credential_invalid: None,
        }
    }

    /// Set a callback to be invoked when credentials are refreshed
    ///
    /// The callback receives the new token information and can perform async operations
    /// such as persisting tokens to storage.
    ///
    /// # Example
    /// ```no_run
    /// use std::sync::Arc;
    /// use std::pin::Pin;
    /// use std::future::Future;
    ///
    /// client.set_on_credential_refreshed(Arc::new(|token| {
    ///     Box::pin(async move {
    ///         // Save token to storage
    ///         println!("New access token: {}", token.access_token);
    ///     })
    /// }));
    /// ```
    pub fn set_on_credential_refreshed(&mut self, callback: OnCredentialRefreshed) {
        self.on_credential_refreshed = Some(callback);
    }

    /// Clear the credential refresh callback
    pub fn clear_on_credential_refreshed(&mut self) {
        self.on_credential_refreshed = None;
    }

    /// Set a callback to be invoked when credentials are invalid or expired
    ///
    /// This callback is triggered when the API returns error codes indicating
    /// authentication failure: 401 (LoginRequired), 40020 (CredentialInvalid),
    /// or 40089 (SessionExpired).
    pub fn set_on_credential_invalid(&mut self, callback: OnCredentialInvalid) {
        self.on_credential_invalid = Some(callback);
    }

    /// Clear the credential invalid callback
    pub fn clear_on_credential_invalid(&mut self) {
        self.on_credential_invalid = None;
    }

    /// Invoke the credential invalid callback if set
    async fn notify_credential_invalid(&self) {
        if let Some(ref callback) = self.on_credential_invalid {
            callback().await;
        }
    }

    /// Set authentication tokens
    pub async fn set_tokens(&self, access_token: String, refresh_token: String) {
        let mut store = self.tokens.write().await;

        // Parse expiration from token if available, otherwise use default
        // In a real implementation, you might want to parse JWT tokens
        let access_expires = Utc::now() + Duration::hours(1);
        let refresh_expires = Utc::now() + Duration::days(7);

        store.access_token = Some(access_token);
        store.refresh_token = Some(refresh_token);
        store.access_token_expires = Some(access_expires);
        store.refresh_token_expires = Some(refresh_expires);
    }

    /// Set tokens from a Token response with explicit expiration times
    ///
    /// This method validates the access token by parsing it as a JWT and checking
    /// that the `scopes` field exists and is not empty.
    pub async fn set_tokens_with_expiry(&self, token: &Token) -> ApiResult<()> {
        // Validate the access token
        Self::validate_jwt_scopes(&token.access_token)?;

        let mut store = self.tokens.write().await;

        store.access_token = Some(token.access_token.clone());
        store.refresh_token = Some(token.refresh_token.clone());

        // Parse RFC3339 timestamps
        if let Ok(exp) = DateTime::parse_from_rfc3339(&token.access_expires) {
            store.access_token_expires = Some(exp.with_timezone(&Utc));
        }
        if let Ok(exp) = DateTime::parse_from_rfc3339(&token.refresh_expires) {
            store.refresh_token_expires = Some(exp.with_timezone(&Utc));
        }

        Ok(())
    }

    /// Validate that a JWT token has a non-empty `scopes` field
    fn validate_jwt_scopes(token: &str) -> ApiResult<()> {
        // JWT format: header.payload.signature
        let parts: Vec<&str> = token.split('.').collect();
        if parts.len() != 3 {
            return Err(ApiError::InvalidToken("Invalid JWT format".to_string()));
        }

        // Decode the payload (second part) - JWT uses base64url encoding
        let payload_b64 = parts[1];
        let payload_bytes = base64::Engine::decode(
            &base64::engine::general_purpose::URL_SAFE_NO_PAD,
            payload_b64,
        )
        .map_err(|e| ApiError::InvalidToken(format!("Failed to decode JWT payload: {}", e)))?;

        let payload: serde_json::Value = serde_json::from_slice(&payload_bytes)
            .map_err(|e| ApiError::InvalidToken(format!("Failed to parse JWT payload: {}", e)))?;

        // Check that scopes field exists and is not empty
        match payload.get("scopes") {
            None => Err(ApiError::InvalidToken(
                "Token missing 'scopes' field".to_string(),
            )),
            Some(scopes) => {
                if let Some(arr) = scopes.as_array() {
                    if arr.is_empty() {
                        return Err(ApiError::InvalidToken(
                            "Token 'scopes' field is empty".to_string(),
                        ));
                    }
                } else if let Some(s) = scopes.as_str() {
                    if s.is_empty() {
                        return Err(ApiError::InvalidToken(
                            "Token 'scopes' field is empty".to_string(),
                        ));
                    }
                } else {
                    return Err(ApiError::InvalidToken(
                        "Token 'scopes' field has invalid type".to_string(),
                    ));
                }
                Ok(())
            }
        }
    }

    /// Clear all authentication tokens
    pub async fn clear_tokens(&self) {
        let mut store = self.tokens.write().await;
        *store = TokenStore::new();
    }

    /// Set the purchase ticket for subsequent requests
    pub async fn set_purchase_ticket(&self, ticket: Option<String>) {
        let mut pt = self.purchase_ticket.write().await;
        *pt = ticket;
    }

    /// Get a valid access token, refreshing if necessary
    pub(crate) async fn get_access_token(&self) -> ApiResult<String> {
        let store = self.tokens.read().await;

        // Check if we have tokens
        if !store.has_tokens() {
            self.notify_credential_invalid().await;
            return Err(ApiError::NoTokensAvailable);
        }

        // Check if refresh token is expired
        if store.is_refresh_token_expired() {
            self.notify_credential_invalid().await;
            return Err(ApiError::RefreshTokenExpired);
        }

        // If access token is not expired, return it
        if !store.is_access_token_expired() {
            return Ok(store.access_token.clone().unwrap());
        }

        // Access token expired, need to refresh
        drop(store); // Release read lock before calling refresh

        self.refresh_access_token().await
    }

    /// Refresh the access token using the refresh token
    async fn refresh_access_token(&self) -> ApiResult<String> {
        let refresh_token = {
            let store = self.tokens.read().await;
            store
                .refresh_token
                .clone()
                .ok_or(ApiError::NoTokensAvailable)?
        };

        // Call refresh token API without credentials - use direct HTTP call to avoid recursion
        let url = self.build_url("/session/token/refresh");
        let request = RefreshTokenRequest { refresh_token };

        let response = self.http_client.post(&url).json(&request).send().await?;

        let api_response: ApiResponse<Token> = response.json().await?;

        if api_response.code != ErrorCode::Success as i32 {
            if let Some(error_code) = ErrorCode::from_code(api_response.code) {
                if error_code.is_credential_error() {
                    self.notify_credential_invalid().await;
                }
            }
            return Err(ApiError::from_response(api_response));
        }

        let token = api_response
            .data
            .ok_or_else(|| ApiError::Other("No token in response".to_string()))?;

        // Update tokens
        self.set_tokens_with_expiry(&token).await?;

        // Invoke callback if set
        if let Some(ref callback) = self.on_credential_refreshed {
            callback(token.clone()).await;
        }

        Ok(token.access_token)
    }

    /// Build the full URL for an API endpoint
    pub(crate) fn build_url(&self, path: &str) -> String {
        format!("{}{}{}", self.config.base_url, API_PREFIX, path)
    }

    /// Internal send method that handles the actual HTTP request
    async fn send_internal<T, R>(
        &self,
        path: &str,
        method: Method,
        body: Option<&T>,
        options: RequestOptions,
    ) -> ApiResult<R>
    where
        T: Serialize + ?Sized,
        R: DeserializeOwned + Default,
    {
        let url = self.build_url(path);
        let mut request = self.http_client.request(method, &url);

        // Add authentication header if needed
        if !options.no_credential {
            let token = self.get_access_token().await?;
            request = request.header("Authorization", format!("Bearer {}", token));
        }

        // Add client ID header if set
        if !self.config.client_id.is_empty() {
            request = request.header(
                format!("{}Client-Id", CR_HEADER_PREFIX),
                self.config.client_id.clone(),
            );
        }

        // Add purchase ticket if requested
        if options.with_purchase_ticket {
            let ticket = self.purchase_ticket.read().await;
            if let Some(t) = ticket.as_ref() {
                request = request.header(format!("{}Purchase-Ticket", CR_HEADER_PREFIX), t);
            }
        }

        // Add body if present
        if let Some(body) = body {
            request = request.json(body);
        }

        // Execute request
        let response = request.send().await?;
        let response_text = response.text().await?;

        // First parse as a generic Value to check the error code
        let raw_value: serde_json::Value = serde_json::from_str(&response_text)?;

        let code = raw_value.get("code").and_then(|c| c.as_i64()).unwrap_or(0) as i32;

        // Handle lock conflict specially - data contains LockConflictDetail
        if code == ErrorCode::LockConflict as i32 {
            let msg = raw_value
                .get("msg")
                .and_then(|m| m.as_str())
                .unwrap_or("")
                .to_string();
            let detail: Option<LockConflictDetail> = raw_value
                .get("data")
                .and_then(|d| serde_json::from_value(d.clone()).ok());
            return Err(ApiError::LockConflict {
                message: msg,
                detail,
            });
        }

        // Parse as the expected response type
        let api_response: ApiResponse<R> = serde_json::from_str(&response_text)?;

        // Check response code
        if api_response.code != ErrorCode::Success as i32 {
            // Check if this is a credential error and invoke callback
            if let Some(error_code) = ErrorCode::from_code(api_response.code) {
                if error_code.is_credential_error() {
                    self.notify_credential_invalid().await;
                }
            }
            return Err(ApiError::from_response(api_response));
        }

        // Return data
        Ok(api_response.data.unwrap_or_default())
    }

    /// Send an API request with automatic token refresh
    pub async fn send<T, R>(
        &self,
        path: &str,
        method: Method,
        body: Option<&T>,
        options: RequestOptions,
    ) -> ApiResult<R>
    where
        T: Serialize + ?Sized,
        R: DeserializeOwned + Default,
    {
        match self
            .send_internal(path, method.clone(), body, options.clone())
            .await
        {
            Ok(result) => Ok(result),
            Err(ApiError::AccessTokenExpired) => {
                // Token expired, refresh and retry
                self.refresh_access_token().await?;
                self.send_internal(path, method, body, options).await
            }
            Err(e) => Err(e),
        }
    }

    /// Send a GET request
    pub async fn get<R>(&self, path: &str, options: RequestOptions) -> ApiResult<R>
    where
        R: DeserializeOwned + Default,
    {
        self.send::<(), R>(path, Method::GET, None, options).await
    }

    /// Send a POST request
    pub async fn post<T, R>(&self, path: &str, body: &T, options: RequestOptions) -> ApiResult<R>
    where
        T: Serialize,
        R: DeserializeOwned + Default,
    {
        self.send(path, Method::POST, Some(body), options).await
    }

    /// Send a PUT request
    pub async fn put<T, R>(&self, path: &str, body: &T, options: RequestOptions) -> ApiResult<R>
    where
        T: Serialize,
        R: DeserializeOwned + Default,
    {
        self.send(path, Method::PUT, Some(body), options).await
    }

    /// Send a DELETE request
    pub async fn delete<R>(&self, path: &str, options: RequestOptions) -> ApiResult<R>
    where
        R: DeserializeOwned + Default,
    {
        self.send::<(), R>(path, Method::DELETE, None, options)
            .await
    }

    /// Send a DELETE request with body
    pub async fn delete_with_body<T, R>(
        &self,
        path: &str,
        body: &T,
        options: RequestOptions,
    ) -> ApiResult<R>
    where
        T: Serialize,
        R: DeserializeOwned + Default,
    {
        self.send(path, Method::DELETE, Some(body), options).await
    }

    /// Send a PATCH request
    pub async fn patch<T, R>(&self, path: &str, body: &T, options: RequestOptions) -> ApiResult<R>
    where
        T: Serialize,
        R: DeserializeOwned + Default,
    {
        self.send(path, Method::PATCH, Some(body), options).await
    }
}
