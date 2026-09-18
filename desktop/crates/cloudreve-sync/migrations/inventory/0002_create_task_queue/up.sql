CREATE TABLE IF NOT EXISTS task_queue (
    id TEXT PRIMARY KEY,
    drive_id TEXT NOT NULL,
    task_type TEXT NOT NULL,
    local_path TEXT NOT NULL,
    status TEXT NOT NULL,
    progress REAL NOT NULL DEFAULT 0,
    total_bytes BIGINT NOT NULL DEFAULT 0,
    processed_bytes BIGINT NOT NULL DEFAULT 0,
    priority INTEGER NOT NULL DEFAULT 0,
    custom_state TEXT,
    error TEXT,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_task_queue_drive ON task_queue(drive_id);
CREATE INDEX IF NOT EXISTS idx_task_queue_status ON task_queue(status);
CREATE INDEX IF NOT EXISTS idx_task_queue_local_path ON task_queue(local_path);

