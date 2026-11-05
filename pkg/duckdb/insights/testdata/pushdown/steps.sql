-- steps collapses to the latest attempt per step, so only run-level columns are pushed.
SELECT
  run_id
FROM
  steps
WHERE
  run_id = '01J0000000000000000000000A'
