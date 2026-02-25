-- Trace Runs Table
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_trace_runs_app_id ON trace_runs(app_id);
