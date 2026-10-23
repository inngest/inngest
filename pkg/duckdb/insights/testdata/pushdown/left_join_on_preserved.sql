-- A LEFT JOIN's ON condition on the preserved side doesn't filter its rows, so it
-- mustn't be pushed.
SELECT
  r.run_id
FROM
  runs r
  LEFT JOIN events e ON e.id = r.run_id
  AND r.function_id = 'my-fn'
