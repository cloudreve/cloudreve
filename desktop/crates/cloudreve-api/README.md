# Cloudreve API Client for Rust

A comprehensive Rust client library for the Cloudreve API with automatic token refresh support.

## Features

- ✅ **Automatic Token Refresh**: Handles access token expiration and refresh automatically
- ✅ **Type-Safe**: Strongly typed API with comprehensive models
- ✅ **Async/Await**: Built on tokio for high-performance async operations
- ✅ **Error Handling**: Detailed error types for different API error conditions
- ✅ **Complete API Coverage**: Support for all major Cloudreve API endpoints
  - User authentication and management
  - File explorer operations (list, create, delete, move, etc.)
  - File upload and download
  - Workflow and task management
  - Site configuration

## Installation

Add this to your `Cargo.toml`:

```toml
[dependencies]
cloudreve-api = "0.1"
tokio = { version = "1.0", features = ["full"] }
```

## Quick Start

```rust
use cloudreve_api::{Client, ClientConfig};
use cloudreve_api::api::{UserApi, ExplorerApi};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Create client
    let config = ClientConfig::new("https://your-cloudreve-instance.com");
    let client = Client::new(config);

    // Login
    let login_response = client.login("user@example.com", "password").await?;
    println!("Logged in as: {}", login_response.user.nickname);

    // Set tokens for subsequent requests
    client.set_tokens_with_expiry(&login_response.token).await?;

    // Use the API - tokens will be automatically refreshed if needed
    let user = client.get_user_me().await?;
    println!("User ID: {}", user.id);

    // List files
    let params = cloudreve_api::models::explorer::ListFileService {
        uri: "/".to_string(),
        page: None,
        page_size: Some(50),
        order_by: None,
        order_direction: None,
        next_page_token: None,
    };

    let files = client.list_files(&params).await?;
    println!("Found {} files", files.files.len());

    for file in files.files {
        println!("- {} ({})", file.name, file.size);
    }

    Ok(())
}
```

## Authentication

### Password Login

```rust
let login_response = client.login("user@example.com", "password").await?;
client.set_tokens_with_expiry(&login_response.token).await?;
```

### Two-Factor Authentication

```rust
// First attempt will fail with 2FA required
match client.login("user@example.com", "password").await {
    Err(e) if e.to_string().contains("2FA") => {
        // Get OTP from user
        let otp = "123456";
        let session_id = "..."; // Extract from error
        let login_response = client.login_2fa(otp, session_id).await?;
        client.set_tokens_with_expiry(&login_response.token).await?;
    }
    Ok(response) => {
        client.set_tokens_with_expiry(&response.token).await?;
    }
    Err(e) => return Err(e.into()),
}
```

### Token Refresh

Tokens are automatically refreshed when they expire. The client handles this transparently:

```rust
// First request - uses existing token
let user = client.get_user_me().await?;

// ... token expires ...

// Next request - automatically refreshes token before making the request
let capacity = client.get_user_capacity().await?; // No error!
```

## Error Handling

The library provides detailed error types for different scenarios:

```rust
use cloudreve_api::error::{ApiError, ErrorCode};

match client.list_files(&params).await {
    Ok(files) => {
        // Process files
    }
    Err(ApiError::LoginRequired(msg)) => {
        // Need to login again
        println!("Login required: {}", msg);
    }
    Err(ApiError::LockConflict { message, detail }) => {
        // File is locked by another user/application
        println!("Lock conflict: {}", message);
        if let Some(d) = detail {
            println!("  Path: {}, Type: {}", d.path, d.lock_type);
        }
    }
    Err(ApiError::BatchError { message, aggregated_errors }) => {
        // Some operations in batch failed
        println!("Batch error: {}", message);
        if let Some(errors) = aggregated_errors {
            for (key, error) in errors {
                println!("  {}: {}", key, error);
            }
        }
    }
    Err(e) => {
        // Other errors
        println!("Error: {}", e);
    }
}
```

## API Examples

### File Operations

```rust
use cloudreve_api::models::explorer::*;

// Create a folder
let create_req = CreateFileService {
    uri: "/MyFolder".to_string(),
    file_type: "folder".to_string(),
    err_on_conflict: Some(false),
    metadata: None,
};
let folder = client.create_file(&create_req).await?;

// Upload a file (simplified)
let upload_req = UploadSessionRequest {
    uri: "/MyFolder/file.txt".to_string(),
    size: 1024,
    policy_id: "default".to_string(),
    last_modified: None,
    entity_type: None,
    metadata: None,
    mime_type: Some("text/plain".to_string()),
    encryption_supported: None,
};
let credential = client.create_upload_session(&upload_req).await?;

// Delete files
let delete_req = DeleteFileService {
    uris: vec!["/MyFolder/file.txt".to_string()],
    unlink: None,
    skip_soft_delete: None,
};
client.delete_files(&delete_req).await?;

// Move files
let move_req = MoveFileService {
    uris: vec!["/OldFolder/file.txt".to_string()],
    dst: "/NewFolder".to_string(),
    copy: Some(false),
};
client.move_files(&move_req).await?;

// Get file info
let info_req = GetFileInfoService {
    uri: Some("/MyFolder".to_string()),
    id: None,
    extended: Some(true),
    folder_summary: Some(true),
};
let file_info = client.get_file_info(&info_req).await?;
```

### Task Management

```rust
use cloudreve_api::models::workflow::*;
use cloudreve_api::api::WorkflowApi;

// Create archive task
let archive_req = ArchiveWorkflowService {
    src: vec!["/folder1".to_string(), "/file1.txt".to_string()],
    dst: "/archive.zip".to_string(),
    preferred_node_id: None,
    encoding: Some("utf-8".to_string()),
    password: None,
    file_mask: None,
};
let task = client.create_archive(&archive_req).await?;
println!("Task created: {}", task.id);

// List tasks
let list_req = ListTaskService {
    page_size: 20,
    category: ListTaskCategory::General,
    next_page_token: None,
};
let tasks = client.list_tasks(&list_req).await?;

// Monitor task
let task_status = client.get_task(&task.id).await?;
println!("Task status: {}", task_status.status);
```

### User Management

```rust
use cloudreve_api::api::UserApi;

// Get user info
let user = client.get_user_me().await?;
println!("User: {} ({})", user.nickname, user.email.unwrap_or_default());

// Get capacity
let capacity = client.get_user_capacity().await?;
println!("Storage: {} / {}", capacity.used, capacity.total);

// Update settings
let mut settings = PatchUserSetting::default();
settings.preferred_theme = Some("dark".to_string());
client.patch_user_settings(&settings).await?;
```

## Advanced Configuration

### Custom Timeout

```rust
let config = ClientConfig::new("https://your-cloudreve-instance.com")
    .with_timeout(60); // 60 seconds timeout
let client = Client::new(config);
```

### Request Options

```rust
// Skip batch error handling (get first error)
let opts = RequestOptions::new()
    .skip_batch_error();

// Include purchase ticket
let opts = RequestOptions::new()
    .with_purchase_ticket();

// No authentication
let opts = RequestOptions::new()
    .no_credential();
```

## Error Codes

The library handles these Cloudreve error codes:

- `0` - Success
- `203` - Continue
- `401` - Login required
- `403` - Permission denied
- `404` - Not found
- `40020` - Credential invalid
- `40069` - Incorrect password
- `40073` - Lock conflict (file locked by another user)
- `40076` - Stale version
- `40081` - Batch operation not fully completed
- `40083` - Purchase required
- `40087` - Domain not licensed
- `40088` - Anonymous access denied

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
