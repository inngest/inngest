-- name: UpsertApp :one
INSERT INTO apps (id, name, sdk_language, sdk_version, framework, metadata, status, error, checksum, url)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT(id) DO UPDATE SET
    name = excluded.name,
    sdk_language = excluded.sdk_language,
    sdk_version = excluded.sdk_version,
    framework = excluded.framework,
    metadata = excluded.metadata,
    status = excluded.status,
    error = excluded.error,
    checksum = excluded.checksum,
    archived_at = NULL
RETURNING *;

-- name: GetApp :one
SELECT * FROM apps WHERE id = $1;

-- name: GetApps :many
SELECT * FROM apps WHERE archived_at IS NULL;

-- name: GetAppByChecksum :one
SELECT * FROM apps WHERE checksum = $1 AND archived_at IS NULL LIMIT 1;

-- name: GetAppByID :one
SELECT * FROM apps WHERE id = $1 LIMIT 1;

-- name: GetAppByURL :one
SELECT * FROM apps WHERE url = $1 AND archived_at IS NULL LIMIT 1;

-- name: GetAllApps :many
SELECT * FROM apps WHERE archived_at IS NULL;

-- name: DeleteApp :exec
UPDATE apps SET archived_at = CURRENT_TIMESTAMP WHERE id = $1;

-- name: UpdateAppURL :one
UPDATE apps SET url = $1 WHERE id = $2 RETURNING *;

-- name: UpdateAppError :one
UPDATE apps SET error = $1 WHERE id = $2 RETURNING *;


--
-- functions
--


-- name: InsertFunction :one
INSERT INTO functions
    (id, app_id, name, slug, config, created_at) VALUES
    ($1, $2, $3, $4, $5, $6) RETURNING *;

-- name: GetFunctions :many
SELECT functions.*
FROM functions
JOIN apps ON apps.id = functions.app_id
WHERE functions.archived_at IS NULL
AND apps.archived_at IS NULL;

-- name: GetAppFunctions :many
SELECT * FROM functions WHERE app_id = $1 AND archived_at IS NULL;

-- name: GetAppFunctionsBySlug :many
SELECT functions.* FROM functions JOIN apps ON apps.id = functions.app_id WHERE apps.name = $1 AND functions.archived_at IS NULL;

-- name: GetFunctionByID :one
SELECT * FROM functions WHERE id = $1;

-- name: GetFunctionBySlug :one
SELECT * FROM functions WHERE slug = $1 AND archived_at IS NULL;

-- name: UpdateFunctionConfig :one
UPDATE functions SET config = $1, archived_at = NULL WHERE id = $2 RETURNING *;

-- name: DeleteFunctionsByAppID :exec
UPDATE functions SET archived_at = CURRENT_TIMESTAMP WHERE app_id = $1;

-- name: DeleteFunctionsByIDs :exec
UPDATE functions SET archived_at = NOW() WHERE id IN (sqlc.slice('ids'));


--
-- function runs
--


-- name: InsertFunctionRun :exec
INSERT INTO function_runs
    (run_id, run_started_at, function_id, function_version, trigger_type, event_id, batch_id, original_run_id, cron) VALUES
    ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: InsertFunctionFinish :exec
INSERT INTO function_finishes
    (run_id, status, output, completed_step_count, created_at) VALUES
    ($1, $2, $3, $4, $5);

-- name: GetFunctionRun :one
SELECT sqlc.embed(function_runs), sqlc.embed(function_finishes)
  FROM function_runs
  LEFT JOIN function_finishes ON function_finishes.run_id = function_runs.run_id
  WHERE function_runs.run_id = $1;

-- name: GetFunctionRuns :many
SELECT sqlc.embed(function_runs), sqlc.embed(function_finishes) FROM function_runs
LEFT JOIN function_finishes ON function_finishes.run_id = function_runs.run_id;

-- name: GetFunctionRunsTimebound :many
SELECT sqlc.embed(function_runs), sqlc.embed(function_finishes) FROM function_runs
LEFT JOIN function_finishes ON function_finishes.run_id = function_runs.run_id
WHERE function_runs.run_started_at > $1 AND function_runs.run_started_at <= $2
ORDER BY function_runs.run_started_at DESC
LIMIT $3;

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
    ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: InsertEventBatch :exec
