-- name: UpsertApp :one
INSERT INTO apps (id, name, sdk_language, sdk_version, framework, metadata, status, error, checksum, url, method, app_version)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    name = excluded.name,
    sdk_language = excluded.sdk_language,
    sdk_version = excluded.sdk_version,
    framework = excluded.framework,
    metadata = excluded.metadata,
    status = excluded.status,
    error = excluded.error,
    checksum = excluded.checksum,
    archived_at = NULL,
    "method" = excluded.method,
    app_version = excluded.app_version
RETURNING *;

-- name: GetApp :one
SELECT * FROM apps WHERE id = ?;

-- name: GetApps :many
SELECT * FROM apps WHERE archived_at IS NULL;

-- name: GetAppByChecksum :one
SELECT * FROM apps WHERE checksum = ? AND archived_at IS NULL LIMIT 1;

-- name: GetAppByID :one
SELECT * FROM apps WHERE id = ? LIMIT 1;

-- name: GetAppByURL :one
SELECT * FROM apps WHERE url = ? AND archived_at IS NULL LIMIT 1;

-- name: GetAppByName :one
SELECT * FROM apps WHERE name = ? AND archived_at IS NULL LIMIT 1;

-- name: GetAllApps :many
SELECT * FROM apps WHERE archived_at IS NULL;

-- name: DeleteApp :exec
UPDATE apps SET archived_at = datetime('now') WHERE id = ?;

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
WHERE functions.archived_at IS NULL
AND apps.archived_at IS NULL;

-- name: GetAppFunctions :many
SELECT * FROM functions WHERE app_id = ? AND archived_at IS NULL;

-- name: GetAppFunctionsBySlug :many
SELECT functions.* FROM functions JOIN apps ON apps.id = functions.app_id WHERE apps.name = ? AND functions.archived_at IS NULL;

-- name: GetFunctionByID :one
SELECT * FROM functions WHERE id = ?;

-- name: GetFunctionBySlug :one
SELECT * FROM functions WHERE slug = ? AND archived_at IS NULL;

-- name: UpdateFunctionConfig :one
UPDATE functions SET config = ?, archived_at = NULL WHERE id = ? RETURNING *;

-- name: DeleteFunctionsByAppID :exec
UPDATE functions SET archived_at = datetime('now') WHERE app_id = ?;

-- name: DeleteFunctionsByIDs :exec
UPDATE functions SET archived_at = datetime('now') WHERE id IN (sqlc.slice('ids'));

--
-- function runs
--

