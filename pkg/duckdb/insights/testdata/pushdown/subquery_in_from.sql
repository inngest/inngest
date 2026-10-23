-- A filter inside a FROM subquery.
SELECT
  run_id
FROM
  (
    SELECT
      *
    FROM
      runs
    WHERE
      status = 'Failed'
  ) AS s
