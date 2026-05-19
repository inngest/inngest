-- +goose NO TRANSACTION
-- +goose Up

-- Index 1: history(created_at, type)
--
-- Covers ALL Grafana alerting queries (P95 regression, P95 backstop, step
-- output size) and the cleanup cronjob DELETE on history.
--
-- Why (created_at, type) and not just (created_at)?
--   The Grafana alerts always filter both created_at range AND type IN (...).
--   Including type as the 2nd column allows index-only filtering without
--   heap access for the type predicate. The cleanup only filters on created_at,
--   which still uses this index efficiently (leading column match).
--
-- Not a partial index because different alerts filter different type values:
--   - P95 alerts: ('FunctionCompleted', 'FunctionFailed')
--   - Step output: ('StepCompleted', 'StepErrored', 'StepFailed')
--   - Cleanup: no type filter (all rows by time)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_history_created_at_type
  ON history (created_at, type);

-- Index 2: traces("timestamp")
--
-- Covers the cleanup cronjob DELETE:
--   DELETE FROM traces WHERE ctid IN (
--     SELECT ctid FROM traces WHERE "timestamp" < $cutoff LIMIT N)
--
-- Previously had no timestamp index at all — forced sequential scan.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_traces_timestamp
  ON traces ("timestamp");

-- Index 3: spans(start_time)
--
-- Covers the cleanup cronjob DELETE:
--   DELETE FROM spans WHERE ctid IN (
--     SELECT ctid FROM spans WHERE start_time < $cutoff LIMIT N)
--
-- Note: idx_spans_active_start_time (partial) was created in 000003 and
-- dropped in 000004 because its partial predicate was too narrow (only 9
-- scans in its lifetime). This replacement has no partial predicate and
-- covers the bare start_time range scan the cleanup needs.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_spans_start_time
  ON spans (start_time);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_history_created_at_type;
DROP INDEX CONCURRENTLY IF EXISTS idx_traces_timestamp;
DROP INDEX CONCURRENTLY IF EXISTS idx_spans_start_time;
