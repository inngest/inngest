-- A literal on the left flips the operator.
SELECT
  run_id
FROM
  runs
WHERE
  'my-fn' <= function_id
