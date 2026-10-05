-- A filter on the CTE's output isn't a filter on the table.
WITH
  c AS (
    SELECT
      run_id,
      function_id
    FROM
      runs
  )
SELECT
  run_id
FROM
  c
WHERE
  function_id = 'my-fn'