-- name: InsertFunctionRun :exec
INSERT INTO function_runs
	(run_id, run_started_at, function_id, function_version, trigger_type, event_id, batch_id, original_run_id, cron, workspace_id) VALUES
	(?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: InsertFunctionFinish :exec
INSERT INTO function_finishes
	(run_id, status, output, completed_step_count, created_at) VALUES
	(?, ?, ?, ?, ?);

-- name: GetFunctionRun :one
SELECT sqlc.embed(function_runs), sqlc.embed(function_finishes)
  FROM function_runs
  LEFT JOIN function_finishes ON function_finishes.run_id = function_runs.run_id
  WHERE function_runs.run_id = @run_id;

-- name: GetFunctionRuns :many
SELECT sqlc.embed(function_runs), sqlc.embed(function_finishes) FROM function_runs
LEFT JOIN function_finishes ON function_finishes.run_id = function_runs.run_id;

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
	(id, created_at, run_started_at, function_id, function_version, run_id, event_id, batch_id, group_id, idempotency_key, type, attempt, latency_ms, step_name, step_id, step_type, url, cancel_request, sleep, wait_for_event, wait_result, invoke_function, invoke_function_result, result) VALUES
	(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetHistoryItem :one
SELECT * FROM history WHERE id = ?;

-- name: GetFunctionRunHistory :many
SELECT * FROM history WHERE run_id = ? ORDER BY created_at ASC;

-- name: HistoryCountRuns :one
SELECT COUNT(DISTINCT run_id) FROM history;


--
-- Traces
--

-- name: InsertTrace :exec
INSERT INTO traces
	(timestamp, timestamp_unix_ms, trace_id, span_id, parent_span_id, trace_state, span_name, span_kind, service_name, resource_attributes, scope_name, scope_version, span_attributes, duration, status_code, status_message, events, links, run_id)
VALUES
	(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: InsertTraceRun :exec
INSERT INTO trace_runs (
    run_id, account_id, workspace_id, app_id, function_id, trace_id, 
    queued_at, started_at, ended_at, status, source_id, trigger_ids, 
    output, batch_id, is_debounce, cron_schedule, has_ai
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(run_id)
DO UPDATE SET
    account_id = excluded.account_id,
    workspace_id = excluded.workspace_id,
    app_id = excluded.app_id,
    function_id = excluded.function_id,
    trace_id = excluded.trace_id,
    queued_at = excluded.queued_at,
    started_at = excluded.started_at,
    ended_at = excluded.ended_at,
    status = excluded.status,
    source_id = excluded.source_id,
    trigger_ids = excluded.trigger_ids,
    output = excluded.output,
    batch_id = excluded.batch_id,
    is_debounce = excluded.is_debounce,
    cron_schedule = excluded.cron_schedule,
    has_ai = CASE
                 WHEN trace_runs.has_ai = 1 THEN 1
                 ELSE excluded.has_ai
             END;

-- name: GetTraceRun :one
SELECT * FROM trace_runs WHERE run_id = @run_id;

-- name: GetTraceSpans :many
SELECT * FROM traces WHERE trace_id = @trace_id AND run_id = @run_id ORDER BY timestamp_unix_ms DESC, duration DESC;

-- name: GetTraceSpanOutput :many
select * from traces where trace_id = @trace_id AND span_id = @span_id ORDER BY timestamp_unix_ms DESC, duration DESC;


--
-- Queue snapshots
--

-- name: GetQueueSnapshotChunks :many
SELECT chunk_id, data
FROM queue_snapshot_chunks
WHERE snapshot_id = ?
ORDER BY chunk_id ASC;

-- name: GetLatestQueueSnapshotChunks :many
SELECT chunk_id, data
FROM queue_snapshot_chunks
WHERE snapshot_id = (
    SELECT MAX(snapshot_id) FROM queue_snapshot_chunks
)
ORDER BY chunk_id ASC;

-- name: InsertQueueSnapshotChunk :exec
INSERT INTO queue_snapshot_chunks (snapshot_id, chunk_id, data)
VALUES
	(?, ?, ?);

-- name: DeleteOldQueueSnapshots :execrows
DELETE FROM queue_snapshot_chunks
WHERE snapshot_id NOT IN (
    SELECT snapshot_id
    FROM queue_snapshot_chunks
    ORDER BY snapshot_id DESC
    LIMIT ?
);

--
-- Worker Connections
--

-- name: InsertWorkerConnection :exec
INSERT INTO worker_connections (
    account_id, workspace_id, app_id, id, gateway_id, instance_id, status, worker_ip, connected_at, last_heartbeat_at, disconnected_at,
    recorded_at, inserted_at, disconnect_reason, group_hash, sdk_lang, sdk_version, sdk_platform, sync_id, app_version, function_count, cpu_cores, mem_bytes, os
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    ON CONFLICT(id)
DO UPDATE SET
    account_id = excluded.account_id,
    workspace_id = excluded.workspace_id,
    app_id = excluded.app_id,

    id = excluded.id,
    gateway_id = excluded.gateway_id,
    instance_id = excluded.instance_id,
    status = excluded.status,
    worker_ip = excluded.worker_ip,

    connected_at = excluded.connected_at,
    last_heartbeat_at = excluded.last_heartbeat_at,
    disconnected_at = excluded.disconnected_at,
    recorded_at = excluded.recorded_at,
    inserted_at = excluded.inserted_at,

    disconnect_reason = excluded.disconnect_reason,

    group_hash = excluded.group_hash,
    sdk_lang = excluded.sdk_lang,
    sdk_version = excluded.sdk_version,
    sdk_platform = excluded.sdk_platform,
    sync_id = excluded.sync_id,
    app_version = excluded.app_version,
    function_count = excluded.function_count,

    cpu_cores = excluded.cpu_cores,
    mem_bytes = excluded.mem_bytes,
    os = excluded.os
;

-- name: GetWorkerConnection :one
SELECT * FROM worker_connections WHERE account_id = @account_id AND workspace_id = @workspace_id AND id = @connection_id;
