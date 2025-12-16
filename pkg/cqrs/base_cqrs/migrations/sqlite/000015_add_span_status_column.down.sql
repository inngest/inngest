-- Drop the indexes first
DROP INDEX IF EXISTS idx_spans_account_status_time;
DROP INDEX IF EXISTS idx_spans_run_status;
DROP INDEX IF EXISTS idx_spans_status;

-- Drop the status column (SQLite doesn't support DROP COLUMN directly)
-- This would require recreating the table, but for now we'll leave as comment
-- ALTER TABLE spans DROP COLUMN status;