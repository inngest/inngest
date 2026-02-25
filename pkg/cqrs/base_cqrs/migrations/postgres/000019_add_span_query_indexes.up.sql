-- Indexes to support the optimized GetSpanRuns query.
--
-- Note: CREATE INDEX (without CONCURRENTLY) acquires a write lock. On large
-- tables, operators may prefer to run these manually with CONCURRENTLY before
-- applying the migration. IF NOT EXISTS ensures idempotency either way.
--
-- For root span lookup with ORDER BY start_time DESC, run_id.
-- Partial index covers the common filters, enabling index scan with LIMIT pushdown.
-- Note: the filter matches newSpanRunsQueryBuilder which excludes 'Skipped' (not 'Cancelled').
CREATE INDEX IF NOT EXISTS idx_spans_executor_run_start
    ON spans(start_time DESC, run_id)
    WHERE name = 'executor.run' AND debug_run_id IS NULL AND (status IS NULL OR status <> 'Skipped');

-- For correlated subqueries: MAX(end_time) and latest status per (run_id, dynamic_span_id).
-- The INCLUDE(status) enables index-only scans (0 heap fetches) for both lookups.
CREATE INDEX IF NOT EXISTS idx_spans_run_dynamic_endtime_status
    ON spans(run_id, dynamic_span_id, end_time DESC) INCLUDE (status);
