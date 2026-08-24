-- +goose Up
CREATE TABLE IF NOT EXISTS inngest.runs (
	account_id UUID NOT NULL,
	env_id UUID NOT NULL,
	run_id VARCHAR NOT NULL,
  queued_at TIMESTAMP_MS NOT NULL,
  started_at TIMESTAMP_MS NULL,
  ended_at TIMESTAMP_MS NULL,
  app_id UUID NOT NULL,
	function_id UUID NOT NULL,
  status VARCHAR NOT NULL,
  inputs JSON[] NOT NULL, -- TODO: maybe just plain JSON as duckdb supports top level JSON arrays
  output JSON,
  inserted_at TIMESTAMP_MS NOT NULL DEFAULT current_timestamp
);
ALTER TABLE inngest.runs
  SET SORTED BY (year(queued_at), month(queued_at), account_id, env_id, run_id, queued_at);
ALTER TABLE inngest.runs
  SET PARTITIONED BY (year(queued_at), month(queued_at), account_id);

-- NOTE: janky plain parquet globbed view. Can't be created until we have at least some data
-- (not duckinngest compatible)
-- CREATE VIEW runs AS
-- SELECT
--   account_id,
--   env_id,
--   run_id,
--   queued_at,
--   started_at,
--   ended_at,
--   app_id,
--   function_id,
--   status,
--   inputs,
--   output
-- FROM read_parquet(getvariable('DATA_PATH') || '/runs/*/*/*/*.parquet', hive_partitioning = true);

CREATE TABLE IF NOT EXISTS inngest.run_trace_spans (
	account_id UUID NOT NULL,
	env_id UUID NOT NULL,
	run_id VARCHAR NOT NULL,
  run_queued_at TIMESTAMP_MS NOT NULL,
  app_id UUID NOT NULL,
	function_id UUID NOT NULL,
  start_time TIMESTAMP_MS NOT NULL,
  end_time TIMESTAMP_MS NOT NULL,
  trace_id VARCHAR NOT NULL,
  span_id VARCHAR NOT NULL,
  parent_span_id VARCHAR,
  attributes JSON NOT NULL,
);
ALTER TABLE inngest.run_trace_spans
  SET SORTED BY (year(run_queued_at), month(run_queued_at), account_id, env_id, run_id, start_time, end_time);
ALTER TABLE inngest.run_trace_spans
  SET PARTITIONED BY (year(run_queued_at), month(run_queued_at), account_id);

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
ALTER TABLE inngest.events
  SET SORTED BY (year(received_at), month(received_at), account_id, env_id, internal_id, received_at);
ALTER TABLE inngest.events
  SET PARTITIONED BY (year(received_at), month(received_at), account_id);

-- +goose Down
DROP TABLE IF EXISTS inngest.runs;
DROP TABLE IF EXISTS inngest.run_trace_spans;
DROP TABLE IF EXISTS inngest.events;
