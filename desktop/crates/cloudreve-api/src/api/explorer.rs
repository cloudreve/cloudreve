use crate::client::{Client, RequestOptions, CR_HEADER_PREFIX};
use crate::error::ApiResult;
use crate::models::common::ListAllRes;
use crate::models::explorer::*;
use async_trait::async_trait;
use bytes::Bytes;
use reqwest::Body;

/// Decode time flow string (for obfuscated thumbnail URLs)
fn decode_time_flow_string(str: &str, time_now: i64) -> ApiResult<String> {
    // Try with current time
    if let Ok(result) = decode_time_flow_string_time(str, time_now) {
        return Ok(result);
    }

    // Try with time - 1000
    if let Ok(result) = decode_time_flow_string_time(str, time_now - 1000) {
        return Ok(result);
    }

    // Try with time + 1000
    if let Ok(result) = decode_time_flow_string_time(str, time_now + 1000) {
        return Ok(result);
    }

    Err(crate::error::ApiError::Other(
        "Failed to decode time flow string".to_string(),
    ))
}

/// Decode time flow string time (for obfuscated thumbnail URLs)
fn decode_time_flow_string_time(str: &str, time_now: i64) -> ApiResult<String> {
    let mut time_now = time_now / 1000;
    let time_now_backup = time_now;

    // Extract time digits
    let mut time_digits: Vec<i64> = Vec::new();

    if str.is_empty() {
        return Ok(String::new());
    }

    while time_now > 0 {
        time_digits.push(time_now % 10);
        time_now /= 10;
    }

    if time_digits.is_empty() {
        return Err(crate::error::ApiError::Other(
            "Invalid time value".to_string(),
        ));
    }

    // Convert string to character array
    let chars: Vec<char> = str.chars().collect();
    let mut res: Vec<char> = chars.clone();
    let mut secret: Vec<char> = chars.clone();

    let mut add = secret.len() % 2 == 0;
    let mut time_digit_index = ((secret.len() - 1) % time_digits.len()) as i64;
    let l = secret.len();

    for pos in 0..l {
        let res_index = l - 1 - pos;
        let mut new_index = res_index as i64;

        if add {
            new_index = new_index + time_digits[time_digit_index as usize] * time_digit_index;
        } else {
            new_index = 2 * time_digit_index * time_digits[time_digit_index as usize] - new_index;
        }

        if new_index < 0 {
            new_index = new_index * -1;
        }

        new_index = new_index % secret.len() as i64;
        let new_index_usize = new_index as usize;

        res[res_index] = secret[new_index_usize];

        // Swap elements in secret
        let a = secret[res_index];
        let b = secret[new_index_usize];
        secret[new_index_usize] = a;
        secret[res_index] = b;

        // Remove last element from secret
        secret.pop();

        add = !add;

        // Decrement timeDigitIndex
        time_digit_index -= 1;
        if time_digit_index < 0 {
            time_digit_index = time_digits.len() as i64 - 1;
        }
    }

    // Convert result back to string
    let res_str: String = res.iter().collect();

    // Validate the result
    let res_sep: Vec<&str> = res_str.split('|').collect();

    if res_sep.is_empty() || res_sep[0] != time_now_backup.to_string() {
        return Err(crate::error::ApiError::Other(
            "Invalid time flow string".to_string(),
        ));
    }

    // Return the part after the first "|"
    let prefix_len = res_sep[0].len() + 1; // +1 for the "|"
    if prefix_len <= res_str.len() {
        Ok(res_str[prefix_len..].to_string())
    } else {
        Ok(String::new())
    }
}

/// File explorer API methods
#[async_trait]
pub trait ExplorerApi {
    /// List files in a directory
    async fn list_files(&self, params: &ListFileService) -> ApiResult<ListResponse>;

    /// Get file thumbnail
    async fn get_file_thumb(
        &self,
        path: &str,
        context_hint: Option<&str>,
    ) -> ApiResult<FileThumbResponse>;

    /// Get file information
    async fn get_file_info(&self, params: &GetFileInfoService) -> ApiResult<FileResponse>;

    /// Create a new file or folder
    async fn create_file(&self, request: &CreateFileService) -> ApiResult<FileResponse>;

