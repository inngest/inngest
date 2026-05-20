-- +goose Up

-- 1 = enums.RunTypePrimary. Existing rows predate the defer feature so they
-- are all primary; new inserts set the column explicitly from the trigger
-- event(s).
ALTER TABLE trace_runs ADD COLUMN run_type INT NOT NULL DEFAULT 1;

-- +goose Down

ALTER TABLE trace_runs DROP COLUMN run_type;
