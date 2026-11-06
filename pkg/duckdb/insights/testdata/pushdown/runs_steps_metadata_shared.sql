-- runs and steps both join the metadata rollup, so neither's run_id filter goes to it.
SELECT
  r.run_id,
  s.step_id
FROM
  runs r
  JOIN steps s ON s.run_id = r.run_id
WHERE
  r.run_id = '01J0000000000000000000000A'
  AND s.run_id = '01J0000000000000000000000A'
