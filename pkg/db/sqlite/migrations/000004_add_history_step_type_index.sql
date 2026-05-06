-- +goose Up

CREATE INDEX IF NOT EXISTS idx_history_run_id_step_type
    ON history (run_id, step_type, created_at)
    WHERE step_type IS NOT NULL;

-- +goose Down

DROP INDEX IF EXISTS idx_history_run_id_step_type;
