-- A numeric cast of a JSON key isn't extracted.
SELECT
  run_id
FROM
  runs
WHERE
  (output ->> 'count')::INTEGER > 5
  AND try_cast (
    attributes ->> '_inngest.function.version' AS BIGINT
  ) = 3
