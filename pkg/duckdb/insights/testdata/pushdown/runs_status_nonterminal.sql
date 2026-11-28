-- A non-terminal status is pushed: the runs delta filters each run's latest row, after its collapse.
SELECT
  run_id
FROM
  runs
WHERE
  status = 'Running'
