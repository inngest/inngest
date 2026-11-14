-- +goose Up

-- The SQLite schema stores workspace_id on function_runs, but the Postgres
-- schema never had the column. That left cqrs.FunctionRun.WorkspaceID as the
-- zero UUID for every Postgres-backed run, so the workspace check in
-- apiv1.CancelFunctionRun always failed and DELETE /v1/runs/{id} answered 404
-- for runs that exist.
ALTER TABLE function_runs ADD COLUMN workspace_id UUID;

-- +goose Down

ALTER TABLE function_runs DROP COLUMN workspace_id;
