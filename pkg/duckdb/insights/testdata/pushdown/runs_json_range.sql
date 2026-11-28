-- Only =, != and IN are extracted on JSON keys.
SELECT
  run_id
FROM
  runs
WHERE
  output ->> 'result' > 'a'
