-- +goose Up

-- 1 = enums.RunTypePrimary. Existing rows predate the defer feature so they
-- are all primary; new inserts stamp the column from the executor.run span's
-- defer.parent_run_ids attribute (present means defer).
ALTER TABLE spans ADD COLUMN run_type INT NOT NULL DEFAULT 1;

-- +goose Down

ALTER TABLE spans DROP COLUMN run_type;
