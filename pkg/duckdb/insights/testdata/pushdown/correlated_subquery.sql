-- A correlated reference to the outer query isn't a literal comparison.
SELECT
  run_id
FROM
  runs
WHERE
  exists (
    SELECT
      1
    FROM
      events
    WHERE
      events.id = runs.run_id
      AND events.name = 'x'
  )
