-- A key of a string attribute is pushed.
SELECT
  run_id
FROM
  runs
WHERE
  attributes ->> '_inngest.event.trigger.name' = 'app/user.created'
