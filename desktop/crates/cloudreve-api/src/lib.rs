//! # Cloudreve API Client
//!
//! A comprehensive Rust client for the Cloudreve API with automatic token refresh support.
//!
//! ## Features
//!
//! - Automatic access token refresh when expired
//! - Comprehensive error handling
//! - Type-safe API methods
//! - Support for all Cloudreve API endpoints
//!
//! ## Example
//!
//! ```no_run
//! use cloudreve_api::{Client, ClientConfig};
//!
//! #[tokio::main]
//! async fn main() -> Result<(), Box<dyn std::error::Error>> {
//!     let config = ClientConfig::new("https://your-cloudreve-instance.com");
//!     let client = Client::new(config);
//!     
//!     // Login and get tokens
//!     let login_response = client.login("user@example.com", "password").await?;
//!     
//!     // Set tokens for subsequent requests
//!     client.set_tokens(
//!         login_response.token.access_token,
//!         login_response.token.refresh_token
//!     ).await;
//!     
//!     // Use the API - tokens will be automatically refreshed if needed
//!     let user = client.get_user_me().await?;
//!     println!("Hello, {}!", user.nickname);
//!     
//!     Ok(())
//! }
//! ```

pub mod api;
pub mod boolset;
pub mod client;
pub mod error;
pub mod models;

pub use boolset::Boolset;
pub use client::{Client, ClientConfig};
pub use error::{ApiError, ApiResult};
