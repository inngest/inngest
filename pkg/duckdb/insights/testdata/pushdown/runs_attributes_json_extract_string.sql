-- json_extract_string is the same key extraction as ->>.
SELECT
  run_id
FROM
  runs
WHERE
  json_extract_string(attributes, '_inngest.event.trigger.name') IN ('a', 'b')
