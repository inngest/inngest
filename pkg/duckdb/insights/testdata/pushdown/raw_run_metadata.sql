-- mode: raw
-- raw run_metadata is the emissions as stored, so every column is pushed.
SELECT
  kind
FROM
  run_metadata
WHERE
  run_id = '01J0000000000000000000000A'
  AND is_user = true
  AND kind = 'cost'
