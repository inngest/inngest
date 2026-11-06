-- An IN-subquery's own WHERE is extracted for its table.
SELECT
  run_id
FROM
  runs
WHERE
  run_id IN (
    SELECT
      id
    FROM
      events
    WHERE
      name = 'app/evt'
  )
