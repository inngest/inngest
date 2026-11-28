-- An IN mixing terminal and non-terminal statuses is pushed whole.
SELECT
  run_id
FROM
  runs
WHERE
  status IN ('Completed', 'Running')
