-- +goose NO TRANSACTION
-- +goose Up

-- run_id follows start_time so root-page queries can use index order for top-N.
-- dynamic_span_id is included for the legacy root lookup.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_spans_run_inner_lookup
  ON spans (account_id, env_id, start_time DESC, run_id)
  INCLUDE (dynamic_span_id, app_id, function_id)
  WHERE name = 'executor.run'
    AND debug_run_id IS NULL
    AND (status IS NULL OR status <> 'Skipped');

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_spans_run_inner_lookup;
