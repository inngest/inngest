-- The views and table macros over the lake tables. They live in
-- inngest_local, a schema in duckdb's own database rather than the lake
-- (an Iceberg catalog can't hold macros or duckdb views), and Migrate
-- re-creates all of them every time it runs, so they're always the current
-- definitions for the current Mode. This is a text/template (see
-- createViews), not a goose migration: each statement ends at a line ending
-- in `;`.

CREATE SCHEMA IF NOT EXISTS inngest_local;

-- raw_runs is the one collapse of inngest.runs (one row per run), unscoped:
-- insights_runs projects over it, and Insights' internal raw mode and the
-- federated executor read it directly. A run's rows collapse by lifecycle
-- stage before recency (final, i.e. ended_at set, beats started beats
-- queued), so a row that arrives out of order can't regress it. Rows
-- that tie (ms timestamps) pick terminal > in-progress > queued status, then
-- status itself, so the pick never depends on insert order. It
-- partitions by account_id/env_id first, so tenant (and run) filters still
-- push below the window into the scan. raw_run_trace_spans is
-- inngest.run_trace_spans as is: spans need no collapse, each row is its own
-- emission.
--
-- The lake stores JSON payloads as JSON text in VARCHAR columns (Iceberg
-- rejects the JSON type, see migrations/000001_baseline.sql). Every macro
-- returns them as JSON (dot access, ->>, JSON results for Insights), but
-- ::JSON is a validating parse per row, so it's only applied to a whole value
-- a macro returns or embeds, in its final projection (after any QUALIFY):
-- the macros' own filters and key extractions (->>, which parses either
-- way) read the text directly.
--
-- A lake whose runs are already one row per run (duckdb.ModeFlatLake: it only
-- holds exported final rows, and a federated read's delta is each run's
-- latest row) skips the collapse, since the window over every run row is
-- most of an Insights query's cost there.
CREATE OR REPLACE MACRO inngest_local.raw_runs(p_runs := 'inngest.runs') AS TABLE
SELECT * REPLACE (attributes::JSON AS attributes, inputs::JSON AS inputs, output::JSON AS output) FROM query_table(p_runs)
{{if .CollapseRuns}}QUALIFY ROW_NUMBER() OVER (PARTITION BY account_id, env_id, run_id ORDER BY ended_at IS NOT NULL DESC, started_at IS NOT NULL DESC, COALESCE(ended_at, started_at, queued_at) DESC, CASE WHEN status IN ('Completed', 'Failed', 'Errored', 'Cancelled', 'TimedOut', 'Skipped') THEN 2 WHEN status IN ('Running', 'Waiting', 'Sleeping', 'Invoking') THEN 1 ELSE 0 END DESC, status DESC) = 1{{end}};

CREATE OR REPLACE MACRO inngest_local.raw_run_trace_spans(p_spans := 'inngest.run_trace_spans') AS TABLE
SELECT * REPLACE (attributes::JSON AS attributes, output::JSON AS output, input::JSON AS input) FROM query_table(p_spans);

-- raw_events is inngest.events as is: an event is never re-written.
CREATE OR REPLACE MACRO inngest_local.raw_events(p_events := 'inngest.events') AS TABLE
SELECT * REPLACE (event_data::JSON AS event_data, event_meta::JSON AS event_meta) FROM query_table(p_events);

-- raw_run_metadata is inngest.run_metadata as is: one row per emission
-- (inngest_local.run_metadata_rollup collapses them).
CREATE OR REPLACE MACRO inngest_local.raw_run_metadata(p_metadata := 'inngest.run_metadata') AS TABLE
SELECT * REPLACE ("values"::JSON AS "values") FROM query_table(p_metadata);

-- inngest_local.run_metadata_rollup is one row per (run_id, span_id) with the
-- latest emission of each kind as user_metadata/internal_metadata (kind ->
-- values). Every metadata write is a full replace of its kind, so there's no
-- merging: an older emission's keys don't survive a newer one, and a null
-- value is kept as null rather than deleting its key the way
-- json_merge_patch (RFC 7386) would. kind is stored without its
-- userland/inngest prefix, so is_user is part of the collapse key or a user
-- "foo" and an internal "foo" would clobber each other. Ties on created_at
-- fall back to the values text so the winner is deterministic;
-- pkg/duckdb/query's GetSpansByRunID collapses with the same ordering.
--
-- It's a table macro over its physical source (query_table, like raw_runs)
-- so a federated executor can roll up lake + buffer emissions together: a
-- span's latest emission of a kind may be on either side.
CREATE OR REPLACE MACRO inngest_local.run_metadata_rollup(p_metadata := 'inngest.run_metadata') AS TABLE
WITH latest AS (
  SELECT *
  FROM query_table(p_metadata)
  QUALIFY ROW_NUMBER() OVER (
    PARTITION BY account_id, env_id, run_id, span_id, is_user, kind
    ORDER BY created_at DESC, values DESC NULLS LAST
  ) = 1
)
SELECT
  account_id,
  env_id,
  run_id,
  max(run_queued_at) as run_queued_at,
  span_id,
  max(scope) as scope,
  max(step_id) as step_id,
  max(step_index) as step_index,
  max(step_attempt) as step_attempt,
  -- values is JSON text: cast to JSON so to_json embeds it as is rather
  -- than encoding it as a string.
  COALESCE(to_json(map_from_entries(list({'k': kind, 'v': values::JSON} ORDER BY kind) FILTER (is_user))), json_object()) as user_metadata,
  COALESCE(to_json(map_from_entries(list({'k': kind, 'v': values::JSON} ORDER BY kind) FILTER (NOT is_user))), json_object()) as internal_metadata,
