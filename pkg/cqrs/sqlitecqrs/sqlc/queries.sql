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

-- name: GetAppByID :one
SELECT * FROM apps WHERE id = ? LIMIT 1;

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

-- name: GetAppFunctionsBySlug :many
SELECT functions.* FROM functions JOIN apps ON apps.id = functions.app_id WHERE apps.name = ?;

-- name: GetFunctionByID :one
SELECT * FROM functions WHERE id = ?;

-- name: GetFunctionBySlug :one
SELECT * FROM functions WHERE slug = ?;


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
	(run_id, run_started_at, function_id, function_version, trigger_type, event_id, batch_id, original_run_id, cron) VALUES
	(?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: InsertFunctionFinish :exec
INSERT INTO function_finishes
	(run_id, status, output, completed_step_count, created_at) VALUES 
	(?, ?, ?, ?, ?);

-- name: GetFunctionRun :one
SELECT sqlc.embed(function_runs), sqlc.embed(function_finishes)
  FROM function_runs
  LEFT JOIN function_finishes ON function_finishes.run_id = function_runs.run_id
  WHERE function_runs.run_id = @run_id;

-- name: GetFunctionRunsTimebound :many
SELECT sqlc.embed(function_runs), sqlc.embed(function_finishes) FROM function_runs
LEFT JOIN function_finishes ON function_finishes.run_id = function_runs.run_id
WHERE function_runs.run_started_at > @after AND function_runs.run_started_at <= @before
ORDER BY function_runs.run_started_at DESC
LIMIT ?;

-- name: GetFunctionRunsFromEvents :many
SELECT sqlc.embed(function_runs), sqlc.embed(function_finishes) FROM function_runs
LEFT JOIN function_finishes ON function_finishes.run_id = function_runs.run_id
WHERE function_runs.event_id IN (sqlc.slice('event_ids'));

-- name: GetFunctionRunFinishesByRunIDs :many
SELECT * FROM function_finishes WHERE run_id IN (sqlc.slice('run_ids'));

--
-- Events
--

-- name: InsertEvent :exec
INSERT INTO events
	(internal_id, received_at, event_id, event_name, event_data, event_user, event_v, event_ts) VALUES
	(?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetEventByInternalID :one
SELECT * FROM events WHERE internal_id = ?;

-- name: GetEventsTimebound :many
SELECT DISTINCT e.*
FROM events AS e
LEFT OUTER JOIN function_runs AS r ON r.event_id = e.internal_id
WHERE
	e.received_at > @after
	AND e.received_at <= @before
	AND (
		-- Include internal events that triggered a run (e.g. an onFailure
		-- handler)
		r.run_id IS NOT NULL

		-- Optionally include internal events that did not trigger a run. It'd
		-- be better to use a boolean param instead of a string param but sqlc
		-- keeps making @include_internal a string.
		OR CASE WHEN e.event_name LIKE 'inngest/%' THEN 'true' ELSE 'false' END = @include_internal
	)
ORDER BY e.received_at DESC
LIMIT ?;

-- name: WorkspaceEvents :many
SELECT * FROM events WHERE internal_id < @cursor AND received_at <= @before AND received_at >= @after ORDER BY internal_id DESC LIMIT ?;

-- name: WorkspaceNamedEvents :many
SELECT * FROM events WHERE internal_id < @cursor AND received_at <= @before AND received_at >= @after AND event_name = @name ORDER BY internal_id DESC LIMIT ?;

--
-- History
--

-- name: InsertHistory :exec
INSERT INTO history
	(id, created_at, run_started_at, function_id, function_version, run_id, event_id, batch_id, group_id, idempotency_key, type, attempt, latency_ms, step_name, step_id, url, cancel_request, sleep, wait_for_event, wait_result, invoke_function, invoke_function_result, result) VALUES
	(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetFunctionRunHistory :many
SELECT * FROM history WHERE run_id = ? ORDER BY created_at ASC;
