-- Each side of a join gets its own table's predicates.
SELECT
  r.run_id
FROM
  runs r
  JOIN events e ON e.id = r.run_id
WHERE
  r.function_id = 'my-fn'
  AND e.name = 'app/evt'
