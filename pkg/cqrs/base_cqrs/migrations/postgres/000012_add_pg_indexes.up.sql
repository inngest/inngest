-- Function Runs Table
CREATE INDEX IF NOT EXISTS idx_function_runs_run_id ON function_runs(run_id);
CREATE INDEX IF NOT EXISTS idx_function_runs_event_id ON function_runs(event_id);
CREATE INDEX IF NOT EXISTS idx_function_runs_timebound ON function_runs(run_started_at DESC, function_id);

-- Function Finishes Table
CREATE INDEX IF NOT EXISTS idx_function_finishes_run_id ON function_finishes(run_id);

-- Events Table
CREATE INDEX IF NOT EXISTS idx_events_internal_id ON events(internal_id);
CREATE INDEX IF NOT EXISTS idx_events_internal_id_received_range ON events(internal_id, received_at DESC);
CREATE INDEX IF NOT EXISTS idx_events_received_name ON events(received_at, event_name);

-- History Table
CREATE INDEX IF NOT EXISTS idx_history_id ON history(id);
CREATE INDEX IF NOT EXISTS idx_history_run_id_created ON history(run_id, created_at ASC);

-- Traces Table
CREATE INDEX IF NOT EXISTS idx_traces_trace_id ON traces(trace_id);