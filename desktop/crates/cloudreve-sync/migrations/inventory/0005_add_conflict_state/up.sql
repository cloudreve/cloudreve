-- Add conflict_state column to track file conflict status
-- NULL: no conflict
-- 'pending': file has conflict, pending user action
-- 'override': user chose to override the remote version
ALTER TABLE file_metadata ADD COLUMN conflict_state TEXT;

CREATE INDEX IF NOT EXISTS idx_conflict_state ON file_metadata(conflict_state);
