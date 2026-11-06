-- Boolean literals against a boolean column.
SELECT
  run_id
FROM
  runs
WHERE
  is_deferred = TRUE
