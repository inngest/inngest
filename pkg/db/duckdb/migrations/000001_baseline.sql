-- +goose Up
-- The lake tables. On an Iceberg lake each one is format-version 3,
-- partitioned by account + month of its run/receive time, and
-- declares a sort order duckdb sorts each write by. DUCKDB_ICEBERG_ONLY is
-- set empty for the in-memory mode, whose plain duckdb tables take none of
-- that. Other Iceberg writers (IE the cloud exporter) have to sort
-- explicitly if they don't honor the table's sort order.
--
-- NOTE: Iceberg stores a column default as a literal, so duckdb freezes
-- bucket_at's current_timestamp default at CREATE TABLE time on a lake. It
-- only stays as a convenience for the in-memory mode and test fixtures:
-- real writers (pkg/duckdb/tracing's inserter, the exporter) set bucket_at
-- explicitly.
--
-- JSON payloads (attributes, inputs/input, output, error, values, event_data,
-- event_meta) are stored as JSON text in plain VARCHAR columns: Iceberg
-- rejects duckdb's JSON type (and duckdb doesn't keep the alias on a string
-- column), and duckdb's shredded VARIANT is a net loss here (no UINT64
-- shredding (duckdb/duckdb#28166), integers > INT64_MAX fail parquet writes
-- (duckdb/duckdb#28168), a parquet column per distinct key, slow writes and
-- whole-value reads). The in-memory mode uses the same type so both modes
-- share one DDL. Everything above the tables reads them as JSON: the
-- views.sql macros cast them back (free, JSON is an alias of VARCHAR).
--
-- Every table also sets:
--   * 'commit.manifest-merge.enabled' = 'false': manifest merge reads every
--     existing manifest on each commit until duckdb/duckdb-iceberg#1650
--     ships, drop this once the extension has it. The tradeoff is a
--     manifest per commit piling up, so reads slow down over time (~35ms ->
--     ~95ms for a point query after 1000 commits) until we compact.
--   * 'commit.retry.num-retries' = '100', 'commit.retry.max-wait-ms' = '200':
--     duckdb's own retry on a 409 commit conflict, whose defaults (4 retries,
--     up to 60s backoff) starve concurrent writers.
--   * 'write.parquet.compression-codec' = 'zstd': duckdb writes snappy by
--     default. zstd is ~35% smaller (2634MB -> 1705MB of data at 10M seeded
--     runs) for ~5% slower inserts, reads are about the same.
-- +goose ENVSUB ON
CREATE TABLE IF NOT EXISTS inngest.runs (
	account_id UUID NOT NULL,
	env_id UUID NOT NULL,
	run_id VARCHAR NOT NULL,
  queued_at TIMESTAMPTZ NOT NULL,
  -- When the run was due to start, fixed at schedule time (from run state;
  -- queued_at for a run whose state predates it).
  scheduled_at TIMESTAMPTZ NOT NULL,
  started_at TIMESTAMPTZ NULL,
  ended_at TIMESTAMPTZ NULL,
  app_id UUID NOT NULL,
  app_name VARCHAR NOT NULL,
	function_id UUID NOT NULL,
  function_slug VARCHAR NOT NULL,
  status VARCHAR NOT NULL,
  attributes VARCHAR NOT NULL,
  inputs VARCHAR NOT NULL,
  -- The run's output, split like run_trace_spans' (below).
  output VARCHAR,
  error VARCHAR,
  -- event_ids holds the run's triggering event(s) internal ULID(s) — see
  -- pkg/duckdb/tracing/listener.go's addRunEventAttrs, which sources it
  -- from sv2.Metadata.Config.EventIDs (the same persisted field
  -- pkg/run/trace_lifecycle.go's OTel "sys.event.ids" span attribute is
  -- derived from for the SQLite/Postgres path). NULL for cron-only runs,
  -- which have no triggering event at all.
  event_ids VARCHAR[],
  -- sessions is the run-level form of pkg/tracing/meta.EventSessions (the
  -- JSON "event.sessions" attribute on the real run span) — one (key, id)
  -- pair per session a triggering event tagged this run with, sourced from
  -- pkg/duckdb/tracing/listener.go's addRunEventAttrs, sorted/deduped/
  -- capped at consts.MaxRunSessions the same way
  -- pkg/execution/executor/executor.go's normalizeRunSessions builds the
  -- real span's attribute. NULL when no triggering event carried any
  -- session tag.
  sessions STRUCT(key VARCHAR, id VARCHAR)[],
  is_deferred BOOLEAN NOT NULL DEFAULT FALSE,
  defer_parent_fn_slug VARCHAR,
  defer_parent_run_ids VARCHAR[],
  -- The run's OTel trace ID, copied from the span row materializeRuns builds
  -- each runs row from, so run listings can report it without joining
  -- run_trace_spans.
  trace_id VARCHAR,
  -- When the row became visible in the buffer: the ClickHouse buffer's Kafka
  -- LogAppendTime for exported rows, the insert time for rows written here
  -- directly. Federated reads take the lake for bucket_at <= W and the buffer
  -- for bucket_at > W, so any W both stores cover gives the same result.
  -- A run row's is its source span's.
  bucket_at TIMESTAMPTZ NOT NULL DEFAULT make_timestamp_ms(epoch_ms(current_timestamp))
) ${DUCKDB_ICEBERG_ONLY-PARTITIONED BY (account_id, month(queued_at)) SORTED BY (account_id, env_id, run_id, queued_at) WITH ('format-version' = '3', 'commit.manifest-merge.enabled' = 'false', 'commit.retry.num-retries' = '100', 'commit.retry.max-wait-ms' = '200', 'write.parquet.compression-codec' = 'zstd')};
-- +goose ENVSUB OFF

