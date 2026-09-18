//! Upload session management and persistence

use crate::uploader::ChunkProgress;
use crate::uploader::providers::PolicyType;
use chrono::Utc;
use cloudreve_api::models::explorer::{EncryptMetadata, UploadCredential};
use serde::{Deserialize, Serialize};
/// Persisted upload session state
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UploadSession {
    /// Unique session ID (from database)
    pub id: String,
    /// Associated task ID
    pub task_id: String,
    /// Drive ID
    pub drive_id: String,
    /// Local file path
    pub local_path: String,
    /// Remote URI (cloudreve path)
    pub remote_uri: String,
    /// Total file size
    pub file_size: u64,
    /// Chunk size for this upload
    pub chunk_size: u64,
    /// Policy type
    #[serde(with = "policy_type_serde")]
    policy_type: PolicyType,
    /// Upload credential from Cloudreve
    credential: UploadCredential,
    // Whether relay upload is enabled
    relay: bool,
    /// Per-chunk progress
    pub chunk_progress: Vec<ChunkProgress>,
    /// Encryption metadata (if encrypted)
    pub encrypt_metadata: Option<EncryptMetadata>,
    /// Session expiration timestamp
    pub expires_at: i64,
    /// Creation timestamp
    pub created_at: i64,
    /// Last update timestamp
    pub updated_at: i64,
}

impl UploadSession {
    /// Create a new upload session from a credential response
    pub fn new(
        task_id: String,
        drive_id: String,
        local_path: String,
        remote_uri: String,
        file_size: u64,
        credential: UploadCredential,
    ) -> Self {
        let chunk_size = credential.chunk_size as u64;
        let num_chunks = Self::calculate_num_chunks(file_size, chunk_size);
        let policy_type = credential
            .storage_policy
            .as_ref()
            .map(|p| PolicyType::from_api(&p.policy_type))
            .unwrap_or(PolicyType::Local);

        let now = Utc::now().timestamp();
        let chunk_progress: Vec<ChunkProgress> =
            (0..num_chunks).map(|i: usize| ChunkProgress::new(i)).collect();
        let relay = credential.storage_policy.as_ref().and_then(|p| p.relay).unwrap_or(false);

        Self {
            id: credential.session_id.clone(),
            task_id,
            drive_id,
            local_path,
            remote_uri,
            file_size,
            chunk_size,
            policy_type,
            encrypt_metadata: credential.encrypt_metadata.clone(),
            chunk_progress,
            expires_at: credential.expires,
            created_at: now,
            updated_at: now,
            credential,
            relay,
        }
    }

    /// Calculate number of chunks for a file
    fn calculate_num_chunks(file_size: u64, chunk_size: u64) -> usize {
        if file_size == 0 || chunk_size == 0 {
            return 1; // Empty file still needs one "chunk"
        }
        ((file_size + chunk_size - 1) / chunk_size) as usize
    }

    /// Get the session ID from the credential
    pub fn session_id(&self) -> &str {
        &self.credential.session_id
    }

    /// Get the upload credential
    pub fn credential(&self) -> &UploadCredential {
        &self.credential
    }

    /// Get the policy type
    pub fn policy_type(&self) -> PolicyType {
        self.policy_type
    }

    pub fn is_relay(&self) -> bool {
        self.relay
    }

    /// Check if the session has expired
    pub fn is_expired(&self) -> bool {
        Utc::now().timestamp() >= self.expires_at
    }

    /// Get total number of chunks
    pub fn num_chunks(&self) -> usize {
        self.chunk_progress.len()
    }

    /// Get chunks that still need to be uploaded
    pub fn pending_chunks(&self) -> Vec<usize> {
        self.chunk_progress
            .iter()
            .filter(|c| !c.is_complete())
            .map(|c| c.index)
            .collect()
    }

    /// Check if all chunks are uploaded
    pub fn all_chunks_complete(&self) -> bool {
        self.chunk_progress.iter().all(|c| c.is_complete())
    }

    /// Get the expected size for a specific chunk
    pub fn chunk_size_for(&self, chunk_index: usize) -> u64 {
        if self.chunk_size == 0 {
            return self.file_size;
        }
        let start = chunk_index as u64 * self.chunk_size;
        let remaining = self.file_size.saturating_sub(start);
        remaining.min(self.chunk_size)
    }

