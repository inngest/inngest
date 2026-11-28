-- Events columns map to their physical names.
SELECT
  id
FROM
  events
WHERE
  name = 'app/user.created'
  AND id IN ('e1', 'e2')
  AND v = '2024-01-01'
