-- +goose NO TRANSACTION
-- +goose Up

-- Support legacy trace_runs list filters ordered by the selected time field.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_trace_runs_acct_ws_queued
  ON trace_runs (account_id, workspace_id, queued_at DESC, run_id);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_trace_runs_acct_ws_started
  ON trace_runs (account_id, workspace_id, started_at DESC, run_id);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_trace_runs_acct_ws_ended
  ON trace_runs (account_id, workspace_id, ended_at DESC, run_id);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_trace_runs_acct_ws_ended;
DROP INDEX CONCURRENTLY IF EXISTS idx_trace_runs_acct_ws_started;
DROP INDEX CONCURRENTLY IF EXISTS idx_trace_runs_acct_ws_queued;
