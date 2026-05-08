-- +goose Up

-- Deduplicate any rows that accumulated before a PRIMARY KEY existed on
-- functions.id. For each group of duplicates, keep the physically latest
-- row (highest ctid) and delete the rest. Referencing tables (function_runs,
-- history, etc.) store function_id as a value — they stay valid because the
-- surviving row retains the same id.
DELETE FROM functions a
USING functions b
WHERE a.id = b.id AND a.ctid < b.ctid;

-- The fork's 000005 migration may have already added this constraint.
-- +goose StatementBegin
DO $$ BEGIN
  ALTER TABLE functions ADD PRIMARY KEY (id);
EXCEPTION WHEN duplicate_object OR invalid_table_definition THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose Down

ALTER TABLE functions DROP CONSTRAINT IF EXISTS functions_pkey;