FROM latest
GROUP BY account_id, env_id, run_id, span_id
;

-- Every insights_* logical table is a parameterized table macro, not a
-- plain view: scoping (p_account_id, p_env_id) is a mandatory call
-- argument pair a caller cannot omit or override with its own WHERE
-- clause -- pkg/duckdb/insights' remapTables stage rewrites every logical
-- table reference to insights_<name>(?, ?), never a bare table/view name
-- -- and account_id/env_id (internal scoping columns, never something a
-- query should see or filter on directly) are excluded from every
-- projection outright rather than merely left unfiltered.

-- The insights_* macros over the run tables read their physical sources by
-- name (query_table), defaulting to the lake tables, so a federated executor
-- can pass lake-plus-buffer relations (e.g. a CTE) instead and reuse each
-- logical table's one definition: its projection, its joins and its
-- collapse. DuckDB plans query_table over a named table like a direct scan:
-- filters and projections still push into it. The runs collapse itself is
-- raw_runs (above).

-- inngest_local.insights_runs(p_account_id, p_env_id) projects raw_runs (one row
-- per run_id) for one tenant, joined to its run-scoped metadata.
CREATE OR REPLACE MACRO inngest_local.insights_runs(p_account_id, p_env_id, p_runs := 'inngest.runs', p_metadata := 'inngest.run_metadata') AS TABLE
SELECT
  r.run_id,
  r.queued_at,
  r.scheduled_at,
  r.started_at,
  r.ended_at,
  r.app_name as app_id,
  r.function_slug as function_id,
  r.status,
  r.event_ids,
  r.sessions,
  r.attributes,
  r.inputs,
  r.output,
  r.is_deferred,
  r.defer_parent_fn_slug AS defer_parent_function_id,
  r.defer_parent_run_ids,
  COALESCE(rmp.user_metadata, json_object()) AS metadata,
  COALESCE(rmp.internal_metadata, json_object()) AS inngest
FROM inngest_local.raw_runs(p_runs := p_runs) r
LEFT JOIN inngest_local.run_metadata_rollup(p_metadata := p_metadata) rmp ON r.account_id = rmp.account_id AND r.env_id = rmp.env_id AND r.run_id = rmp.run_id AND rmp.scope = 'run'
WHERE r.account_id = p_account_id AND r.env_id = p_env_id;

-- Direct passthrough macros -- see docs/plans/010-duckdb-insights-query-layer.md's
-- Logical tables and views table.
CREATE OR REPLACE MACRO inngest_local.insights_events(p_account_id, p_env_id, p_events := 'inngest.events') AS TABLE
SELECT
    event_id as id,
    event_name as name,
    event_data::JSON as data,
    event_v as v,
    event_ts as ts,
    event_meta::JSON as meta,
    received_at,
FROM query_table(p_events)
WHERE account_id = p_account_id AND env_id = p_env_id;

CREATE OR REPLACE MACRO inngest_local.insights_metadata(p_account_id, p_env_id, p_metadata := 'inngest.run_metadata') AS TABLE
SELECT
  run_id,
  run_queued_at,
  span_id,
  scope,
  step_id,
  step_index,
  step_attempt,
  internal_metadata as inngest,
  user_metadata as metadata,
FROM inngest_local.run_metadata_rollup(p_metadata := p_metadata)
WHERE account_id = p_account_id AND env_id = p_env_id;

-- insights_extended_trace_spans is scoped to actual SDK-emitted extended
-- (userland) trace spans -- pkg/tracing/v3/consts.go's
-- SpanNameExtendedTrace = "sdk.extended_trace" -- not every row in
-- run_trace_spans (which also holds executor.run/executor.step/etc.
-- lifecycle spans, exposed separately via insights_steps/
-- insights_step_attempts below).
CREATE OR REPLACE MACRO inngest_local.insights_extended_trace_spans(p_account_id, p_env_id, p_spans := 'inngest.run_trace_spans', p_metadata := 'inngest.run_metadata') AS TABLE
SELECT
  rts.run_id,
  rts.run_queued_at,
  rts.app_name as app_id,
  rts.function_slug as function_id,
  rts.trace_id,
  rts.span_id,
  rts.parent_span_id,
  rts.name,
  rts.start_time,
  rts.end_time,
  rts.attributes::JSON AS attributes,
  COALESCE(rmp.user_metadata, json_object()) AS metadata,
  COALESCE(rmp.internal_metadata, json_object()) AS inngest
