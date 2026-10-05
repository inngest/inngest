-- Each UNION operand reads runs: two reads, nothing pushed.
SELECT
  run_id
FROM
  runs
WHERE
  function_id = 'a'
UNION ALL
SELECT
  run_id
FROM
  runs
WHERE
  function_id = 'b'
