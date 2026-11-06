-- A cast on the column isn't extracted, even to the column's own type.
SELECT
  run_id
FROM
  runs
WHERE
  function_id::VARCHAR = 'my-fn'
  AND cast(app_id AS VARCHAR) = 'app'
