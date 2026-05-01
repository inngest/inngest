-- +goose Up

-- Mirror of postgres migration 000004: drop redundant spans indexes (APP-2785).
-- SQLite does not support DROP INDEX CONCURRENTLY or IF EXISTS on all versions,
-- but the dev server uses modernc/sqlite which supports IF EXISTS.
DROP INDEX IF EXISTS idx_spans_run_status;
DROP INDEX IF EXISTS idx_spans_status;

-- +goose Down

CREATE INDEX IF NOT EXISTS idx_spans_run_status ON spans(run_id, status);
CREATE INDEX IF NOT EXISTS idx_spans_status ON spans(status);
