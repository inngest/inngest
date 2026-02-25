-- Trace Runs Table
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_trace_runs_function_id ON trace_runs(function_id);
