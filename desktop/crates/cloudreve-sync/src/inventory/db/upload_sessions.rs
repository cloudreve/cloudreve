use super::InventoryDb;
use anyhow::{Context, Result};
use chrono::Utc;
use diesel::prelude::*;

use crate::inventory::schema::upload_sessions::{self, dsl as upload_sessions_dsl};

impl InventoryDb {
    /// Insert a new upload session
    pub fn insert_upload_session(&self, session: &crate::uploader::UploadSession) -> Result<()> {
        let mut conn = self.connection()?;
        let row = UploadSessionRow::from_session(session)?;
        diesel::insert_into(upload_sessions::table)
            .values(&row)
            .execute(&mut conn)
            .context("Failed to insert upload session")?;
        Ok(())
    }

    /// Get upload session by task ID
    pub fn get_upload_session(
        &self,
        task_id: &str,
    ) -> Result<Option<crate::uploader::UploadSession>> {
        let mut conn = self.connection()?;
        let row = upload_sessions_dsl::upload_sessions
            .filter(upload_sessions_dsl::task_id.eq(task_id))
            .first::<UploadSessionQueryRow>(&mut conn)
            .optional()
            .context("Failed to query upload session")?;

        row.map(crate::uploader::UploadSession::try_from)
            .transpose()
    }

    /// Get upload session by local file path
    pub fn get_upload_session_by_path(
        &self,
        path: &str,
    ) -> Result<Option<crate::uploader::UploadSession>> {
        let mut conn = self.connection()?;
        let row = upload_sessions_dsl::upload_sessions
            .filter(upload_sessions_dsl::local_path.eq(path))
            .first::<UploadSessionQueryRow>(&mut conn)
            .optional()
            .context("Failed to query upload session by ID")?;

        row.map(crate::uploader::UploadSession::try_from)
            .transpose()
    }

    /// Delete upload session
    pub fn delete_upload_session(&self, session_id: &str) -> Result<()> {
        let mut conn = self.connection()?;
        diesel::delete(
            upload_sessions_dsl::upload_sessions.filter(upload_sessions_dsl::id.eq(session_id)),
        )
        .execute(&mut conn)
        .context("Failed to delete upload session")?;
        Ok(())
    }

    pub fn batch_delete_upload_session_by_path(&self, paths: &[&str]) -> Result<bool> {
        if paths.is_empty() {
            return Ok(false);
        }

        let mut conn = self.connection()?;
        let affected = diesel::delete(
            upload_sessions_dsl::upload_sessions
                .filter(upload_sessions_dsl::local_path.eq_any(paths)),
        )
        .execute(&mut conn)
        .context("Failed to batch delete upload session by path")?;
        Ok(affected > 0)
    }

    /// Delete expired upload sessions
    pub fn delete_expired_upload_sessions(&self) -> Result<usize> {
        let mut conn = self.connection()?;
        let now = Utc::now().timestamp();
        let deleted = diesel::delete(
            upload_sessions_dsl::upload_sessions.filter(upload_sessions_dsl::expires_at.lt(now)),
        )
        .execute(&mut conn)
        .context("Failed to delete expired upload sessions")?;
        Ok(deleted)
    }
}

// =========================================================================
// Row Types
// =========================================================================

#[derive(Queryable)]
pub(crate) struct UploadSessionQueryRow {
    pub id: String,
    pub task_id: String,
    pub drive_id: String,
    pub local_path: String,
    pub remote_uri: String,
    pub file_size: i64,
    #[allow(dead_code)]
    pub chunk_size: i64,
     #[allow(dead_code)]
    pub policy_type: String,
    pub session_data: String,
    pub chunk_progress: String,
    pub encrypt_metadata: Option<String>,
     #[allow(dead_code)]
    pub expires_at: i64,
    pub created_at: i64,
    pub updated_at: i64,
}

#[derive(Insertable)]
#[diesel(table_name = upload_sessions)]
struct UploadSessionRow {
    id: String,
    task_id: String,
    drive_id: String,
    local_path: String,
    remote_uri: String,
    file_size: i64,
    chunk_size: i64,
    policy_type: String,
    session_data: String,
    chunk_progress: String,
    encrypt_metadata: Option<String>,
    expires_at: i64,
    created_at: i64,
    updated_at: i64,
}

impl UploadSessionRow {
    fn from_session(session: &crate::uploader::UploadSession) -> Result<Self> {
        let credential_json = serde_json::to_string(session.credential())
            .context("Failed to serialize upload credential")?;
        let chunk_progress_json = serde_json::to_string(&session.chunk_progress)
            .context("Failed to serialize chunk progress")?;
        let encrypt_metadata_json = session
            .encrypt_metadata
            .as_ref()
            .map(|m| serde_json::to_string(m))
            .transpose()
            .context("Failed to serialize encrypt metadata")?;

        Ok(Self {
            id: session.id.clone(),
            task_id: session.task_id.clone(),
            drive_id: session.drive_id.clone(),
            local_path: session.local_path.clone(),
            remote_uri: session.remote_uri.clone(),
            file_size: session.file_size as i64,
            chunk_size: session.chunk_size as i64,
            policy_type: session.policy_type().as_str().to_string(),
            session_data: credential_json,
            chunk_progress: chunk_progress_json,
            encrypt_metadata: encrypt_metadata_json,
            expires_at: session.expires_at,
            created_at: session.created_at,
            updated_at: session.updated_at,
        })
    }
}

impl TryFrom<UploadSessionQueryRow> for crate::uploader::UploadSession {
    type Error = anyhow::Error;

    fn try_from(row: UploadSessionQueryRow) -> Result<Self> {
        let credential: cloudreve_api::models::explorer::UploadCredential =
            serde_json::from_str(&row.session_data)
                .context("Failed to deserialize upload credential")?;
        let chunk_progress: Vec<crate::uploader::ChunkProgress> =
            serde_json::from_str(&row.chunk_progress)
                .context("Failed to deserialize chunk progress")?;
        let encrypt_metadata = row
            .encrypt_metadata
            .map(|s| serde_json::from_str(&s))
            .transpose()
            .context("Failed to deserialize encrypt metadata")?;

        // Reconstruct the session
        let mut session = crate::uploader::UploadSession::new(
            row.task_id,
            row.drive_id,
            row.local_path,
            row.remote_uri,
            row.file_size as u64,
            credential,
        );

        // Restore persisted state
        session.id = row.id;
        session.chunk_progress = chunk_progress;
        session.encrypt_metadata = encrypt_metadata;
        session.created_at = row.created_at;
        session.updated_at = row.updated_at;

        Ok(session)
    }
}
