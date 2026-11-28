-- Status is mutable, but a run that reached a terminal status never leaves it, so
-- terminal values are pushed.
SELECT
  run_id
FROM
  runs
WHERE
  status = 'Completed'
