-- A table read twice could be filtered for only one of its reads: nothing is pushed
-- for it.
SELECT
  a.run_id
FROM
  runs a
  JOIN runs b ON a.run_id = b.run_id
WHERE
  a.function_id = 'x'
