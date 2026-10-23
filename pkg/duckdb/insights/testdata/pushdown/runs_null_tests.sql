-- NULL tests aren't null-rejecting filters: not extracted.
SELECT
  run_id
FROM
  runs
WHERE
  ended_at IS NULL
  AND output IS NOT NULL