-- inngest.run_metadata holds one row per metadata emission (executor
-- generator/experiment metadata, the AddRunMetadata API, and userland span
-- metadata extraction -- see pkg/execution/sync_lifecycle.go's
-- OnMetadataEntry and its callers). It is a standalone table, joined onto
-- inngest.run_trace_spans by (run_id, span_id) rather than a set of columns
-- on that table, mirroring the reference metadata insights table's own
-- separation of concerns.
--
-- This table has no metadata "op" (merge/set/delete/add): every metadata
-- write is a full replace of its kind, so each row is the full value for
-- its kind. kind is stored without its userland/inngest prefix (is_user
-- tells them apart). A reader wanting "the current metadata for this span"
-- takes the latest row per (run_id, span_id, is_user, kind) (see
-- inngest_local.run_metadata_rollup) rather than replaying an op history.
-- +goose ENVSUB ON
CREATE TABLE IF NOT EXISTS inngest.run_metadata (
  account_id     UUID NOT NULL,
  env_id         UUID NOT NULL,
  run_id         VARCHAR NOT NULL,
  run_queued_at  TIMESTAMPTZ NOT NULL,
  -- TODO: decide whether we can add app/function ids here
  -- span_id is the run/step/request span this metadata annotates (the
  -- "parent" every metadata span is created under) -- the join key onto
  -- inngest.run_trace_spans.span_id, not this row's own identity.
  span_id        VARCHAR NOT NULL,
  scope          VARCHAR NOT NULL,
  -- step_id/step_index/step_attempt identify the step this metadata belongs
  -- to, independently of span_id (a request-scoped metadata span's span_id
  -- is the request's execution span, not the step span, but it still
  -- belongs to a step). All three are NULL for run-scoped metadata.
  -- step_id is the same hashed step ID used to compute a step's own
  -- deterministic span identity, not the SDK-facing userland step ID.
  step_id        VARCHAR,
  step_index     INTEGER,
  step_attempt   INTEGER,
  kind           VARCHAR NOT NULL,
  is_user        BOOLEAN NOT NULL,
  values         VARCHAR NOT NULL,
  created_at     TIMESTAMPTZ NOT NULL,
  -- When the row became visible in the buffer, as on inngest.runs: the
  -- federation split (lake bucket_at <= W, buffer bucket_at > W).
  bucket_at      TIMESTAMPTZ NOT NULL DEFAULT make_timestamp_ms(epoch_ms(current_timestamp))
) ${DUCKDB_ICEBERG_ONLY-PARTITIONED BY (account_id, month(run_queued_at)) SORTED BY (account_id, env_id, run_id, scope, step_id, step_index, step_attempt, span_id, kind) WITH ('format-version' = '3', 'commit.manifest-merge.enabled' = 'false', 'commit.retry.num-retries' = '100', 'commit.retry.max-wait-ms' = '200', 'write.parquet.compression-codec' = 'zstd')};
-- +goose ENVSUB OFF


-- Column set mirrors pkg/db/sqlite's `spans` table (see
-- pkg/duckdb/tracing/tracing.go's SpanExporter doc comment), minus the
-- dynamic-span rollup columns (dynamic_span_id, status, event_ids,
-- is_deferred) — this table only ever holds flat, non-dynamic spans, so
-- those values live in `attributes` like any other span attribute instead
-- of getting a dedicated column.
-- +goose ENVSUB ON
CREATE TABLE IF NOT EXISTS inngest.run_trace_spans (
	account_id UUID NOT NULL,
	env_id UUID NOT NULL,
	run_id VARCHAR NOT NULL,
  run_queued_at TIMESTAMPTZ NOT NULL,
  app_id UUID NOT NULL,
  app_name VARCHAR NOT NULL,
	function_id UUID NOT NULL,
  function_slug VARCHAR NOT NULL,
  name VARCHAR NOT NULL,
  start_time TIMESTAMPTZ NOT NULL,
  end_time TIMESTAMPTZ NOT NULL,
  trace_id VARCHAR NOT NULL,
  span_id VARCHAR NOT NULL,
  parent_span_id VARCHAR,
  attributes VARCHAR NOT NULL,
  -- The SDK's output envelope, split once at write time
  -- (cqrs.SplitSpanOutputEnvelope): output is the {"data": ...} value and
  -- error the {"error": ...} value, NULL unless the span failed. A
  -- step.waitForEvent-shaped or unrecognized output is all output.
  output VARCHAR,
  error VARCHAR,
  input VARCHAR,
  -- When the row became visible in the buffer: the ClickHouse buffer's Kafka
  -- LogAppendTime for exported rows, the insert time for rows written here
  -- directly. Federated reads take the lake for bucket_at <= W and the buffer
  -- for bucket_at > W, so any W both stores cover gives the same result.
  bucket_at TIMESTAMPTZ NOT NULL DEFAULT make_timestamp_ms(epoch_ms(current_timestamp))
) ${DUCKDB_ICEBERG_ONLY-PARTITIONED BY (account_id, month(run_queued_at)) SORTED BY (account_id, env_id, run_id, start_time, end_time) WITH ('format-version' = '3', 'commit.manifest-merge.enabled' = 'false', 'commit.retry.num-retries' = '100', 'commit.retry.max-wait-ms' = '200', 'write.parquet.compression-codec' = 'zstd')};
-- +goose ENVSUB OFF