    /// Delete files
    async fn delete_files(&self, request: &DeleteFileService) -> ApiResult<()>;

    /// Rename a file
    async fn rename_file(&self, request: &RenameFileService) -> ApiResult<FileResponse>;

    /// Move files
    async fn move_files(&self, request: &MoveFileService) -> ApiResult<()>;

    /// Restore files from trash
    async fn restore_files(&self, request: &DeleteFileService) -> ApiResult<()>;

    /// Patch file metadata
    async fn patch_metadata(&self, request: &PatchMetadataService) -> ApiResult<()>;

    /// Get file entity URL
    async fn get_file_url(&self, request: &FileURLService) -> ApiResult<FileURLResponse>;

    /// Unlock files
    async fn unlock_files(&self, request: &UnlockFileService) -> ApiResult<()>;

    /// Set current version
    async fn set_current_version(&self, request: &VersionControlService) -> ApiResult<()>;

    /// Delete version
    async fn delete_version(&self, request: &VersionControlService) -> ApiResult<()>;

    /// Update file content
    async fn update_file(&self, params: &FileUpdateService, data: Bytes)
        -> ApiResult<FileResponse>;

    /// Get storage policy options
    async fn get_storage_policy_options(&self) -> ApiResult<Vec<StoragePolicy>>;

    /// Mount storage policy
    async fn mount_storage_policy(
        &self,
        request: &MountPolicyService,
    ) -> ApiResult<Vec<StoragePolicy>>;

    /// Set file permissions
    async fn set_permissions(&self, request: &SetPermissionService) -> ApiResult<()>;

    /// Create upload session
    async fn create_upload_session(
        &self,
        request: &UploadSessionRequest,
    ) -> ApiResult<UploadCredential>;

    /// Upload chunk
    async fn upload_chunk(
        &self,
        session_id: &str,
        chunk_index: usize,
        data: Bytes,
    ) -> ApiResult<()>;

    /// Upload chunk using streaming body (for large chunks)
    async fn upload_chunk_stream(
        &self,
        session_id: &str,
        chunk_index: usize,
        content_length: u64,
        body: Body,
    ) -> ApiResult<()>;

    /// Delete upload session
    async fn delete_upload_session(&self, request: &DeleteUploadSessionService) -> ApiResult<()>;

    /// Complete S3-like upload
    async fn complete_s3_upload(
        &self,
        policy_type: &str,
        session_id: &str,
        session_key: &str,
    ) -> ApiResult<()>;

    /// Complete OneDrive upload
    async fn complete_onedrive_upload(&self, session_id: &str, session_key: &str) -> ApiResult<()>;
}

#[async_trait]
pub trait ExplorerApiExt {
    async fn list_files_all(
        &self,
        previous_response: Option<&ListAllRes<ListResponse>>,
        uri: &str,
        page_size: i32,
    ) -> ApiResult<ListAllRes<ListResponse>>;
}

#[async_trait]
impl ExplorerApiExt for Client {
    async fn list_files_all(
        &self,
        previous_response: Option<&ListAllRes<ListResponse>>,
        uri: &str,
        page_size: i32,
    ) -> ApiResult<ListAllRes<ListResponse>> {
        const MIN_PAGE_SIZE: i32 = 1;

        // Extract pagination info from previous response
        let (page, next_token) = if let Some(prev) = previous_response {
            let prev_pagination = &prev.res.pagination;

            // Determine next page parameters based on pagination type
            if prev_pagination.next_token.is_some() {
                // Token-based pagination
                (None, prev_pagination.next_token.clone())
            } else if prev_pagination.total_items.is_some() {
                // Page-based pagination
                let current_page = prev_pagination.page;
                (Some(current_page + 1), None)
            } else {
                // No pagination info, start fresh
                (None, None)
            }
        } else {
            // First page
            (None, None)
        };

        // Call list_files with current pagination state
        let params = ListFileService {
            uri: uri.to_string(),
            page,
            page_size: Some(page_size),
            order_by: None,
            order_direction: None,
            next_page_token: next_token,
        };

        let response = self.list_files(&params).await?;

        // Determine if there's more data to load
        let page_size_val = if response.pagination.page_size > 0 {
            response.pagination.page_size
        } else {
            MIN_PAGE_SIZE
        };

        let has_more = if response.pagination.next_token.is_some() {
            // Token-based: more data if next_token exists
            true
        } else if let Some(total_items) = response.pagination.total_items {
            // Page-based: calculate if there are more pages
            let total_pages = (total_items as f64 / page_size_val as f64).ceil() as i32;
            let current_page = response.pagination.page;
            current_page + 1 < total_pages
        } else {
            // No pagination info, assume no more data
            false
        };

        Ok(ListAllRes {
            res: response,
            more: has_more,
        })
    }
}

