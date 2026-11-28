-- != is pushed too: exact on each run's latest row.
SELECT
  run_id
FROM
  runs
WHERE
  status <> 'Completed'
