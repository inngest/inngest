-- +goose NO TRANSACTION
-- +goose Up

-- Drop indexes identified as redundant by pg_stat_user_indexes audit (APP-2785).
-- Observation window covers the full database lifetime (stats never reset).
--
-- idx_spans_active_start_time: 9 scans, 0 tuples fetched, 32 MB.
--   The partial predicate (debug_run_id IS NULL AND status <> 'Skipped')
--   is too narrow to help the planner; the name+start_time compound index
--   serves the actual inner subquery better.
DROP INDEX CONCURRENTLY IF EXISTS idx_spans_active_start_time;

-- idx_spans_run_status: 57 scans, 57 tuples fetched, 75 MB.
--   Worst size-to-usage ratio. No query filters by (run_id, status) together;
--   status filtering always goes through idx_spans_account_status_time.
DROP INDEX CONCURRENTLY IF EXISTS idx_spans_run_status;

-- idx_spans_name: 260 scans, 34 MB.
--   Fully redundant — name is the leading column of
--   idx_spans_name_start_time_dynamic_span_id which has 765 scans.
DROP INDEX CONCURRENTLY IF EXISTS idx_spans_name;

-- idx_spans_status: 264 scans reading 2.8M tuples (near-full scans), 34 MB.
--   Low cardinality makes standalone status index ineffective; every
--   status-filtered query also filters by account_id or run_id, served
--   by idx_spans_account_status_time or idx_spans_run_id_dynamic_start_time.
DROP INDEX CONCURRENTLY IF EXISTS idx_spans_status;

-- +goose Down

-- Recreate all four indexes. Use CONCURRENTLY to avoid locking the table.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_spans_active_start_time
  ON spans (start_time)
  WHERE debug_run_id IS NULL
    AND (status IS NULL OR status <> 'Skipped');

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_spans_run_status
  ON spans (run_id, status);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_spans_name
  ON spans (name);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_spans_status
  ON spans (status);
