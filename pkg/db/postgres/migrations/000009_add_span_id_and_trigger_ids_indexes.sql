-- +goose NO TRANSACTION
-- +goose Up

-- GetSpanOutput queries spans by span_id alone, but the PK is (trace_id,
-- span_id) which cannot serve a span_id-only lookup efficiently.
-- 35 calls, 10 s total, 286 ms mean in dev pg_stat_statements.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_spans_span_id
  ON spans (span_id);

-- GetTraceRunsByTriggerId does a full table scan with
-- POSITION(event_id IN convert_from(trigger_ids, 'UTF8')) > 0.
-- 220 calls, 32.3 s total, 147 ms mean in dev pg_stat_statements.
-- Enable pg_trgm and add a GIN index so the rewritten LIKE query can use it.
--
-- convert_from is STABLE (not IMMUTABLE) in Postgres, so we wrap it in a
-- thin IMMUTABLE function. The encoding literal 'UTF8' never changes, making
-- this safe to mark immutable.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE OR REPLACE FUNCTION trigger_ids_as_text(val bytea) RETURNS text
  LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE
AS $$ SELECT convert_from(val, 'UTF8') $$;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_trace_runs_trigger_ids_trgm
  ON trace_runs USING gin (trigger_ids_as_text(trigger_ids) gin_trgm_ops);

-- +goose Down

DROP INDEX CONCURRENTLY IF EXISTS idx_trace_runs_trigger_ids_trgm;
DROP INDEX CONCURRENTLY IF EXISTS idx_spans_span_id;
DROP FUNCTION IF EXISTS trigger_ids_as_text(bytea);
