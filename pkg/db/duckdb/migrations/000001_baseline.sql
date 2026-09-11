-- +goose Up
CREATE TABLE IF NOT EXISTS inngest.runs (
	account_id UUID NOT NULL,
	env_id UUID NOT NULL,
	run_id VARCHAR NOT NULL,
  queued_at TIMESTAMP_MS NOT NULL,
  scheduled_at TIMESTAMP_MS NULL,
  started_at TIMESTAMP_MS NULL,
  ended_at TIMESTAMP_MS NULL,
  app_id UUID NOT NULL,
  app_name VARCHAR NOT NULL,
	function_id UUID NOT NULL,
  function_slug VARCHAR NOT NULL,
  status VARCHAR NOT NULL,
  attributes JSON NOT NULL,
  inputs JSON NOT NULL,
  output JSON,
  -- event_ids holds the run's triggering event(s) internal ULID(s) — see
  -- pkg/execution/dualwrite/listener.go's addRunEventAttrs, which sources it
  -- from sv2.Metadata.Config.EventIDs (the same persisted field
  -- pkg/run/trace_lifecycle.go's OTel "sys.event.ids" span attribute is
  -- derived from for the SQLite/Postgres path). NULL for cron-only runs,
  -- which have no triggering event at all.
  event_ids VARCHAR[],
  -- sessions is the run-level form of pkg/tracing/meta.EventSessions (the
  -- JSON "event.sessions" attribute on the real run span) — one (key, id)
  -- pair per session a triggering event tagged this run with, sourced from
  -- pkg/execution/dualwrite/listener.go's addRunEventAttrs, sorted/deduped/
  -- capped at consts.MaxRunSessions the same way
  -- pkg/execution/executor/executor.go's normalizeRunSessions builds the
  -- real span's attribute. NULL when no triggering event carried any
  -- session tag.
  sessions STRUCT(key VARCHAR, id VARCHAR)[],
  is_deferred BOOLEAN NOT NULL DEFAULT FALSE,
  defer_parent_fn_slug VARCHAR,
  defer_parent_run_ids VARCHAR[],
  inserted_at TIMESTAMP_MS NOT NULL DEFAULT current_timestamp
);
-- +goose ENVSUB ON
${DUCKDB_DUCKLAKE_ONLY-ALTER TABLE inngest.runs SET SORTED BY (year(queued_at), month(queued_at), account_id, env_id, run_id, queued_at)};
${DUCKDB_DUCKLAKE_ONLY-ALTER TABLE inngest.runs SET PARTITIONED BY (year(queued_at), month(queued_at), account_id)};
-- +goose ENVSUB OFF

-- inngest.run_metadata holds one row per metadata emission (executor
-- generator/experiment metadata, the AddRunMetadata API, and userland span
-- metadata extraction -- see pkg/execution/sync_lifecycle.go's
-- OnMetadataEntry and its callers). It is a standalone table, joined onto
-- inngest.run_trace_spans by (run_id, span_id) rather than a set of columns
-- on that table, mirroring the reference metadata insights table's own
-- separation of concerns.
--
-- Deliberately simpler than that reference design: this table drops the
-- concept of a metadata "op" (merge/set/delete/add) entirely. Each row is
-- the caller's full, already-resolved value set for one metadata emission,
-- not a delta to be folded with prior emissions by key. A reader wanting
-- "the current metadata for this span" collapses to the latest row per
-- (run_id, span_id, kind) -- the same latest-row-wins pattern
-- pkg/cqrs/duckdbquery already uses for inngest.runs -- rather than
-- replaying an op history.
CREATE TABLE IF NOT EXISTS inngest.run_metadata (
  account_id     UUID NOT NULL,
  env_id         UUID NOT NULL,
  run_id         VARCHAR NOT NULL,
  run_queued_at  TIMESTAMP_MS NOT NULL,
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
  values         JSON NOT NULL,
  created_at     TIMESTAMP_MS NOT NULL
);
-- +goose ENVSUB ON
${DUCKDB_DUCKLAKE_ONLY-ALTER TABLE inngest.run_metadata SET SORTED BY (year(run_queued_at), month(run_queued_at), account_id, env_id, run_id, scope, step_id, step_index, step_attempt, span_id, kind)};
${DUCKDB_DUCKLAKE_ONLY-ALTER TABLE inngest.run_metadata SET PARTITIONED BY (year(run_queued_at), month(run_queued_at), account_id)};
-- +goose ENVSUB OFF


