-- +goose NO TRANSACTION
-- +goose Up

-- Supports the outer scan used by the run list. The partial predicate mirrors
-- the query's active-run filter so the index stays narrow and the planner
-- picks it over a parallel seq scan for typical 1-7 day windows.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_spans_active_start_time
  ON spans (start_time)
  WHERE debug_run_id IS NULL
    AND (status IS NULL OR status <> 'Skipped');

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_spans_active_start_time;