#[async_trait]
impl ExplorerApi for Client {
    async fn list_files(&self, params: &ListFileService) -> ApiResult<ListResponse> {
        // Build query string
        let mut query_params = vec![format!("uri={}", urlencoding::encode(&params.uri))];

        if let Some(page) = params.page {
            query_params.push(format!("page={}", page));
        }
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

        let query = format!("?{}", query_params.join("&"));

        self.get(
            &format!("/file{}", query),
            RequestOptions::new().with_purchase_ticket(),
        )
        .await
    }

    async fn get_file_thumb(
        &self,
        path: &str,
        _context_hint: Option<&str>,
    ) -> ApiResult<FileThumbResponse> {
        let query = format!("?uri={}", urlencoding::encode(path));

        // TODO: Add context hint header support if needed
        let mut response: FileThumbResponse = self
            .get(
                &format!("/file/thumb{}", query),
                RequestOptions::new().with_purchase_ticket(),
            )
            .await?;

        if response.obfuscated {
            // Decode the obfuscated URL
            let time_now_sec = std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .map_err(|e| crate::error::ApiError::Other(format!("System time error: {}", e)))?
                .as_secs() as i64;

            response.url = decode_time_flow_string(&response.url, time_now_sec)?;
        }

        Ok(response)
    }

    async fn get_file_info(&self, params: &GetFileInfoService) -> ApiResult<FileResponse> {
        let mut query_params = vec![];

        if let Some(uri) = &params.uri {
            query_params.push(format!("uri={}", urlencoding::encode(uri)));
        }
        if let Some(id) = &params.id {
            query_params.push(format!("id={}", id));
        }
        if let Some(extended) = params.extended {
            query_params.push(format!("extended={}", extended));
        }
        if let Some(folder_summary) = params.folder_summary {
            query_params.push(format!("folder_summary={}", folder_summary));
        }

        let query = if query_params.is_empty() {
            String::new()
        } else {
            format!("?{}", query_params.join("&"))
        };

        self.get(
            &format!("/file/info{}", query),
            RequestOptions::new().with_purchase_ticket(),
        )
        .await
    }

    async fn create_file(&self, request: &CreateFileService) -> ApiResult<FileResponse> {
        self.post(
            "/file/create",
            request,
            RequestOptions::new().with_purchase_ticket(),
        )
        .await
    }

    async fn delete_files(&self, request: &DeleteFileService) -> ApiResult<()> {
        let opts = if request.uris.len() == 1 {
            RequestOptions::new()
                .with_purchase_ticket()
                .skip_batch_error()
        } else {
            RequestOptions::new().with_purchase_ticket()
        };

        self.delete_with_body("/file", request, opts).await
    }

    async fn rename_file(&self, request: &RenameFileService) -> ApiResult<FileResponse> {
        self.post(
            "/file/rename",
            request,
            RequestOptions::new().with_purchase_ticket(),
        )
        .await
    }

    async fn move_files(&self, request: &MoveFileService) -> ApiResult<()> {
        let opts = if request.uris.len() == 1 {
            RequestOptions::new()
                .with_purchase_ticket()
                .skip_batch_error()
        } else {
            RequestOptions::new().with_purchase_ticket()
        };

        self.post("/file/move", request, opts).await
    }

    async fn restore_files(&self, request: &DeleteFileService) -> ApiResult<()> {
        let opts = if request.uris.len() == 1 {
            RequestOptions::new()
                .with_purchase_ticket()
                .skip_batch_error()
        } else {
            RequestOptions::new().with_purchase_ticket()
        };

        self.post("/file/restore", request, opts).await
    }

