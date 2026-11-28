-- LIKE is extracted; a function of a column (lower(app_id)) isn't.
SELECT
  run_id
FROM
  runs
WHERE
  function_id like 'my-%'
  AND lower(app_id) = 'a'
