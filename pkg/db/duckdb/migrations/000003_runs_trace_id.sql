-- +goose Up

-- trace_id is the run's OTel trace ID, copied from the span row
-- pkg/duckdb/tracing's materializeRuns builds each runs row from, so run
-- listings can report it without joining run_trace_spans. Appended (not in
-- the baseline) so already-persisted catalogs pick it up; rows written
-- before this migration have NULL.
ALTER TABLE inngest.runs ADD COLUMN trace_id VARCHAR;

-- +goose Down

ALTER TABLE inngest.runs DROP COLUMN trace_id;
