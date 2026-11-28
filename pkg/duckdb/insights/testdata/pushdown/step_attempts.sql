-- step_attempts doesn't collapse, so its span columns are pushed (the attributes-derived step columns aren't).
SELECT
  span_id
FROM
  step_attempts
WHERE
  run_id = '01J0000000000000000000000A'
  AND name = 'executor.step'
  AND step_attempt > 0
