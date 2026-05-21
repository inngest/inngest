-- +goose NO TRANSACTION
-- +goose Up

-- GetSpanOutput queries spans by span_id alone, but the PK is (trace_id,
-- span_id) which cannot serve a span_id-only lookup efficiently.
-- 35 calls, 10 s total, 286 ms mean in dev pg_stat_statements.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_spans_span_id
  ON spans (span_id);

-- IMMUTABLE wrapper around convert_from for use in queries (and in a GIN
-- index once pg_trgm is available). The function itself does not require
-- any extension.
CREATE OR REPLACE FUNCTION trigger_ids_as_text(val bytea) RETURNS text
  LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE
AS $$ SELECT convert_from(val, 'UTF8') $$;

-- NOTE: The GIN trigram index for GetTraceRunsByTriggerId is NOT created
-- here because pg_trgm is not allow-listed on Azure Database for
-- PostgreSQL. A follow-up migration will add the index once pg_trgm is
-- enabled in the Azure infrastructure configuration.

-- +goose Down

DROP INDEX CONCURRENTLY IF EXISTS idx_trace_runs_trigger_ids_trgm;
DROP INDEX CONCURRENTLY IF EXISTS idx_spans_span_id;
DROP FUNCTION IF EXISTS trigger_ids_as_text(bytea);
