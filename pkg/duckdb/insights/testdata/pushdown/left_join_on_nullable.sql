-- A LEFT JOIN's ON condition on the nullable side only limits which rows match.
SELECT
  r.run_id
FROM
  runs r
  LEFT JOIN events e ON e.id = r.run_id
  AND e.name = 'app/evt'
