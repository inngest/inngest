-- +goose NO TRANSACTION
-- +goose Up

-- Reconcile prod-only indexes that were created manually before the goose
-- migrations became the source of truth.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_spans_run_dynamic_endtime_status
  ON spans (run_id, dynamic_span_id, end_time DESC NULLS LAST) INCLUDE (status);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_spans_executor_run_start
  ON spans (start_time DESC, run_id)
  WHERE name = 'executor.run'
    AND debug_run_id IS NULL
    AND (status IS NULL OR status <> 'Skipped');

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_spans_dynamic_debug_starttime
  ON spans (dynamic_span_id, start_time)
  WHERE debug_run_id IS NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_spans_name_dynamic_span_id
  ON spans (name, dynamic_span_id);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_spans_name_dynamic_span_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_spans_dynamic_debug_starttime;
DROP INDEX CONCURRENTLY IF EXISTS idx_spans_executor_run_start;
DROP INDEX CONCURRENTLY IF EXISTS idx_spans_run_dynamic_endtime_status;
