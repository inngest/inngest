-- name: InsertApp :one
INSERT INTO apps
	(id, name, sdk_language, sdk_version, framework, metadata, status, error, checksum, url) VALUES
	(?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING *;

-- name: GetApp :one
SELECT * FROM apps WHERE id = ?;

-- name: GetApps :many
SELECT * FROM apps WHERE deleted_at IS NULL;

-- name: GetAppByChecksum :one
SELECT * FROM apps WHERE checksum = ? LIMIT 1;

-- name: GetAppByURL :one
SELECT * FROM apps WHERE url = ? LIMIT 1;

-- name: GetAllApps :many
SELECT * FROM apps;

-- name: DeleteApp :exec
UPDATE apps SET deleted_at = NOW() WHERE id = ?;

-- name: HardDeleteApp :exec
DELETE FROM apps WHERE id = ?;

-- name: UpdateAppURL :one
UPDATE apps SET url = ? WHERE id = ? RETURNING *;

-- name: UpdateAppError :one
UPDATE apps SET error = ? WHERE id = ? RETURNING *;


--
-- functions
--


-- note - this is very basic right now.
-- name: InsertFunction :one
INSERT INTO functions
	(id, app_id, name, slug, config, created_at) VALUES
	(?, ?, ?, ?, ?, ?) RETURNING *;

-- name: GetFunctions :many
SELECT * FROM functions;

-- name: GetAppFunctions :many
SELECT * FROM functions WHERE app_id = ?;

-- name: GetFunctionByID :one
SELECT * FROM functions WHERE id = ?;

-- name: UpdateFunctionConfig :one
UPDATE functions SET config = ? WHERE id = ? RETURNING *;

-- name: DeleteFunctionsByAppID :exec
DELETE FROM functions WHERE app_id = ?;

-- name: DeleteFunctionsByIDs :exec
DELETE FROM functions WHERE id IN (sqlc.slice('ids'));


--
-- function runs
--

-- name: InsertFunctionRun :exec
INSERT INTO function_runs
	(run_id, run_started_at, function_id, function_version, event_id, batch_id, original_run_id) VALUES
	(?, ?, ?, ?, ?, ?, ?);

--
-- Events
--

-- name: InsertEvent :exec
INSERT INTO events
	(internal_id, event_id, event_data, event_user, event_v, event_ts) VALUES
	(?, ?, ?, ?, ?, ?);

-- name: GetEventByInternalID :one
SELECT * FROM events WHERE internal_id = ?;

--
-- History
--

-- name: InsertHistory :exec
INSERT INTO history
	(id, created_at, run_started_at, function_id, function_version, run_id, event_id, batch_id, group_id, idempotency_key, type, attempt, step_name, step_id, url, cancel_request, sleep, wait_for_event, result) VALUES
	(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
