-- Drop the indexes first
DROP INDEX IF EXISTS idx_spans_account_status_time;
DROP INDEX IF EXISTS idx_spans_run_status;
DROP INDEX IF EXISTS idx_spans_status;

-- Drop the status column
ALTER TABLE spans DROP COLUMN IF EXISTS status;