-- Column set mirrors pkg/db/sqlite's `spans` table (see
-- pkg/execution/dualwrite/span_exporter.go's doc comment), minus the
-- dynamic-span rollup columns (dynamic_span_id, status, event_ids,
-- is_deferred) — this table only ever holds flat, non-dynamic spans, so
-- those values live in `attributes` like any other span attribute instead
-- of getting a dedicated column.
CREATE TABLE IF NOT EXISTS inngest.run_trace_spans (
	account_id UUID NOT NULL,
	env_id UUID NOT NULL,
	run_id VARCHAR NOT NULL,
  run_queued_at TIMESTAMP_MS NOT NULL,
  app_id UUID NOT NULL,
  app_name VARCHAR NOT NULL,
	function_id UUID NOT NULL,
  function_slug VARCHAR NOT NULL,
  name VARCHAR NOT NULL,
  start_time TIMESTAMP_MS NOT NULL,
  end_time TIMESTAMP_MS NOT NULL,
  trace_id VARCHAR NOT NULL,
  span_id VARCHAR NOT NULL,
  parent_span_id VARCHAR,
  attributes JSON NOT NULL,
  links JSON,
  output JSON,
  input JSON,
);
-- +goose ENVSUB ON
${DUCKDB_DUCKLAKE_ONLY-ALTER TABLE inngest.run_trace_spans SET SORTED BY (year(run_queued_at), month(run_queued_at), account_id, env_id, run_id, start_time, end_time)};
${DUCKDB_DUCKLAKE_ONLY-ALTER TABLE inngest.run_trace_spans SET PARTITIONED BY (year(run_queued_at), month(run_queued_at), account_id)};
-- +goose ENVSUB OFF

-- CREATE TRIGGER inngest_runs_mv AFTER INSERT ON inngest.run_trace_spans is
-- commented out for now: DuckLake's own table type does not support triggers
-- ("Not implemented Error: Triggers are not supported for this table type"
-- against a real DuckLake-attached table — see
-- pkg/execution/dualwrite/helpers_test.go's newTestDuckDB doc comment). Its
-- logic is instead run as an explicit secondary query on every span batch
-- insertion in the duckdb dualwrite path — see
-- pkg/execution/dualwrite/batch.go's materializeRuns.
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
--     make_timestamp_ms(TRY_CAST(attributes->>'_inngest.scheduled_at' AS BIGINT)) AS scheduled_at,
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
--     TRY_CAST(attributes ->> '_inngest.event.ids' AS VARCHAR[]) AS event_ids,
--     TRY_CAST (attributes ->> '_inngest.event.sessions' AS STRUCT(key VARCHAR, id VARCHAR)[]) AS sessions,
--     attributes->>'_inngest.defer.parent_fn_slug' IS NOT NULL AS is_deferred,
--     attributes->>'_inngest.defer.parent_fn_slug' AS defer_parent_fn_slug,
--     TRY_CAST(attributes->>'_inngest.defer.parent_run_ids' AS VARCHAR[]) AS defer_parent_run_ids,
--     current_timestamp,
--   FROM new_rows
--   WHERE name IN ('executor.run', 'executor.run.queued', 'executor.run.started')
-- ;


CREATE TABLE IF NOT EXISTS inngest.events (
	account_id UUID NOT NULL,
	env_id UUID NOT NULL,
  internal_id VARCHAR NOT NULL,
  received_at TIMESTAMP_MS NOT NULL,
  source VARCHAR NOT NULL,
  source_id VARCHAR NULL,
	event_id VARCHAR NOT NULL,
	event_name VARCHAR NOT NULL,
  event_data JSON NOT NULL DEFAULT '{}',
  event_v VARCHAR NOT NULL,
  event_ts TIMESTAMP_MS NOT NULL,
  event_meta JSON NOT NULL DEFAULT '{}',
);
-- +goose ENVSUB ON
${DUCKDB_DUCKLAKE_ONLY-ALTER TABLE inngest.events SET SORTED BY (year(received_at), month(received_at), account_id, env_id, internal_id, received_at)};
${DUCKDB_DUCKLAKE_ONLY-ALTER TABLE inngest.events SET PARTITIONED BY (year(received_at), month(received_at), account_id)};
-- +goose ENVSUB OFF

-- +goose Down
DROP TABLE IF EXISTS inngest.runs;
DROP TABLE IF EXISTS inngest.run_metadata;
DROP TABLE IF EXISTS inngest.run_trace_spans;
DROP TABLE IF EXISTS inngest.events;
