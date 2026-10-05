-- IN over terminal statuses only is pushed.
SELECT
  run_id
FROM
  runs
WHERE
  status IN ('Completed', 'Failed', 'Cancelled')
