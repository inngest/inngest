-- metadata isn't federated: its predicates are extracted, but nothing is pushed.
SELECT
  run_id
FROM
  metadata
WHERE
  scope = 'step'
  AND step_index BETWEEN 1 AND 3
  AND step_attempt > -1
