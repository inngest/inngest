-- +goose NO TRANSACTION
-- +goose Up

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_history_run_id_step_type
  ON history (run_id, step_type, created_at)
  WHERE step_type IS NOT NULL;

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_history_run_id_step_type;
