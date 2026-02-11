-- Add indexes to improve GetSpanRuns query performance on large datasets.
-- The GetSpanRuns query (used by the runs list view) performs:
--   1. A subquery: SELECT DISTINCT dynamic_span_id FROM spans WHERE name = 'executor.run'
--   2. A main filter: WHERE debug_run_id IS NULL AND start_time >= $1 AND start_time < $2
-- Without these indexes, both operations require full table scans.

CREATE INDEX IF NOT EXISTS idx_spans_name_dynamic_span_id ON spans(name, dynamic_span_id);
CREATE INDEX IF NOT EXISTS idx_spans_debug_run_id_start_time ON spans(debug_run_id, start_time);
