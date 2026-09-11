-- +goose Up

CREATE VIEW inngest.run_metadata_rollup AS
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
  reduce(coalesce(list(json_object(kind, values) ORDER BY created_at) FILTER (is_user), []), lambda a,b: json_merge_patch(a,b), '{}') as user_metadata,
  reduce(coalesce(list(json_object(kind, values) ORDER BY created_at) FILTER (NOT is_user), []), lambda a,b: json_merge_patch(a,b), '{}') as internal_metadata,
FROM run_metadata
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

-- inngest.insights_runs(p_account_id, p_env_id) collapses inngest.runs to
-- one row per run_id -- the same latest-row-wins pattern
-- pkg/cqrs/duckdbquery's latestRunsCTE uses (pkg/cqrs/duckdbquery/runs.go),
-- exposed as a macro instead of reimplemented ad hoc. QUALIFY filters on
-- the window function directly, no subquery wrapper needed.
CREATE MACRO inngest.insights_runs(p_account_id, p_env_id) AS TABLE
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
FROM inngest.runs r
LEFT JOIN inngest.run_metadata_rollup rmp ON r.account_id = rmp.account_id AND r.env_id = rmp.env_id AND r.run_id = rmp.run_id AND rmp.scope = 'run'
WHERE r.account_id = p_account_id AND r.env_id = p_env_id
QUALIFY ROW_NUMBER() OVER (
  PARTITION BY r.run_id ORDER BY COALESCE(ended_at, started_at, queued_at) DESC
) = 1;

-- Direct passthrough macros -- see docs/plans/010-duckdb-insights-query-layer.md's
-- Logical tables and views table.
CREATE MACRO inngest.insights_events(p_account_id, p_env_id) AS TABLE
SELECT
    event_id as id,
    event_name as name,
    event_data as data,
    event_v as v,
    event_ts as ts,
    event_meta as meta,
    received_at,
FROM inngest.events
WHERE account_id = p_account_id AND env_id = p_env_id;

CREATE MACRO inngest.insights_metadata(p_account_id, p_env_id) AS TABLE
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
FROM inngest.run_metadata_rollup
WHERE account_id = p_account_id AND env_id = p_env_id;

-- insights_extended_trace_spans is scoped to actual SDK-emitted extended
-- (userland) trace spans -- pkg/tracing/v3/consts.go's
-- SpanNameExtendedTrace = "sdk.extended_trace" -- not every row in
-- run_trace_spans (which also holds executor.run/executor.step/etc.
-- lifecycle spans, exposed separately via insights_steps/
-- insights_step_attempts below).
CREATE MACRO inngest.insights_extended_trace_spans(p_account_id, p_env_id) AS TABLE
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
  rts.attributes,
  COALESCE(rmp.user_metadata, json_object()) AS metadata,
  COALESCE(rmp.internal_metadata, json_object()) AS inngest
FROM inngest.run_trace_spans rts
LEFT JOIN inngest.run_metadata_rollup rmp ON rts.account_id = rmp.account_id AND rts.env_id = rmp.env_id AND rts.run_id = rmp.run_id AND rts.span_id = rmp.span_id AND rmp.scope = 'extended_trace'
WHERE rts.account_id = p_account_id AND rts.env_id = p_env_id AND name = 'sdk.extended_trace';

-- inngest.insights_step_attempts(p_account_id, p_env_id) unpacks the
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
CREATE MACRO inngest.insights_step_attempts(p_account_id, p_env_id) AS TABLE
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
  rts.attributes ->> '_inngest.step.userland.id' AS step_id,
  TRY_CAST(rts.attributes ->> '_inngest.step.userland.index' AS INTEGER) AS step_index,
  TRY_CAST(rts.attributes ->> '_inngest.step.attempt' AS INTEGER) AS step_attempt,
  TRY_CAST(rts.attributes ->> '_inngest.step.max_attempts' AS INTEGER) AS step_max_attempts,
  rts.attributes ->> '_inngest.step.type' AS step_type,
  -- The step's own status, mirroring cqrs.ApplyExtractedSpanAttributes'
  -- promotion of Attributes.DynamicStatus onto OtelSpan.Status.
  rts.attributes ->> '_inngest.dynamic.status' AS status,
  COALESCE(rmp.user_metadata, json_object()) AS metadata,
  COALESCE(rmp.internal_metadata, json_object()) AS inngest
FROM inngest.run_trace_spans rts
LEFT JOIN inngest.run_metadata_rollup rmp ON rts.account_id = rmp.account_id AND rts.env_id = rmp.env_id AND rts.run_id = rmp.run_id AND rts.span_id = rmp.span_id AND rmp.scope = 'step'
WHERE rts.account_id = p_account_id AND rts.env_id = p_env_id
  AND rts.name IN ('executor.step', 'executor.step.discovery', 'executor.step.pause_started', 'executor.step.planned');

-- inngest.insights_steps(p_account_id, p_env_id) collapses
-- insights_step_attempts to the latest attempt per (run_id, step_id) --
-- highest step_attempt, tie-broken by start_time -- mirroring the
-- reference pkg/insights' steps/step_attempts split (docs/plans/010's
-- Logical tables and views table).
CREATE MACRO inngest.insights_steps(p_account_id, p_env_id) AS TABLE
SELECT *
FROM inngest.insights_step_attempts(p_account_id, p_env_id)
QUALIFY ROW_NUMBER() OVER (
  PARTITION BY run_id, step_id ORDER BY COALESCE(step_attempt, 0) DESC, start_time DESC
) = 1;

-- +goose Down
DROP MACRO IF EXISTS inngest.insights_steps;
DROP MACRO IF EXISTS inngest.insights_step_attempts;
DROP MACRO IF EXISTS inngest.insights_extended_trace_spans;
DROP MACRO IF EXISTS inngest.insights_metadata;
DROP MACRO IF EXISTS inngest.insights_events;
DROP MACRO IF EXISTS inngest.insights_runs;
DROP VIEW IF EXISTS inngest.run_metadata_rollup;
