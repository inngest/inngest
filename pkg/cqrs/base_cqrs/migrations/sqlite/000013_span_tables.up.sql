CREATE TABLE spans (
  span_id TEXT NOT NULL,
  trace_id TEXT NOT NULL,
  parent_span_id TEXT,
  name TEXT NOT NULL,
  start_time DATETIME NOT NULL,
  end_time DATETIME,
  run_id TEXT,
  start_attributes JSON,
  end_attributes JSON,
  PRIMARY KEY (trace_id, span_id)
);

CREATE INDEX idx_spans_run_id ON spans(run_id);
