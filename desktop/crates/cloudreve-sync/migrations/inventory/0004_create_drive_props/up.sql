-- Drive props table to store cached properties for each drive
-- Stores user capacity, storage policies, and other settings
CREATE TABLE drive_props (
    id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    drive_id TEXT NOT NULL UNIQUE,
    -- User capacity information (JSON serialized)
    capacity TEXT,
    capacity_updated_at INTEGER,
    -- Storage policies (JSON serialized array)
    storage_policies TEXT,
    storage_policies_updated_at INTEGER,
    -- User settings (JSON serialized)
    user_settings TEXT,
    user_settings_updated_at INTEGER,
    -- General timestamps
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- Index for drive_id lookups
CREATE INDEX idx_drive_props_drive_id ON drive_props(drive_id);


