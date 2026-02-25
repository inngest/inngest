-- Spans Table
CREATE INDEX IF NOT EXISTS idx_spans_name ON spans(name);
CREATE INDEX IF NOT EXISTS idx_spans_run_dynamic_end_time ON spans(run_id, dynamic_span_id, end_time DESC);
CREATE INDEX IF NOT EXISTS idx_spans_start_time ON spans(start_time DESC);

-- Trace Runs Table
CREATE INDEX IF NOT EXISTS idx_trace_runs_queued_at ON trace_runs(queued_at DESC, run_id);
CREATE INDEX IF NOT EXISTS idx_trace_runs_started_at ON trace_runs(started_at DESC, run_id);
CREATE INDEX IF NOT EXISTS idx_trace_runs_ended_at ON trace_runs(ended_at DESC, run_id);
CREATE INDEX IF NOT EXISTS idx_trace_runs_app_id ON trace_runs(app_id);
CREATE INDEX IF NOT EXISTS idx_trace_runs_function_id ON trace_runs(function_id);
