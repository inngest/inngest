-- A literal must have the column's own type: no implicit casts.
SELECT
  run_id
FROM
  runs
WHERE
  is_deferred = 'true'
  AND function_id = 5
