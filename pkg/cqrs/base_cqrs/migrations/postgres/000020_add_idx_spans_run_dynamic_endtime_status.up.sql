-- Spans Table
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_spans_run_dynamic_endtime_status
    ON spans(run_id, dynamic_span_id, end_time DESC) INCLUDE (status);
