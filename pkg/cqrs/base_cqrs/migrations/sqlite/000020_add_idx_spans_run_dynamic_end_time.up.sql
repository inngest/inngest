-- Spans Table
CREATE INDEX IF NOT EXISTS idx_spans_run_dynamic_end_time ON spans(run_id, dynamic_span_id, end_time DESC);