-- CREATE TRIGGER inngest_runs_mv AFTER INSERT ON inngest.run_trace_spans is
-- commented out for now: lake tables don't support triggers ("Not
-- implemented Error: Triggers are not supported for this table type"). Its
-- logic is instead run as an explicit secondary query on every span batch
-- insertion in the duckdb dual-write path — see
-- pkg/duckdb/tracing/batch.go's materializeRuns.
--
-- CREATE TRIGGER inngest_runs_mv AFTER INSERT ON inngest.run_trace_spans
-- REFERENCING NEW TABLE AS new_rows
-- FOR EACH STATEMENT
--   INSERT INTO inngest.runs
--   SELECT
--     account_id,
--     env_id,
--     run_id,
--     run_queued_at AS queued_at,
--     COALESCE(make_timestamp_ms(TRY_CAST(attributes->>'_inngest.scheduled_at' AS BIGINT)), run_queued_at) AS scheduled_at,
--     make_timestamp_ms(TRY_CAST(attributes->>'_inngest.started_at' AS BIGINT)) AS started_at,
--     make_timestamp_ms(TRY_CAST(attributes->>'_inngest.ended_at' AS BIGINT)) AS ended_at,
--     app_id,
--     app_name,
--     function_id,
--     function_slug,
--     attributes->>'_inngest.dynamic.status' AS status,
--     attributes,
--     input as inputs,
--     output,
--     error,
--     TRY_CAST(attributes ->> '_inngest.event.ids' AS VARCHAR[]) AS event_ids,
--     TRY_CAST (attributes ->> '_inngest.event.sessions' AS STRUCT(key VARCHAR, id VARCHAR)[]) AS sessions,
--     attributes->>'_inngest.defer.parent_fn_slug' IS NOT NULL AS is_deferred,
--     attributes->>'_inngest.defer.parent_fn_slug' AS defer_parent_fn_slug,
--     TRY_CAST(attributes->>'_inngest.defer.parent_run_ids' AS VARCHAR[]) AS defer_parent_run_ids,
--     current_timestamp,
--   FROM new_rows
--   WHERE name IN ('executor.run', 'executor.run.queued', 'executor.run.started')
-- ;


-- +goose ENVSUB ON
CREATE TABLE IF NOT EXISTS inngest.events (
	account_id UUID NOT NULL,
	env_id UUID NOT NULL,
  internal_id VARCHAR NOT NULL,
  received_at TIMESTAMPTZ NOT NULL,
  source VARCHAR NOT NULL,
  source_id VARCHAR NULL,
	event_id VARCHAR NOT NULL,
	event_name VARCHAR NOT NULL,
  event_data VARCHAR NOT NULL,
  event_v VARCHAR NOT NULL,
  event_ts TIMESTAMPTZ NOT NULL,
  event_meta VARCHAR NOT NULL,
  -- When the row became visible in the buffer: the ClickHouse buffer's Kafka
  -- LogAppendTime for exported rows, the insert time for rows written here
  -- directly. Federated reads take the lake for bucket_at <= W and the buffer
  -- for bucket_at > W, so any W both stores cover gives the same result.
  bucket_at TIMESTAMPTZ NOT NULL DEFAULT make_timestamp_ms(epoch_ms(current_timestamp))
) ${DUCKDB_ICEBERG_ONLY-PARTITIONED BY (account_id, month(received_at)) SORTED BY (account_id, env_id, internal_id, received_at) WITH ('format-version' = '3', 'commit.manifest-merge.enabled' = 'false', 'commit.retry.num-retries' = '100', 'commit.retry.max-wait-ms' = '200', 'write.parquet.compression-codec' = 'zstd')};
-- +goose ENVSUB OFF

-- +goose Down
DROP TABLE IF EXISTS inngest.runs;
DROP TABLE IF EXISTS inngest.run_metadata;
DROP TABLE IF EXISTS inngest.run_trace_spans;
DROP TABLE IF EXISTS inngest.events;
