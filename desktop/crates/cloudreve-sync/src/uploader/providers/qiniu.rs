//! Qiniu Cloud Storage upload implementation

use crate::uploader::chunk::ChunkInfo;
use crate::uploader::session::UploadSession;
use anyhow::{Context, Result, bail};
use bytes::Bytes;
use futures::Stream;
use reqwest::{Body, Client as HttpClient};
use serde::{Deserialize, Serialize};
use std::io;
use tracing::debug;

/// Qiniu chunk upload response
#[derive(Debug, Deserialize)]
struct QiniuChunkResponse {
    etag: String,
    #[serde(default)]
    #[allow(dead_code)]
    md5: String,
}

/// Qiniu error response
#[derive(Debug, Deserialize)]
struct QiniuError {
    error: String,
}

/// Qiniu completion request part info
#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
struct QiniuPartInfo {
    etag: String,
    part_number: usize,
}

/// Qiniu completion request
#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
struct QiniuCompleteRequest {
    parts: Vec<QiniuPartInfo>,
    #[serde(skip_serializing_if = "Option::is_none")]
    mime_type: Option<String>,
}

/// Upload chunk to Qiniu using generic stream
pub async fn upload_chunk_generic<S>(
    http_client: &HttpClient,
    chunk: &ChunkInfo,
    stream: S,
    session: &UploadSession,
) -> Result<Option<String>>
where
    S: Stream<Item = Result<Bytes, io::Error>> + Send + Sync + Unpin + 'static,
{
    let base_url = session.upload_url().context("no upload URL for Qiniu")?;

    // Qiniu uses 1-based part numbers in URL
    let url = format!("{}/{}", base_url, chunk.index + 1);
    let credential = session.credential_string();

    debug!(
        target: "uploader::qiniu",
        chunk = chunk.index,
        size = chunk.size,
        url = %url,
        "Uploading chunk to Qiniu (streaming)"
    );

    let body = Body::wrap_stream(stream);

    let response = http_client
        .put(&url)
        .header("Content-Type", "application/octet-stream")
        .header("Content-Length", chunk.size)
        .header("Authorization", format!("UpToken {}", credential))
        .body(body)
        .send()
        .await
        .with_context(|| format!("failed to upload chunk {} to Qiniu", chunk.index))?;

    if !response.status().is_success() {
        let status = response.status();
        let body = response.text().await.unwrap_or_default();

        // Try to parse Qiniu error
        if let Ok(error) = serde_json::from_str::<QiniuError>(&body) {
            bail!("Qiniu error: {}", error.error);
        }

        bail!(
            "Qiniu chunk {} upload failed: HTTP {}: {}",
            chunk.index,
            status,
            body
        );
    }

    // Parse response to get ETag
    let chunk_response: QiniuChunkResponse = response
        .json()
        .await
        .context("failed to parse Qiniu response")?;

    Ok(Some(chunk_response.etag))
}

/// Complete Qiniu multipart upload
pub async fn complete_upload(http_client: &HttpClient, session: &UploadSession) -> Result<()> {
    let url = session
        .upload_url()
        .context("no upload URL for Qiniu completion")?;

    let credential = session.credential_string();

    // Build completion request
    let parts: Vec<QiniuPartInfo> = session
        .chunk_progress
        .iter()
        .filter_map(|c| {
            c.etag.as_ref().map(|etag| QiniuPartInfo {
                etag: etag.clone(),
                part_number: c.index + 1,
            })
        })
        .collect();

    let request = QiniuCompleteRequest {
        parts,
        mime_type: session.mime_type().map(|s| s.to_string()),
    };

    debug!(
        target: "uploader::qiniu",
        url = %url,
        parts = request.parts.len(),
        "Completing Qiniu multipart upload"
    );

    let response = http_client
        .post(url)
        .header("Content-Type", "application/json")
        .header("Authorization", format!("UpToken {}", credential))
        .json(&request)
        .send()
        .await
        .context("failed to complete Qiniu upload")?;

    if !response.status().is_success() {
        let status = response.status();
        let body = response.text().await.unwrap_or_default();

        // Try to parse Qiniu error
        if let Ok(error) = serde_json::from_str::<QiniuError>(&body) {
            bail!("Qiniu completion error: {}", error.error);
        }

        bail!("Qiniu completion failed: HTTP {}: {}", status, body);
    }

    Ok(())
}
