-- Add local snapshot columns to file_metadata so the sync planner can detect
-- local edits even when the CFAPI IN_SYNC flag is stale, missing or cleared
-- by a race. Together with etag/updated_at of the remote state this forms a
-- git-like base/local/remote three-way comparison.
--
-- local_updated_at: local file mtime in unix milliseconds at last sync point
-- local_size:       local file size in bytes at last sync point
ALTER TABLE file_metadata ADD COLUMN local_updated_at BIGINT;
ALTER TABLE file_metadata ADD COLUMN local_size BIGINT;
