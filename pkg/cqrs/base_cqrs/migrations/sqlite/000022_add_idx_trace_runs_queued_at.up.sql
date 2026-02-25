-- Trace Runs Table
CREATE INDEX IF NOT EXISTS idx_trace_runs_queued_at ON trace_runs(queued_at DESC, run_id);