    async fn patch_metadata(&self, request: &PatchMetadataService) -> ApiResult<()> {
        let opts = if request.uris.len() == 1 {
            RequestOptions::new()
                .with_purchase_ticket()
                .skip_batch_error()
        } else {
            RequestOptions::new().with_purchase_ticket()
        };

        self.patch("/file/metadata", request, opts).await
    }

    async fn get_file_url(&self, request: &FileURLService) -> ApiResult<FileURLResponse> {
        let opts = if request.uris.len() == 1 {
            RequestOptions::new()
                .with_purchase_ticket()
                .skip_batch_error()
        } else {
            RequestOptions::new().with_purchase_ticket()
        };

        self.post("/file/url", request, opts).await
    }

    async fn unlock_files(&self, request: &UnlockFileService) -> ApiResult<()> {
        self.delete_with_body(
            "/file/lock",
            request,
            RequestOptions::new().skip_lock_conflict(),
        )
        .await
    }

    async fn set_current_version(&self, request: &VersionControlService) -> ApiResult<()> {
        self.post(
            "/file/version/current",
            request,
            RequestOptions::new().with_purchase_ticket(),
        )
        .await
    }

    async fn delete_version(&self, request: &VersionControlService) -> ApiResult<()> {
        self.delete_with_body(
            "/file/version",
            request,
            RequestOptions::new().with_purchase_ticket(),
        )
        .await
    }

    async fn update_file(
        &self,
        params: &FileUpdateService,
        data: Bytes,
    ) -> ApiResult<FileResponse> {
        // Build query string
        let mut query_params = vec![format!("uri={}", urlencoding::encode(&params.uri))];

        if let Some(previous) = &params.previous {
            query_params.push(format!("previous={}", previous));
        }

        let query = format!("?{}", query_params.join("&"));

        // We need to use a custom request here since we're sending binary data
        let url = self.build_url(&format!("/file/content{}", query));
        let token = self.get_access_token().await?;

        let response = self
            .http_client
            .put(&url)
            .header("Authorization", format!("Bearer {}", token))
            .header("Content-Type", "application/octet-stream")
            .body(data)
            .send()
            .await?;

        let api_response: crate::error::ApiResponse<FileResponse> = response.json().await?;

        if api_response.code != 0 {
            return Err(crate::error::ApiError::from_response(api_response));
        }

        api_response.data.ok_or_else(|| {
            crate::error::ApiError::Other("API returned success but no data".to_string())
        })
    }

    async fn get_storage_policy_options(&self) -> ApiResult<Vec<StoragePolicy>> {
        self.get("/user/setting/policies", RequestOptions::new())
            .await
    }

    async fn mount_storage_policy(
        &self,
        request: &MountPolicyService,
    ) -> ApiResult<Vec<StoragePolicy>> {
        self.patch(
            "/file/policy",
            request,
            RequestOptions::new().with_purchase_ticket(),
        )
        .await
    }

    async fn set_permissions(&self, request: &SetPermissionService) -> ApiResult<()> {
        let opts = if request.uris.len() == 1 {
            RequestOptions::new()
                .with_purchase_ticket()
                .skip_batch_error()
        } else {
            RequestOptions::new().with_purchase_ticket()
        };

        self.post("/file/permission", request, opts).await
    }

    async fn create_upload_session(
        &self,
        request: &UploadSessionRequest,
    ) -> ApiResult<UploadCredential> {
        self.put(
            "/file/upload",
            request,
            RequestOptions::new().with_purchase_ticket(),
        )
        .await
    }

    async fn upload_chunk(
        &self,
        session_id: &str,
        chunk_index: usize,
        data: Bytes,
    ) -> ApiResult<()> {
        let url = self.build_url(&format!("/file/upload/{}/{}", session_id, chunk_index));
        let token = self.get_access_token().await?;

        let response = self
            .http_client
            .post(&url)
            .header("Authorization", format!("Bearer {}", token))
            .header("Content-Type", "application/octet-stream")
            .body(data)
            .send()
            .await?;

        let api_response: crate::error::ApiResponse<UploadCredential> = response.json().await?;

        if api_response.code != 0 {
            return Err(crate::error::ApiError::from_response(api_response));
        }

        Ok(())
    }

