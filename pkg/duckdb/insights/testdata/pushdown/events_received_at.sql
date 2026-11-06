-- received_at is a datetime: not extracted.
SELECT
  id
FROM
  events
WHERE
  received_at > now() - INTERVAL 1 hour
