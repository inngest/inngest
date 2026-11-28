-- IN on columns; each AND conjunct is its own predicate.
SELECT
  run_id
FROM
  runs
WHERE
  app_id IN ('app-a', 'app-b')
  AND run_id = '01J0000000000000000000000A'
