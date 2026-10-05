-- Datetime columns aren't extracted: the query's time bound is extracted separately.
SELECT
  run_id
FROM
  runs
WHERE
  queued_at > TIMESTAMP '2026-01-01'
  AND ended_at <= now()
