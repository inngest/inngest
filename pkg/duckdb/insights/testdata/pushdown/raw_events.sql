-- Raw-mode events.
-- mode: raw
SELECT
  event_id
FROM
  events
WHERE
  event_name = 'app/evt'
  AND event_data ->> 'k' = 'v'
