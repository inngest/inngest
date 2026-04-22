-- +goose Up

-- Deduplicate any rows that accumulated before a PRIMARY KEY existed on
-- functions.id. For each group of duplicates, keep the physically latest
-- row (highest ctid) and delete the rest. Referencing tables (function_runs,
-- history, etc.) store function_id as a value — they stay valid because the
-- surviving row retains the same id.
DELETE FROM functions a
USING functions b
WHERE a.id = b.id AND a.ctid < b.ctid;

ALTER TABLE functions ADD PRIMARY KEY (id);

-- +goose Down

ALTER TABLE functions DROP CONSTRAINT functions_pkey;
