//! Upyun upload implementation
//!
//! Upyun uses form-based upload with policy and authorization

use crate::uploader::chunk::ChunkInfo;
use crate::uploader::session::UploadSession;
use anyhow::{Context, Result, bail};
use bytes::Bytes;
use futures::Stream;
use reqwest::Client as HttpClient;
use reqwest::multipart::{Form, Part};
use serde::Deserialize;
use std::io;
use tracing::debug;

/// Upyun error response
#[derive(Debug, Deserialize)]
struct UpyunError {
    message: String,
    code: i32,
}

/// Upload to Upyun (single request, form-based) using generic stream
///
/// Note: Upyun doesn't support chunked uploads in the same way as other providers.
/// The entire file is uploaded in a single form submission.
pub async fn upload_chunk_generic<S>(
    http_client: &HttpClient,
    chunk: &ChunkInfo,
    stream: S,
    session: &UploadSession,
) -> Result<Option<String>>
where
    S: Stream<Item = Result<Bytes, io::Error>> + Send + Sync + Unpin + 'static,
{
    // Upyun only supports single-chunk uploads
    if chunk.index != 0 {
        bail!(
            "Upyun only supports single-chunk uploads (got chunk {})",
            chunk.index
        );
    }

    let url = session.upload_url().context("no upload URL for Upyun")?;

    let policy = session
        .upload_policy()
        .context("no upload policy for Upyun")?;

    let credential = session.credential_string();

    debug!(
        target: "uploader::upyun",
        size = chunk.size,
        url = %url,
        "Uploading file to Upyun (streaming)"
    );

    // Build multipart form with streaming body
    // Use Part::stream to create a streaming file part
    let body = reqwest::Body::wrap_stream(stream);
    let file_part = Part::stream_with_length(body, chunk.size)
        .file_name("file")
        .mime_str("application/octet-stream")
        .context("failed to set MIME type for file part")?;

    let mut form = Form::new()
        .text("policy", policy.to_string())
        .text("authorization", credential.to_string())
        .part("file", file_part);

    // Add MIME type if available
    if let Some(mime) = session.mime_type() {
        form = form.text("content-type", mime.to_string());
    }

    let response = http_client
        .post(url)
        .multipart(form)
        .send()
        .await
        .context("failed to upload to Upyun")?;

    if !response.status().is_success() {
        let status = response.status();
        let body = response.text().await.unwrap_or_default();

        // Try to parse Upyun error
        if let Ok(error) = serde_json::from_str::<UpyunError>(&body) {
            bail!("Upyun error ({}): {}", error.code, error.message);
        }

        bail!("Upyun upload failed: HTTP {}: {}", status, body);
    }

    Ok(None)
}
