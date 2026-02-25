-- Trace Runs Table
CREATE INDEX IF NOT EXISTS idx_trace_runs_started_at ON trace_runs(started_at DESC, run_id);
