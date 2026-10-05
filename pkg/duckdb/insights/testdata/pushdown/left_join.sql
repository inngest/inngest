-- Extracted predicates are null-rejecting, so they're safe on the nullable side of an
-- outer join too.
SELECT
  r.run_id
FROM
  runs r
  LEFT JOIN events e ON e.id = r.run_id
WHERE
  e.name = 'app/evt'
