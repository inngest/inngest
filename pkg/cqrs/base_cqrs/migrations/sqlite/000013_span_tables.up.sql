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
  dynamic_span_id TEXT,
  account_id TEXT NOT NULL,
  app_id TEXT NOT NULL,
  function_id TEXT NOT NULL,
  run_id TEXT NOT NULL,
  env_id TEXT NOT NULL,
  output JSON,

  PRIMARY KEY (trace_id, span_id)
);

CREATE INDEX idx_spans_run_id ON spans(run_id); -- mainly for debugging
CREATE INDEX idx_spans_run_id_dynamic_start_time ON spans(run_id, dynamic_span_id, start_time);
