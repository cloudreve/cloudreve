-- Upload sessions table to persist multipart upload state
CREATE TABLE upload_sessions (
    id TEXT PRIMARY KEY NOT NULL,
    task_id TEXT NOT NULL,
    drive_id TEXT NOT NULL,
    local_path TEXT NOT NULL,
    remote_uri TEXT NOT NULL,
    file_size INTEGER NOT NULL,
    chunk_size INTEGER NOT NULL,
    policy_type TEXT NOT NULL,
    -- Serialized UploadCredential from cloudreve-api
    session_data TEXT NOT NULL,
    -- JSON array of chunk progress: [{"index": 0, "loaded": 1024, "etag": "abc"}]
    chunk_progress TEXT NOT NULL DEFAULT '[]',
    -- Encryption metadata if file is encrypted
    encrypt_metadata TEXT,
    expires_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- Index for task-based lookups
CREATE INDEX idx_upload_sessions_task_id ON upload_sessions(task_id);

-- Index for drive-based lookups
CREATE INDEX idx_upload_sessions_drive_id ON upload_sessions(drive_id);

-- Index for path-based lookups (for cancellation)
CREATE INDEX idx_upload_sessions_local_path ON upload_sessions(local_path);

