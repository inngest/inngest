-- Add indexes to trace_runs for common query patterns (filter by app, function, time, status)
CREATE INDEX IF NOT EXISTS idx_trace_runs_app_id ON trace_runs(app_id);
CREATE INDEX IF NOT EXISTS idx_trace_runs_function_id ON trace_runs(function_id);
CREATE INDEX IF NOT EXISTS idx_trace_runs_queued_at ON trace_runs(queued_at);
CREATE INDEX IF NOT EXISTS idx_trace_runs_started_at ON trace_runs(started_at);
CREATE INDEX IF NOT EXISTS idx_trace_runs_ended_at ON trace_runs(ended_at);
CREATE INDEX IF NOT EXISTS idx_trace_runs_status ON trace_runs(status);

-- Add index on spans.name for the subquery filtering by span name (e.g. 'executor.run')
CREATE INDEX IF NOT EXISTS idx_spans_name ON spans(name);
