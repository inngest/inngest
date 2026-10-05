-- OR pushes nothing, but a sibling AND conjunct still does.
SELECT
  run_id
FROM
  runs
WHERE
  (
    function_id = 'a'
    OR function_id = 'b'
  )
  AND app_id = 'x'
