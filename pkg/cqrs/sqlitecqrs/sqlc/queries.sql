-- name: UpsertApp :one
INSERT INTO apps (id, name, sdk_language, sdk_version, framework, metadata, status, error, checksum, url)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    name = excluded.name,
    sdk_language = excluded.sdk_language,
    sdk_version = excluded.sdk_version,
    framework = excluded.framework,
    metadata = excluded.metadata,
    status = excluded.status,
    error = excluded.error,
    checksum = excluded.checksum,
    deleted_at = NULL
RETURNING *;

-- name: GetApp :one
SELECT * FROM apps WHERE id = ?;

-- name: GetApps :many
SELECT * FROM apps WHERE deleted_at IS NULL;

-- name: GetAppByChecksum :one
SELECT * FROM apps WHERE checksum = ? AND deleted_at IS NULL LIMIT 1;

-- name: GetAppByID :one
SELECT * FROM apps WHERE id = ? LIMIT 1;

-- name: GetAppByURL :one
SELECT * FROM apps WHERE url = ? AND deleted_at IS NULL LIMIT 1;

-- name: GetAllApps :many
SELECT * FROM apps WHERE deleted_at IS NULL;

-- name: DeleteApp :exec
UPDATE apps SET deleted_at = datetime('now') WHERE id = ?;

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
SELECT functions.*
FROM functions
JOIN apps ON apps.id = functions.app_id
WHERE functions.deleted_at IS NULL
AND apps.deleted_at IS NULL;

-- name: GetAppFunctions :many
SELECT * FROM functions WHERE app_id = ? AND deleted_at IS NULL;

-- name: GetAppFunctionsBySlug :many
SELECT functions.* FROM functions JOIN apps ON apps.id = functions.app_id WHERE apps.name = ? AND functions.deleted_at IS NULL;

-- name: GetFunctionByID :one
SELECT * FROM functions WHERE id = ?;

-- name: GetFunctionBySlug :one
SELECT * FROM functions WHERE slug = ? AND deleted_at IS NULL;

-- name: UpdateFunctionConfig :one
UPDATE functions SET config = ? WHERE id = ? RETURNING *;

-- name: DeleteFunctionsByAppID :exec
UPDATE functions SET deleted_at = datetime('now') WHERE app_id = ?;

-- name: DeleteFunctionsByIDs :exec
UPDATE functions SET deleted_at = datetime('now') WHERE id IN (sqlc.slice('ids'));

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

-- name: InsertEventBatch :exec
INSERT INTO event_batches
	(id, account_id, workspace_id, app_id, workflow_id, run_id, started_at, executed_at, event_ids) VALUES
	(?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetEventByInternalID :one
SELECT * FROM events WHERE internal_id = ?;

-- name: GetEventsByInternalIDs :many
SELECT * FROM events WHERE internal_id IN (sqlc.slice('ids'));

-- name: GetEventBatchByRunID :one
SELECT * FROM event_batches WHERE run_id = ?;

-- name: GetEventBatchesByEventID :many
SELECT * FROM event_batches WHERE INSTR(CAST(event_ids AS TEXT), ?) > 0;

-- name: GetEventsIDbound :many
SELECT DISTINCT e.*
FROM events AS e
LEFT OUTER JOIN function_runs AS r ON r.event_id = e.internal_id
WHERE
	e.internal_id > @after
	AND e.internal_id < @before
	AND (
		-- Include internal events that triggered a run (e.g. an onFailure
		-- handler)
		r.run_id IS NOT NULL

		-- Optionally include internal events that did not trigger a run. It'd
		-- be better to use a boolean param instead of a string param but sqlc
		-- keeps making @include_internal a string.
		OR CASE WHEN e.event_name LIKE 'inngest/%' THEN 'true' ELSE 'false' END = @include_internal
	)
ORDER BY e.internal_id DESC
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


--
-- Traces
--

-- name: InsertTrace :exec
INSERT INTO traces
	(timestamp, timestamp_unix_ms, trace_id, span_id, parent_span_id, trace_state, span_name, span_kind, service_name, resource_attributes, scope_name, scope_version, span_attributes, duration, status_code, status_message, events, links, run_id)
VALUES
	(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: InsertTraceRun :exec
INSERT OR REPLACE INTO trace_runs
	(account_id, workspace_id, app_id, function_id, trace_id, run_id, queued_at, started_at, ended_at, status, source_id, trigger_ids, output, batch_id, is_debounce, cron_schedule)
VALUES
	(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetTraceRun :one
SELECT * FROM trace_runs WHERE run_id = @run_id;

-- name: GetTraceSpans :many
SELECT * FROM traces WHERE trace_id = @trace_id AND run_id = @run_id ORDER BY timestamp_unix_ms DESC, duration DESC;

-- name: GetTraceSpanOutput :many
select * from traces where trace_id = @trace_id AND span_id = @span_id ORDER BY timestamp_unix_ms DESC, duration DESC;
