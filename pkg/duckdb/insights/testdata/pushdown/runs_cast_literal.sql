-- A cast on the literal isn't extracted either.
SELECT
  run_id
FROM
  runs
WHERE
  function_id = cast('my-fn' AS VARCHAR)
