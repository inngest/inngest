-- Spans Table
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_spans_start_time ON spans(start_time DESC);
