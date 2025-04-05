CREATE TABLE spans (
  span_id TEXT PRIMARY KEY,
  trace_id TEXT NOT NULL,
  parent_span_id TEXT,
  name TEXT NOT NULL,
  start_time TIMESTAMPTZ NOT NULL,
  end_time TIMESTAMPTZ,
  run_id TEXT,
  start_attributes JSONB,
  end_attributes JSONB
);

CREATE INDEX idx_spans_run_id ON spans(run_id);
