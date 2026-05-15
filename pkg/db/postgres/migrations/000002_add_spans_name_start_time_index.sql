-- +goose NO TRANSACTION
-- +goose Up

-- Supports the inner subquery used by the run list (GetSpanRuns):
--   SELECT dynamic_span_id FROM spans
--   WHERE name = 'executor.run' AND start_time >= $from AND start_time < $until
-- With all three columns in the index the planner can use an index-only scan
-- instead of a bitmap heap scan + re-check, and the subquery stops scanning
-- historical executor.run rows as the deployment ages.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_spans_name_start_time_dynamic_span_id
  ON spans (name, start_time, dynamic_span_id);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_spans_name_start_time_dynamic_span_id;
