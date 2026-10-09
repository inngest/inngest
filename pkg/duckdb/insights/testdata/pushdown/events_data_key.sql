-- Event data is immutable, so any key filter is pushed.
SELECT
  id
FROM
  events
WHERE
  data ->> 'userId' = 'u_123'
  AND meta ->> 'source' != 'api'
