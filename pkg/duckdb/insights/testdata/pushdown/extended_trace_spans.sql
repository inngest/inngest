-- extended_trace_spans federates onto run_trace_spans.
SELECT
  span_id
FROM
  extended_trace_spans
WHERE
  name = 'my.span'
  AND function_id = 'my-fn'
  AND attributes ->> '_inngest.app.name' = 'app'