    /// Get the byte range for a specific chunk
    pub fn chunk_range(&self, chunk_index: usize) -> (u64, u64) {
        if self.chunk_size == 0 {
            return (0, self.file_size);
        }
        let start = chunk_index as u64 * self.chunk_size;
        let size = self.chunk_size_for(chunk_index);
        (start, start + size)
    }

    /// Update chunk progress
    pub fn update_chunk_progress(&mut self, chunk_index: usize, loaded: u64, etag: Option<String>) {
        if chunk_index < self.chunk_progress.len() {
            self.chunk_progress[chunk_index].loaded = loaded;
            if let Some(tag) = etag {
                self.chunk_progress[chunk_index].etag = Some(tag);
            }
            self.updated_at = Utc::now().timestamp();
        }
    }

    /// Mark a chunk as complete
    pub fn complete_chunk(&mut self, chunk_index: usize, etag: Option<String>) {
        if chunk_index < self.chunk_progress.len() {
            let size = self.chunk_size_for(chunk_index);
            self.chunk_progress[chunk_index].loaded = size;
            if let Some(tag) = etag {
                self.chunk_progress[chunk_index].etag = Some(tag);
            }
            self.updated_at = Utc::now().timestamp();
        }
    }

    /// Get total bytes uploaded so far
    pub fn total_uploaded(&self) -> u64 {
        self.chunk_progress.iter().map(|c| c.loaded).sum()
    }

    /// Get upload progress as a percentage (0.0 - 1.0)
    pub fn progress(&self) -> f64 {
        if self.file_size == 0 {
            return 1.0;
        }
        self.total_uploaded() as f64 / self.file_size as f64
    }

    /// Get the upload URL for a specific chunk
    pub fn upload_url_for_chunk(&self, chunk_index: usize) -> Option<&str> {
        self.credential
            .upload_urls
            .as_ref()
            .and_then(|urls| urls.get(chunk_index))
            .map(|s| s.as_str())
    }

    /// Get the first upload URL (for providers that use single URL)
    pub fn upload_url(&self) -> Option<&str> {
        self.credential
            .upload_urls
            .as_ref()
            .and_then(|urls| urls.first().map(|s| s.as_str()))
    }

    /// Get the completion URL for multipart uploads
    pub fn complete_url(&self) -> &str {
        self.credential.complete_url.as_deref().unwrap_or_default()
    }

    /// Get the callback secret
    pub fn callback_secret(&self) -> &str {
        &self.credential.callback_secret
    }

    /// Get the upload credential string
    pub fn credential_string(&self) -> &str {
        &self.credential.credential
    }

    /// Get upload policy (for Upyun)
    pub fn upload_policy(&self) -> Option<&str> {
        self.credential.upload_policy.as_deref()
    }

    /// Get MIME type
    pub fn mime_type(&self) -> Option<&str> {
        self.credential.mime_type.as_deref()
    }

    /// Check if encryption is enabled
    pub fn is_encrypted(&self) -> bool {
        self.encrypt_metadata.is_some()
    }

    /// Check if streaming encryption is supported
    pub fn supports_streaming_encryption(&self) -> bool {
        self.credential
            .storage_policy
            .as_ref()
            .and_then(|p| p.streaming_encryption)
            .unwrap_or(false)
    }

    /// Get chunk upload concurrency from storage policy
    ///
    /// Returns the configured concurrency level for concurrent chunk uploads.
    /// Defaults to 1 (sequential uploads) if not specified.
    pub fn chunk_concurrency(&self) -> usize {
        self.credential
            .storage_policy
            .as_ref()
            .and_then(|p| p.chunk_concurrency)
            .map(|c| c.max(1) as usize)
            .unwrap_or(1)
    }
}

/// Serde helper for PolicyType
mod policy_type_serde {
    use super::PolicyType;
    use serde::{self, Deserialize, Deserializer, Serializer};

    pub fn serialize<S>(policy_type: &PolicyType, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        serializer.serialize_str(policy_type.as_str())
    }

    pub fn deserialize<'de, D>(deserializer: D) -> Result<PolicyType, D::Error>
    where
        D: Deserializer<'de>,
    {
        let s = String::deserialize(deserializer)?;
        Ok(PolicyType::from_str(&s))
    }
}
