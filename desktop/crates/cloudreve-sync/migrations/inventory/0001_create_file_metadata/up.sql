CREATE TABLE IF NOT EXISTS file_metadata (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    drive_id TEXT NOT NULL,
    is_folder BOOLEAN NOT NULL,
    local_path TEXT NOT NULL UNIQUE,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    etag TEXT NOT NULL,
    metadata TEXT NOT NULL,
    props TEXT,
    permissions TEXT NOT NULL,
    shared BOOLEAN NOT NULL,
    size BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_drive_id ON file_metadata(drive_id);
CREATE INDEX IF NOT EXISTS idx_local_path ON file_metadata(local_path);
CREATE INDEX IF NOT EXISTS idx_updated_at ON file_metadata(updated_at);

