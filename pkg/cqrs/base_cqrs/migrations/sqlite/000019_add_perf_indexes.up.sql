-- Spans: covers the WHERE name = 'executor.run' inner subquery in GetSpanRuns
CREATE INDEX IF NOT EXISTS idx_spans_name ON spans(name);

-- Spans: covers the correlated subquery ORDER BY end_time DESC with filter on (run_id, dynamic_span_id)
CREATE INDEX IF NOT EXISTS idx_spans_run_dynamic_end_time ON spans(run_id, dynamic_span_id, end_time DESC);

-- Spans: covers the time range filter WHERE start_time >= ... AND start_time < ...
CREATE INDEX IF NOT EXISTS idx_spans_start_time ON spans(start_time DESC);

-- Trace runs: covers time range filters used by GetTraceRuns
CREATE INDEX IF NOT EXISTS idx_trace_runs_queued_at ON trace_runs(queued_at DESC);
CREATE INDEX IF NOT EXISTS idx_trace_runs_started_at ON trace_runs(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_trace_runs_ended_at ON trace_runs(ended_at DESC);

-- Trace runs: covers status and scope filters
CREATE INDEX IF NOT EXISTS idx_trace_runs_status ON trace_runs(status);
CREATE INDEX IF NOT EXISTS idx_trace_runs_app_id ON trace_runs(app_id);
CREATE INDEX IF NOT EXISTS idx_trace_runs_function_id ON trace_runs(function_id);