FROM query_table(p_spans) rts
LEFT JOIN inngest_local.run_metadata_rollup(p_metadata := p_metadata) rmp ON rts.account_id = rmp.account_id AND rts.env_id = rmp.env_id AND rts.run_id = rmp.run_id AND rts.span_id = rmp.span_id AND rmp.scope = 'extended_trace'
WHERE rts.account_id = p_account_id AND rts.env_id = p_env_id AND name = 'sdk.extended_trace';

-- inngest_local.insights_step_attempts(p_account_id, p_env_id) unpacks the
-- step-specific attributes keys pkg/tracing/meta/attributes.go's Attrs map
-- defines (step.id, step.name, step.op, dynamic.status, etc.) into typed
-- columns, filtered to step-shaped spans -- the span names
-- pkg/tracing/meta/consts.go and pkg/tracing/v3/consts.go define as
-- executor.step*. Every attempt of a step is its own row here;
-- insights_steps (below) collapses to the latest attempt per
-- (run_id, step_id).
--
-- Every extracted key is prefixed "_inngest." -- see
-- pkg/tracing/meta/consts.go's AttrKeyPrefix and serializers.go's
-- withPrefix, which every *Attr constructor in attributes.go's Attrs map
-- applies to its declared key before it's ever written to a span's raw
-- OTel attributes (and therefore to this JSON column) -- e.g.
-- Attrs.StepID ("step.id") is actually stored under the literal key
-- "_inngest.step.id".
--
-- app_id/function_id/run_id are NOT re-derived from attributes here --
-- inngest.run_trace_spans already carries them as real typed columns
-- (see 007's flat-span finding), so this view just passes them through
-- like insights_extended_trace_spans does.
--
-- step_attempt_rows is its body w/ output/input/attributes still JSON text,
-- so insights_steps' collapse runs before the JSON casts.
CREATE OR REPLACE MACRO inngest_local.step_attempt_rows(p_account_id, p_env_id, p_spans := 'inngest.run_trace_spans', p_metadata := 'inngest.run_metadata') AS TABLE
SELECT
  rts.run_id as run_id,
  rts.run_queued_at as run_queued_at,
  rts.app_name as app_id,
  rts.function_slug as function_id,
  rts.span_id as span_id,
  rts.trace_id as trace_id,
  rts.parent_span_id as parent_span_id,
  rts.name as name,
  rts.start_time as start_time,
  rts.end_time as end_time,
  rts.output as output,
  rts.input as input,
  rts.attributes,
  rts.attributes->>'$."_inngest.step.userland.id"' AS step_id,
  TRY_CAST(rts.attributes->>'$."_inngest.step.userland.index"' AS INTEGER) AS step_index,
  TRY_CAST(rts.attributes->>'$."_inngest.step.attempt"' AS INTEGER) AS step_attempt,
  TRY_CAST(rts.attributes->>'$."_inngest.step.max_attempts"' AS INTEGER) AS step_max_attempts,
  rts.attributes->>'$."_inngest.step.type"' AS step_type,
  rts.attributes->>'$."_inngest.dynamic.status"' AS status,
  COALESCE(rmp.user_metadata, json_object()) AS metadata,
  COALESCE(rmp.internal_metadata, json_object()) AS inngest
FROM query_table(p_spans) rts
LEFT JOIN inngest_local.run_metadata_rollup(p_metadata := p_metadata) rmp ON rts.account_id = rmp.account_id AND rts.env_id = rmp.env_id AND rts.run_id = rmp.run_id AND rts.span_id = rmp.span_id AND rmp.scope = 'step'
WHERE rts.account_id = p_account_id AND rts.env_id = p_env_id
  AND rts.name IN ('executor.step', 'executor.step.discovery', 'executor.step.pause_started', 'executor.step.planned');

CREATE OR REPLACE MACRO inngest_local.insights_step_attempts(p_account_id, p_env_id, p_spans := 'inngest.run_trace_spans', p_metadata := 'inngest.run_metadata') AS TABLE
SELECT * REPLACE (output::JSON AS output, input::JSON AS input, attributes::JSON AS attributes)
FROM inngest_local.step_attempt_rows(p_account_id, p_env_id, p_spans := p_spans, p_metadata := p_metadata);

-- inngest_local.insights_steps(p_account_id, p_env_id) collapses
-- insights_step_attempts to the latest attempt per (run_id, step_id) --
-- highest step_attempt, tie-broken by start_time -- mirroring the
-- reference pkg/insights' steps/step_attempts split (docs/plans/010's
-- Logical tables and views table).
CREATE OR REPLACE MACRO inngest_local.insights_steps(p_account_id, p_env_id, p_spans := 'inngest.run_trace_spans', p_metadata := 'inngest.run_metadata') AS TABLE
SELECT * REPLACE (output::JSON AS output, input::JSON AS input, attributes::JSON AS attributes)
FROM inngest_local.step_attempt_rows(p_account_id, p_env_id, p_spans := p_spans, p_metadata := p_metadata)
QUALIFY ROW_NUMBER() OVER (
  PARTITION BY run_id, step_id ORDER BY COALESCE(step_attempt, 0) DESC, start_time DESC
) = 1;
