CREATE TABLE spans (
  -- otel
  span_id TEXT NOT NULL,
  trace_id TEXT NOT NULL,
  parent_span_id TEXT,
  name TEXT NOT NULL,
  start_time TIMESTAMPTZ NOT NULL,
  end_time TIMESTAMPTZ NOT NULL,
  attributes JSONB,
  links JSONB,

  -- custom
  run_id TEXT NOT NULL,
  dynamic_span_id TEXT,

  PRIMARY KEY (trace_id, span_id)
);

CREATE INDEX idx_spans_run_id ON spans(run_id); -- mainly for debugging
CREATE INDEX idx_spans_run_id_dynamic_start_time ON spans(run_id, dynamic_span_id, start_time);
