-- +goose Up

CREATE TABLE IF NOT EXISTS run_defers (
    parent_run_id BYTEA NOT NULL,
    defer_id VARCHAR NOT NULL,
    user_defer_id VARCHAR NOT NULL,
    fn_slug VARCHAR NOT NULL,
    status VARCHAR NOT NULL,
    child_run_id BYTEA,
    PRIMARY KEY (parent_run_id, defer_id)
);

CREATE INDEX IF NOT EXISTS idx_run_defers_child ON run_defers(child_run_id);

-- +goose Down

DROP INDEX IF EXISTS idx_run_defers_child;
DROP TABLE IF EXISTS run_defers;
