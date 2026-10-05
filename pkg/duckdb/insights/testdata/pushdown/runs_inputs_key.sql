-- inputs is forwarded, but the buffer stores it as an array, so its streamer can't apply a key filter on it.
SELECT
  run_id
FROM
  runs
WHERE
  inputs ->> 'name' = 'app/evt'