    async fn upload_chunk_stream(
        &self,
        session_id: &str,
        chunk_index: usize,
        content_length: u64,
        body: Body,
    ) -> ApiResult<()> {
        let url = self.build_url(&format!("/file/upload/{}/{}", session_id, chunk_index));
        let token = self.get_access_token().await?;

        let response = self
            .http_client
            .post(&url)
            .header("Authorization", format!("Bearer {}", token))
            .header("Content-Type", "application/octet-stream")
            .header("Content-Length", content_length)
            .body(body)
            .send()
            .await?;

        let api_response: crate::error::ApiResponse<()> = response.json().await?;

        if api_response.code != 0 {
            return Err(crate::error::ApiError::from_response(api_response));
        }

        Ok(())
    }

    async fn delete_upload_session(&self, request: &DeleteUploadSessionService) -> ApiResult<()> {
        self.delete_with_body(
            "/file/upload",
            request,
            RequestOptions::new().with_purchase_ticket(),
        )
        .await
    }

    async fn complete_s3_upload(
        &self,
        policy_type: &str,
        session_id: &str,
        session_key: &str,
    ) -> ApiResult<()> {
        self.get(
            &format!("/callback/{}/{}/{}", policy_type, session_id, session_key),
            RequestOptions::new(),
        )
        .await
    }

    async fn complete_onedrive_upload(&self, session_id: &str, session_key: &str) -> ApiResult<()> {
        self.post::<(), ()>(
            &format!("/callback/onedrive/{}/{}", session_id, session_key),
            &(),
            RequestOptions::new(),
        )
        .await
    }
}

/// A subscription handle for file events SSE stream
pub struct FileEventSubscription {
    response: reqwest::Response,
    buffer: String,
}

impl FileEventSubscription {
    /// Create a new subscription from a response
    fn new(response: reqwest::Response) -> Self {
        Self {
            response,
            buffer: String::new(),
        }
    }

    /// Receive the next file event from the stream.
    /// Returns None when the stream ends.
    pub async fn next_event(&mut self) -> ApiResult<Option<FileEvent>> {
        loop {
            // Try to parse a complete event from the buffer
            if let Some(event) = self.try_parse_event()? {
                return Ok(Some(event));
            }

            // Need more data from the stream
            match self.response.chunk().await {
                Ok(Some(chunk)) => {
                    let text = String::from_utf8_lossy(&chunk);
                    self.buffer.push_str(&text);
                }
                Ok(None) => {
                    // Stream ended
                    // Try to parse any remaining data
                    if !self.buffer.is_empty() {
                        if let Some(event) = self.try_parse_event()? {
                            return Ok(Some(event));
                        }
                    }
                    return Ok(None);
                }
                Err(e) => {
                    return Err(crate::error::ApiError::SseStreamError(e.to_string()));
                }
            }
        }
    }

    /// Try to parse a complete SSE event from the buffer
    fn try_parse_event(&mut self) -> ApiResult<Option<FileEvent>> {
        // SSE events are separated by double newlines
        // Format:
        // event:eventname
        // data:payload
        //
        // (blank line)

        // Find the end of an event (double newline)
        let event_end = if let Some(pos) = self.buffer.find("\n\n") {
            pos + 2
        } else if let Some(pos) = self.buffer.find("\r\n\r\n") {
            pos + 4
        } else {
            return Ok(None);
        };

        // Extract the event block
        let event_block = self.buffer[..event_end].to_string();
        self.buffer = self.buffer[event_end..].to_string();

        // Parse the event
        let mut event_type: Option<&str> = None;
        let mut data: Option<&str> = None;

        for line in event_block.lines() {
            if let Some(rest) = line.strip_prefix("event:") {
                event_type = Some(rest.trim());
            } else if let Some(rest) = line.strip_prefix("data:") {
                data = Some(rest.trim());
            }
        }

        // Match on event type
        match event_type {
            Some("resumed") => Ok(Some(FileEvent::Resumed)),
            Some("subscribed") => Ok(Some(FileEvent::Subscribed)),
            Some("keep-alive") | Some("keepalive") => Ok(Some(FileEvent::KeepAlive)),
            Some("reconnect-required") => Ok(Some(FileEvent::ReconnectRequired)),
            Some("event") => {
                if let Some(data_str) = data {
                    // Skip nil data
                    if data_str == "<nil>" || data_str.is_empty() {
                        // This shouldn't happen for "event" type, but handle gracefully
                        return Ok(None);
                    }
                    // Try to parse as array first (batch of events)
                    if let Ok(event_data_list) =
                        serde_json::from_str::<Vec<FileEventData>>(data_str)
                    {
                        if event_data_list.is_empty() {
                            return Ok(None);
                        }
                        return Ok(Some(FileEvent::Event(event_data_list)));
                    }
                    // Fall back to parsing as single event for backwards compatibility
                    let event_data: FileEventData = serde_json::from_str(data_str)?;
                    Ok(Some(FileEvent::Event(vec![event_data])))
                } else {
                    Ok(None)
                }
            }
            _ => {
                // Unknown event type, skip it
                Ok(None)
            }
        }
    }
}

