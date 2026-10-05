-- Output is set once, when the run ends; a null-rejecting filter can only match ended
-- runs, so it's pushed.
SELECT
  run_id
FROM
  runs
WHERE
  output ->> 'result' = 'ok'
