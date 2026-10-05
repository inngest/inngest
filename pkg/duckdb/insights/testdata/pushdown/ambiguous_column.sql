-- An unqualified column two joined tables could both have is rejected by validation.
SELECT
  r.run_id
FROM
  runs r
  JOIN extended_trace_spans s ON s.run_id = r.run_id
WHERE
  run_id = 'x'
