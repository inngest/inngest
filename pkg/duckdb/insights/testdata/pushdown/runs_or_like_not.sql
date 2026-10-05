-- An OR, a NOT, a LIKE and a NOT IN each become one predicate tree.
SELECT
  run_id
FROM
  runs
WHERE
  (status = 'Failed' OR app_id LIKE 'billing-%')
  AND NOT (status = 'Running')
  AND app_id NOT IN ('a', 'b')
  AND attributes ->> 'user.tier' LIKE 'g%'
