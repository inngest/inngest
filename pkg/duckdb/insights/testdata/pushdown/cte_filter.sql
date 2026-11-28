-- A filter inside a CTE that reads the table once is pushed.
WITH
  c AS (
    SELECT
      run_id,
      function_id
    FROM
      runs
    WHERE
      function_id = 'my-fn'
  )
SELECT
  run_id
FROM
  c