INSERT INTO event_batches
    (id, account_id, workspace_id, app_id, workflow_id, run_id, started_at, executed_at, event_ids) VALUES
    ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: GetEventByInternalID :one
SELECT * FROM events WHERE internal_id = $1;

-- name: GetEventsByInternalIDs :many
SELECT * FROM events WHERE internal_id IN (sqlc.slice('ids'));;

-- name: GetEventBatchByRunID :one
SELECT * FROM event_batches WHERE run_id = $1;

-- name: GetEventBatchesByEventID :many
SELECT * FROM event_batches WHERE POSITION(CAST($1 AS TEXT) IN CAST(event_ids AS TEXT)) > 0;

-- name: GetEventsIDbound :many
SELECT DISTINCT e.*
FROM events AS e
LEFT OUTER JOIN function_runs AS r ON r.event_id = e.internal_id
WHERE
    e.internal_id > $1
    AND e.internal_id < $2
    AND (
        r.run_id IS NOT NULL
        OR CASE WHEN e.event_name LIKE 'inngest/%' THEN TRUE ELSE FALSE END = $3
    )
ORDER BY e.internal_id DESC
LIMIT $4;

-- name: WorkspaceEvents :many
SELECT * FROM events WHERE internal_id < $1 AND received_at <= $2 AND received_at >= $3 ORDER BY internal_id DESC LIMIT $4;

-- name: WorkspaceNamedEvents :many
SELECT * FROM events WHERE internal_id < $1 AND received_at <= $2 AND received_at >= $3 AND event_name = $4 ORDER BY internal_id DESC LIMIT $5;


--
-- History
--


-- name: InsertHistory :exec
INSERT INTO history
    (id, created_at, run_started_at, function_id, function_version, run_id, event_id, batch_id, group_id, idempotency_key, type, attempt, latency_ms, step_name, step_id, step_type, url, cancel_request, sleep, wait_for_event, wait_result, invoke_function, invoke_function_result, result) VALUES
    ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24);

-- name: GetHistoryItem :one
SELECT * FROM history WHERE id = $1;

-- name: GetFunctionRunHistory :many
SELECT * FROM history WHERE run_id = $1 ORDER BY created_at ASC;

-- name: HistoryCountRuns :one
SELECT COUNT(DISTINCT run_id) FROM history;


--
-- Traces
--


-- name: InsertTrace :exec
INSERT INTO traces
    (timestamp, timestamp_unix_ms, trace_id, span_id, parent_span_id, trace_state, span_name, span_kind, service_name, resource_attributes, scope_name, scope_version, span_attributes, duration, status_code, status_message, events, links, run_id)
VALUES
    ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19);

-- name: InsertTraceRun :exec
INSERT INTO trace_runs
    (account_id, workspace_id, app_id, function_id, trace_id, run_id, queued_at, started_at, ended_at, status, source_id, trigger_ids, output, batch_id, is_debounce, cron_schedule)
VALUES
    ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16);

-- name: GetTraceRun :one
SELECT * FROM trace_runs WHERE run_id = $1;

-- name: GetTraceSpans :many
SELECT * FROM traces WHERE trace_id = $1 AND run_id = $2 ORDER BY timestamp_unix_ms DESC, duration DESC;

-- name: GetTraceSpanOutput :many
SELECT * FROM traces WHERE trace_id = $1 AND span_id = $2 ORDER BY timestamp_unix_ms DESC, duration DESC;


--
-- Queue snapshots
--


-- name: GetQueueSnapshotChunks :many
SELECT chunk_id, data
FROM queue_snapshot_chunks
WHERE snapshot_id = $1
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
    ($1, $2, $3);

-- name: DeleteOldQueueSnapshots :execrows
DELETE FROM queue_snapshot_chunks
WHERE snapshot_id NOT IN (
    SELECT snapshot_id
    FROM queue_snapshot_chunks
    ORDER BY snapshot_id DESC
    LIMIT $1
);
