-- A mutable attribute key is pushed: the runs delta filters each run's latest row.
SELECT
  run_id
FROM
  runs
WHERE
  attributes ->> '_inngest.dynamic.status' = 'Completed'
