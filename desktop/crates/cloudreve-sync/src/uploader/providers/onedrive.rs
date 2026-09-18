//! OneDrive upload implementation

use crate::uploader::chunk::ChunkInfo;
use crate::uploader::session::UploadSession;
use anyhow::{Context, Result, bail};
use bytes::Bytes;
use cloudreve_api::Client as CrClient;
use cloudreve_api::api::ExplorerApi;
use futures::Stream;
use reqwest::{Body, Client as HttpClient};
use serde::Deserialize;
use std::io;
use std::sync::Arc;
use tracing::{debug, warn};

/// OneDrive chunk upload response
#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
#[allow(dead_code)]
struct OneDriveChunkResponse {
    #[serde(default)]
    expiration_date_time: Option<String>,
    #[serde(default)]
    next_expected_ranges: Vec<String>,
}

/// OneDrive error response
#[derive(Debug, Deserialize)]
struct OneDriveError {
    error: OneDriveErrorDetails,
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
struct OneDriveErrorDetails {
    code: String,
    message: String,
    #[serde(default)]
    innererror: Option<OneDriveInnerError>,
    #[serde(default)]
    #[allow(dead_code)]
    retry_after_seconds: Option<u32>,
}

#[derive(Debug, Deserialize)]
struct OneDriveInnerError {
    code: String,
}

/// Upload chunk to OneDrive using generic stream
pub async fn upload_chunk_generic<S>(
    http_client: &HttpClient,
    chunk: &ChunkInfo,
    stream: S,
    session: &UploadSession,
) -> Result<Option<String>>
where
    S: Stream<Item = Result<Bytes, io::Error>> + Send + Sync + Unpin + 'static,
{
    // OneDrive doesn't support empty files
    if session.file_size == 0 {
        bail!("OneDrive does not support empty file uploads");
    }

    let url = session.upload_url().context("no upload URL for OneDrive")?;

    // Calculate byte range
    let range_start = chunk.offset;
    let range_end = chunk.offset + chunk.size - 1;
    let content_range = format!("bytes {}-{}/{}", range_start, range_end, session.file_size);

    debug!(
        target: "uploader::onedrive",
        chunk = chunk.index,
        range = %content_range,
        "Uploading chunk to OneDrive (streaming)"
    );

    let body = Body::wrap_stream(stream);

    let response = http_client
        .put(url)
        .header("Content-Type", "application/octet-stream")
        .header("Content-Length", chunk.size)
        .header("Content-Range", &content_range)
        .body(body)
        .send()
        .await
        .with_context(|| format!("failed to upload chunk {} to OneDrive", chunk.index))?;

    let status = response.status();

    if status.is_success() || status.as_u16() == 202 {
        // Success or Accepted (more chunks needed)
        return Ok(None);
    }

    // Parse error response
    let body = response.text().await.unwrap_or_default();

    if let Ok(error) = serde_json::from_str::<OneDriveError>(&body) {
        // Check for fragment overlap error
        if let Some(ref inner) = error.error.innererror {
            if inner.code == "fragmentOverlap" {
                // This is a recoverable error - the chunk was already uploaded
                warn!(
                    target: "uploader::onedrive",
                    chunk = chunk.index,
                    "Fragment overlap detected, chunk may be already uploaded"
                );
                bail!("OneDrive chunk overlap: {}", error.error.message);
            }
        }

        bail!(
            "OneDrive error ({}): {}",
            error.error.code,
            error.error.message
        );
    }

    bail!(
        "OneDrive chunk {} upload failed: HTTP {}: {}",
        chunk.index,
        status,
        body
    )
}

/// Query OneDrive session status to get next expected range
#[allow(dead_code)]
pub async fn query_session_status(
    http_client: &HttpClient,
    session: &UploadSession,
) -> Result<Vec<String>> {
    let url = session.upload_url().context("no upload URL for OneDrive")?;

    let response = http_client
        .get(url)
        .send()
        .await
        .context("failed to query OneDrive session status")?;

    if !response.status().is_success() {
        let status = response.status();
        let body = response.text().await.unwrap_or_default();
        bail!(
            "failed to query OneDrive session: HTTP {}: {}",
            status,
            body
        );
    }

    let chunk_response: OneDriveChunkResponse = response
        .json()
        .await
        .context("failed to parse OneDrive session response")?;

    Ok(chunk_response.next_expected_ranges)
}

/// Complete OneDrive upload by calling Cloudreve callback
pub async fn complete_upload(cr_client: &Arc<CrClient>, session: &UploadSession) -> Result<()> {
    debug!(
        target: "uploader::onedrive",
        session_id = session.session_id(),
        "Completing OneDrive upload"
    );

    cr_client
        .complete_onedrive_upload(session.session_id(), session.callback_secret())
        .await
        .context("OneDrive upload callback failed")?;

    Ok(())
}
