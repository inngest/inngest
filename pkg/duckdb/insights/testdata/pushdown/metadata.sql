-- metadata rolls emissions up per span, so only its group key (run_id, span_id) and run_queued_at are pushed.
SELECT
  run_id
FROM
  metadata
WHERE
  run_id = '01J0000000000000000000000A'
  AND scope = 'step'
  AND step_index BETWEEN 1 AND 3
  AND step_attempt > -1
