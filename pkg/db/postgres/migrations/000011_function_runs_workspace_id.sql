-- +goose Up

-- function_runs.workspace_id matches SQLite. Without it, reads return the
-- zero UUID and DELETE /v1/runs/{id} treats an existing run as missing.
ALTER TABLE function_runs ADD COLUMN workspace_id UUID;

-- Rows written before this column existed belong to the single dev-server
-- environment (00000000-0000-4000-b000-000000000000).
UPDATE function_runs
SET workspace_id = '00000000-0000-4000-b000-000000000000'
WHERE workspace_id IS NULL;

-- +goose Down

ALTER TABLE function_runs DROP COLUMN workspace_id;
