-- +goose NO TRANSACTION
-- +goose Up

-- Supports the scoped root-run lookup used by GetSpanRuns:
--   SELECT dynamic_span_id FROM spans
--   WHERE name = 'executor.run'
--     AND account_id = $account_id
--     AND env_id = $env_id
--     AND start_time < $until
--     [AND start_time >= $from]
-- The partial predicate mirrors the visible runs-list root filter.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_spans_run_inner_lookup
  ON spans (account_id, env_id, start_time DESC, dynamic_span_id)
  INCLUDE (app_id, function_id, run_id)
  WHERE name = 'executor.run'
    AND debug_run_id IS NULL
    AND (status IS NULL OR status <> 'Skipped');

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_spans_run_inner_lookup;
