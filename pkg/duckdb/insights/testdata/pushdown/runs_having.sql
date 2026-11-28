-- Only WHERE conjuncts are extracted, not HAVING.
SELECT
  function_id,
  count(*)
FROM
  runs
GROUP BY
  function_id
HAVING
  function_id = 'my-fn'
