CREATE TABLE spans (
  span_id TEXT PRIMARY KEY,
  trace_id TEXT NOT NULL,
  parent_span_id TEXT,
  name TEXT NOT NULL,
  start_time DATETIME NOT NULL,
  end_time DATETIME NOT NULL,
  run_id TEXT,
  attributes JSON
);

CREATE INDEX idx_spans_run_id ON spans(run_id);
