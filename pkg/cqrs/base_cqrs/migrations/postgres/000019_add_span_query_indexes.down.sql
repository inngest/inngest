-- Drop indexes added for the optimized GetSpanRuns query.
DROP INDEX IF EXISTS idx_spans_executor_run_start;
DROP INDEX IF EXISTS idx_spans_run_dynamic_endtime_status;
