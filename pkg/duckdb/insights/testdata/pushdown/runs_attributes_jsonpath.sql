-- A $-JSONPath isn't a plain key: not extracted.
SELECT
  run_id
FROM
  runs
WHERE
  attributes ->> '$."_inngest.event.trigger.name"' = 'a'
