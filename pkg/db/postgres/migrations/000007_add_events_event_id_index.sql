-- +goose NO TRANSACTION
-- +goose Up

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_events_event_id
  ON events (event_id);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_events_event_id;
