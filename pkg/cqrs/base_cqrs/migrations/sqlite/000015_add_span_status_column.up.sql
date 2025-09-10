-- Add status column to spans table for efficient filtering and querying
ALTER TABLE spans ADD COLUMN status TEXT;

-- Add index on status for efficient filtering
CREATE INDEX idx_spans_status ON spans(status);

-- Add composite index on run_id, status for common queries
CREATE INDEX idx_spans_run_status ON spans(run_id, status);

-- Add composite index on account_id, status, start_time for filtered listing
CREATE INDEX idx_spans_account_status_time ON spans(account_id, status, start_time);