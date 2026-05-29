-- +goose Up

-- Nullable boolean: TRUE for spans emitted as part of a deferred run, NULL
-- otherwise. tracer_sqlc stamps the column from the executor.run span's
-- defer.parents attribute (present means deferred).
ALTER TABLE spans ADD COLUMN is_deferred BOOLEAN;

-- +goose Down

ALTER TABLE spans DROP COLUMN is_deferred;