/// File events SSE API methods
#[async_trait]
pub trait FileEventsApi {
    /// Subscribe to file events for a given URI.
    ///
    /// This connects to the SSE endpoint at /v4/file/events with the provided URI.
    /// Returns a subscription handle that can be used to receive events.
    ///
    /// # Arguments
    /// * `uri` - The filesystem URI to watch for events (e.g., "cloudreve://my-drive/")
    ///
    /// # Returns
    /// * `Ok(FileEventSubscription)` - A handle to receive events from
    /// * `Err(ApiError::SseNotUpgraded)` - If the server returned an error instead of SSE stream
    /// * `Err(ApiError::RequestError)` - If the HTTP request failed
    ///
    /// # Example
    /// ```no_run
    /// use cloudreve_api::{Client, ClientConfig};
    /// use cloudreve_api::api::explorer::FileEventsApi;
    ///
    /// async fn watch_events(client: &Client) -> Result<(), Box<dyn std::error::Error>> {
    ///     let mut subscription = client.subscribe_file_events("cloudreve://my-drive/").await?;
    ///
    ///     while let Some(event) = subscription.next_event().await? {
    ///         match event {
    ///             cloudreve_api::models::explorer::FileEvent::Event(data) => {
    ///                 println!("File event: {:?} on {}", data.event_type, data.from);
    ///             }
    ///             _ => {}
    ///         }
    ///     }
    ///
    ///     Ok(())
    /// }
    /// ```
    async fn subscribe_file_events(&self, uri: &str) -> ApiResult<FileEventSubscription>;
}

#[async_trait]
impl FileEventsApi for Client {
    async fn subscribe_file_events(&self, uri: &str) -> ApiResult<FileEventSubscription> {
        let query = format!("?uri={}", urlencoding::encode(uri));
        let url = self.build_url(&format!("/file/events{}", query));
        let token = self.get_access_token().await?;

        let response = self
            .http_client
            .get(&url)
            .header(
                format!("{}Client-Id", CR_HEADER_PREFIX),
                self.config.client_id.clone(),
            )
            .header("Authorization", format!("Bearer {}", token))
            .header("Accept", "text/event-stream")
            .send()
            .await?;

        // Check if we got an SSE response by looking at content-type
        let content_type = response
            .headers()
            .get(reqwest::header::CONTENT_TYPE)
            .and_then(|v| v.to_str().ok())
            .unwrap_or("");

        if content_type.contains("text/event-stream") {
            // Successfully upgraded to SSE
            Ok(FileEventSubscription::new(response))
        } else {
            // Server returned a regular response (likely an error)
            // Try to parse it as an API error response
            let response_text = response.text().await?;

            // Try to parse as API response
            if let Ok(api_response) =
                serde_json::from_str::<crate::error::ApiResponse<()>>(&response_text)
            {
                if api_response.code != 0 {
                    return Err(crate::error::ApiError::SseNotUpgraded {
                        code: api_response.code,
                        message: api_response.msg,
                    });
                }
            }

            // If we couldn't parse it, return a generic error
            Err(crate::error::ApiError::SseNotUpgraded {
                code: -1,
                message: format!("Unexpected response: {}", response_text),
            })
        }
    }
}
