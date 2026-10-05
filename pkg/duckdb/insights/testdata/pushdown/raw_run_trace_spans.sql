-- Raw mode reads run_trace_spans itself.
-- mode: raw
SELECT
  span_id
FROM
  run_trace_spans
WHERE
  env_id = '00000000-0000-4000-b000-000000000000'
  AND name = 'executor.step'
  AND output ->> 'data' = 'x'
  AND input ->> 'k' = 'v'
