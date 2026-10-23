-- Raw mode applies the same push policies.
-- mode: raw
SELECT
  run_id
FROM
  runs
WHERE
  status = 'Running'
  AND attributes ->> '_inngest.dynamic.status' = 'Running'
