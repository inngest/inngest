CREATE TABLE spans (
  -- otel
  span_id TEXT NOT NULL,
  trace_id TEXT NOT NULL,
  parent_span_id TEXT,
  name TEXT NOT NULL,
  start_time DATETIME NOT NULL,
  end_time DATETIME NOT NULL,
  attributes JSON,
  links JSON,

  -- custom
  run_id TEXT NOT NULL,
  dynamic_span_id TEXT,

  PRIMARY KEY (trace_id, span_id)
);

CREATE INDEX idx_spans_run_id ON spans(run_id);
