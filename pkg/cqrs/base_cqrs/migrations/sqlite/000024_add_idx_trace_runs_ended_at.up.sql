-- Trace Runs Table
CREATE INDEX IF NOT EXISTS idx_trace_runs_ended_at ON trace_runs(ended_at DESC, run_id);
