-- NOT, NOT IN and NOT BETWEEN each become a NOT predicate.
SELECT
  run_id
FROM
  runs
WHERE
  NOT (function_id = 'a')
  AND app_id NOT IN ('b')
  AND run_id NOT BETWEEN 'a' AND 'b'
