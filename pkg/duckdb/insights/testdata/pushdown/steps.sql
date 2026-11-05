-- steps isn't federated: its predicates are extracted, but nothing is pushed.
SELECT
  run_id
FROM
  steps
WHERE
  run_id = '01J0000000000000000000000A'
