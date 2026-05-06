-- +goose Up

CREATE INDEX IF NOT EXISTS idx_events_event_id ON events (event_id);

-- +goose Down

DROP INDEX IF EXISTS idx_events_event_id;
