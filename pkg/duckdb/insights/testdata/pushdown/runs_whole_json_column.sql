-- Comparing a whole JSON column isn't a key filter: not pushed.
SELECT
  run_id
FROM
  runs
WHERE
  output = '{}'
