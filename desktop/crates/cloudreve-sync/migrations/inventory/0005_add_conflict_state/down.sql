-- SQLite doesn't support DROP COLUMN directly in older versions
-- For newer SQLite (3.35.0+), we can use:
ALTER TABLE file_metadata DROP COLUMN conflict_state;

DROP INDEX IF EXISTS idx_conflict_state;
