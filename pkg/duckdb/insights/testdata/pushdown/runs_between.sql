-- BETWEEN becomes a >= and a <= predicate.
SELECT
  run_id
FROM
  runs
WHERE
  function_id BETWEEN 'a' AND 'm'
