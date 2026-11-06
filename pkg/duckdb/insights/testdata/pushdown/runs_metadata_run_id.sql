-- runs' run_id filter also goes to the metadata rollup's source, which nothing else reads.
SELECT
  run_id,
  metadata
FROM
  runs
WHERE
  run_id = '01J0000000000000000000000A'
  AND status = 'Completed'
