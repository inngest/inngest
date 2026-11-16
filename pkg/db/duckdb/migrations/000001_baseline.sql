-- +goose Up
-- The lake tables. On an Iceberg lake they're format-version 3, partitioned
-- by account + month, and set (DUCKDB_ICEBERG_ONLY is empty in-memory):
--   * commit.manifest-merge.enabled=false: merging reads every manifest on
--     each commit until duckdb/duckdb-iceberg#1650 ships.
--   * commit.retry.*: duckdb's defaults (4 retries, up to 60s backoff)
--     starve concurrent writers.
--   * zstd: ~35% smaller than duckdb's snappy default.
-- No SORTED BY (sorting each small append only slows inserts), compaction
-- owns sorting. JSON payloads are JSON text in VARCHAR columns (Iceberg
-- rejects duckdb's JSON type), the views cast them back.
--
-- bucket_at is when the row reached the buffer (Kafka LogAppendTime, or the
-- insert time here): federated reads take the lake for bucket_at <= W and
-- the buffer above it. Its default only matters in-memory/tests, Iceberg
-- freezes defaults at CREATE time so writers always set it.
-- +goose ENVSUB ON
CREATE TABLE IF NOT EXISTS inngest.runs (
	account_id UUID NOT NULL,
	env_id UUID NOT NULL,
	run_id VARCHAR NOT NULL,
  queued_at TIMESTAMPTZ NOT NULL,
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
  output VARCHAR,
  error VARCHAR,
  event_ids VARCHAR[],
  sessions STRUCT(key VARCHAR, id VARCHAR)[],
  is_deferred BOOLEAN NOT NULL DEFAULT FALSE,
  defer_parent_fn_slug VARCHAR,
  defer_parent_run_ids VARCHAR[],
  trace_id VARCHAR,
  bucket_at TIMESTAMPTZ NOT NULL DEFAULT make_timestamp_ms(epoch_ms(current_timestamp))
) ${DUCKDB_ICEBERG_ONLY-PARTITIONED BY (account_id, month(queued_at)) WITH ('format-version' = '3', 'commit.manifest-merge.enabled' = 'false', 'commit.retry.num-retries' = '100', 'commit.retry.max-wait-ms' = '200', 'write.parquet.compression-codec' = 'zstd')};
-- +goose ENVSUB OFF

-- One row per metadata write, joined onto run_trace_spans by
-- (run_id, span_id). Each write replaces its kind, so the current value is
-- the latest row per (run_id, span_id, is_user, kind).
-- +goose ENVSUB ON
CREATE TABLE IF NOT EXISTS inngest.run_metadata (
  account_id     UUID NOT NULL,
  env_id         UUID NOT NULL,
  run_id         VARCHAR NOT NULL,
  run_queued_at  TIMESTAMPTZ NOT NULL,
  span_id        VARCHAR NOT NULL,
  scope          VARCHAR NOT NULL,
  step_id        VARCHAR,
  step_index     INTEGER,
  step_attempt   INTEGER,
  kind           VARCHAR NOT NULL,
  is_user        BOOLEAN NOT NULL,
  values         VARCHAR NOT NULL,
  created_at     TIMESTAMPTZ NOT NULL,
  bucket_at      TIMESTAMPTZ NOT NULL DEFAULT make_timestamp_ms(epoch_ms(current_timestamp))
) ${DUCKDB_ICEBERG_ONLY-PARTITIONED BY (account_id, month(run_queued_at)) WITH ('format-version' = '3', 'commit.manifest-merge.enabled' = 'false', 'commit.retry.num-retries' = '100', 'commit.retry.max-wait-ms' = '200', 'write.parquet.compression-codec' = 'zstd')};
-- +goose ENVSUB OFF

-- output/error are the SDK's output envelope split at write time
-- (cqrs.SplitSpanOutputEnvelope). runs rows are built from these spans
-- (pkg/duckdb/tracing's runsInsertSQL) since lake tables have no triggers.
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
  output VARCHAR,
  error VARCHAR,
  input VARCHAR,
  bucket_at TIMESTAMPTZ NOT NULL DEFAULT make_timestamp_ms(epoch_ms(current_timestamp))
) ${DUCKDB_ICEBERG_ONLY-PARTITIONED BY (account_id, month(run_queued_at)) WITH ('format-version' = '3', 'commit.manifest-merge.enabled' = 'false', 'commit.retry.num-retries' = '100', 'commit.retry.max-wait-ms' = '200', 'write.parquet.compression-codec' = 'zstd')};
-- +goose ENVSUB OFF

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
  bucket_at TIMESTAMPTZ NOT NULL DEFAULT make_timestamp_ms(epoch_ms(current_timestamp))
) ${DUCKDB_ICEBERG_ONLY-PARTITIONED BY (account_id, month(received_at)) WITH ('format-version' = '3', 'commit.manifest-merge.enabled' = 'false', 'commit.retry.num-retries' = '100', 'commit.retry.max-wait-ms' = '200', 'write.parquet.compression-codec' = 'zstd')};
-- +goose ENVSUB OFF

-- +goose Down
DROP TABLE IF EXISTS inngest.runs;
DROP TABLE IF EXISTS inngest.run_metadata;
DROP TABLE IF EXISTS inngest.run_trace_spans;
DROP TABLE IF EXISTS inngest.events;
