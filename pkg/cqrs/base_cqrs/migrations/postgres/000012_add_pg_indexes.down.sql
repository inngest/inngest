-- Drop indexes from Function Runs Table
DROP INDEX IF EXISTS idx_function_runs_run_id;
DROP INDEX IF EXISTS idx_function_runs_event_id;
DROP INDEX IF EXISTS idx_function_runs_timebound;

-- Drop index from Function Finishes Table
DROP INDEX IF EXISTS idx_function_finishes_run_id;

-- Drop indexes from Events Table
DROP INDEX IF EXISTS idx_events_internal_id;
DROP INDEX IF EXISTS idx_events_internal_id_received_range;
DROP INDEX IF EXISTS idx_events_received_name;

-- Drop indexes from History Table
DROP INDEX IF EXISTS idx_history_id;
DROP INDEX IF EXISTS idx_history_run_id_created;

-- Drop index from Traces Table
DROP INDEX IF EXISTS idx_traces_trace_id;