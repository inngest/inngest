-- name: UpsertApp :one
INSERT INTO apps (id, name, sdk_language, sdk_version, framework, metadata, status, error, checksum, url, method, app_version)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
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
SELECT * FROM apps WHERE id = $1;

-- name: GetApps :many
SELECT * FROM apps WHERE archived_at IS NULL;

-- name: GetAppByChecksum :one
SELECT * FROM apps WHERE checksum = $1 AND archived_at IS NULL LIMIT 1;

-- name: GetAppByID :one
SELECT * FROM apps WHERE id = $1 LIMIT 1;

-- name: GetAppByURL :one
SELECT * FROM apps WHERE url = $1 AND archived_at IS NULL LIMIT 1;

-- name: GetAppByName :one
SELECT * FROM apps WHERE name = $1 AND archived_at IS NULL LIMIT 1;

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
SELECT sqlc.embed(function_runs),
    COALESCE(function_finishes.status, '') AS finish_status,
    COALESCE(function_finishes.output, '') AS finish_output,
    COALESCE(function_finishes.completed_step_count, 0) AS finish_completed_step_count,
    COALESCE(function_finishes.created_at, function_runs.run_started_at) AS finish_created_at
FROM function_runs
LEFT JOIN function_finishes ON function_finishes.run_id = function_runs.run_id
WHERE function_runs.event_id IN (SELECT UNNEST(sqlc.slice('event_ids')::BYTEA[]));

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
SELECT * FROM events WHERE internal_id = ANY($1::BYTEA[]);

-- name: GetEventBatchByRunID :one
SELECT * FROM event_batches WHERE run_id = CAST($1 AS CHAR(26));

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
    (sqlc.arg('timestamp'), sqlc.arg('timestamp_unix_ms'), sqlc.arg('trace_id'), sqlc.arg('span_id'), sqlc.arg('parent_span_id'), sqlc.arg('trace_state'), sqlc.arg('span_name'), sqlc.arg('span_kind'), sqlc.arg('service_name'), sqlc.arg('resource_attributes'), sqlc.arg('scope_name'), sqlc.arg('scope_version'), sqlc.arg('span_attributes'), sqlc.arg('duration'), sqlc.arg('status_code'), sqlc.arg('status_message'), sqlc.arg('events'), sqlc.arg('links'), sqlc.arg('run_id')::CHAR(26));

-- name: InsertTraceRun :exec
INSERT INTO trace_runs
    (account_id, workspace_id, app_id, function_id, trace_id, run_id, queued_at, started_at, ended_at, status, source_id, trigger_ids, output, batch_id, is_debounce, cron_schedule, has_ai)
VALUES
    (sqlc.arg('account_id'), sqlc.arg('workspace_id'), sqlc.arg('app_id'), sqlc.arg('function_id'), sqlc.arg('trace_id'), sqlc.arg('run_id')::CHAR(26), sqlc.arg('queued_at'), sqlc.arg('started_at'), sqlc.arg('ended_at'), sqlc.arg('status'), sqlc.arg('source_id'), sqlc.arg('trigger_ids'), sqlc.arg('output'), sqlc.arg('batch_id')::BYTEA, sqlc.arg('is_debounce'), sqlc.arg('cron_schedule'), sqlc.arg('has_ai'))
ON CONFLICT (run_id) DO UPDATE SET
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
                WHEN trace_runs.has_ai = TRUE THEN TRUE
                ELSE excluded.has_ai
             END;

-- name: GetTraceRun :one
SELECT * FROM trace_runs WHERE run_id = sqlc.arg('run_id')::CHAR(26);

-- name: GetTraceSpans :many
SELECT * FROM traces WHERE trace_id = sqlc.arg('trace_id') AND run_id = sqlc.arg('run_id')::CHAR(26) ORDER BY timestamp_unix_ms DESC, duration DESC;

-- name: GetTraceSpanOutput :many
SELECT * FROM traces WHERE trace_id = sqlc.arg('trace_id') AND span_id = sqlc.arg('span_id') ORDER BY timestamp_unix_ms DESC, duration DESC;


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

--
-- Worker Connections
--

-- name: InsertWorkerConnection :exec
INSERT INTO worker_connections (
    account_id, workspace_id, app_id, id, gateway_id, instance_id, status, worker_ip, connected_at, last_heartbeat_at, disconnected_at,
    recorded_at, inserted_at, disconnect_reason, group_hash, sdk_lang, sdk_version, sdk_platform, sync_id, app_version, function_count, cpu_cores, mem_bytes, os
)
VALUES (
        sqlc.arg('account_id'),
        sqlc.arg('workspace_id'),
        sqlc.arg('app_id'),
        sqlc.arg('id'),
        sqlc.arg('gateway_id'),
        sqlc.arg('instance_id'),
        sqlc.arg('status'),
        sqlc.arg('worker_ip'),
        sqlc.arg('connected_at'),
        sqlc.arg('last_heartbeat_at'),
        sqlc.arg('disconnected_at'),
        sqlc.arg('recorded_at'),
        sqlc.arg('inserted_at'),
        sqlc.arg('disconnect_reason'),
        sqlc.arg('group_hash'),
        sqlc.arg('sdk_lang'),
        sqlc.arg('sdk_version'),
        sqlc.arg('sdk_platform'),
        sqlc.arg('sync_id'),
        sqlc.arg('app_version'),
        sqlc.arg('function_count'),
        sqlc.arg('cpu_cores'),
        sqlc.arg('mem_bytes'),
        sqlc.arg('os')
        )
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
SELECT * FROM worker_connections WHERE account_id = sqlc.arg('account_id') AND workspace_id = sqlc.arg('workspace_id') AND id = sqlc.arg('connection_id');
