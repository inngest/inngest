import type { QueryTemplate } from '@/components/Insights/types';

export const TEMPLATES: QueryTemplate[] = [
  {
    id: 'recent-events',
    name: 'Recent events',
    query: 'SELECT * FROM events',
    explanation:
      'View recents events subject to row and plan history limit restrictions.',
    templateKind: 'time',
  },
  {
    id: 'event-type-volume-per-hour',
    name: 'Events by type per hour',
    query: `SELECT
    toStartOfHour(fromUnixTimestamp64Milli(ts)) AS hour_bucket,
    name,
    COUNT(*) AS event_count
FROM
    events
WHERE
    ts > toUnixTimestamp64Milli(subtractDays(now64(), 3))
GROUP BY
    hour_bucket,
    name
ORDER BY
    hour_bucket,
    name DESC`,
    explanation: 'Examine hourly volume by event type.',
    templateKind: 'time',
  },
  {
    id: 'specific-event-per-hour',
    name: 'Specific event per hour',
    query: `SELECT
    toStartOfHour(fromUnixTimestamp64Milli(ts)) AS hour_bucket,
    name,
    COUNT(*) AS event_count
FROM
    events
WHERE
    ts > toUnixTimestamp64Milli(subtractDays(now64(), 3))
    AND name = '{{ event_name }}'
GROUP BY
    hour_bucket,
    name
ORDER BY
    hour_bucket,
    name DESC`,
    explanation: 'View hourly volume of a specific event.',
    templateKind: 'time',
  },
  {
    id: 'recent-function-failures',
    name: 'Recent function failures',
    query: `SELECT
    data.function_id AS function_id,
    COUNT(*) as failed_count
FROM
    events
WHERE
    name = 'inngest/function.failed'
    AND ts > toUnixTimestamp64Milli(subtractDays(now64(), 1))
GROUP BY
    function_id
ORDER BY
    failed_count DESC`,
    explanation: 'View failed function runs within the past 24 hours.',
    templateKind: 'error',
  },
  {
    id: 'recent-function-cancellations',
    name: 'Recent function cancellations',
    query: `SELECT
    data.function_id AS function_id,
    COUNT(*) as cancelled_count
FROM
    events
WHERE
    name = 'inngest/function.cancelled'
    AND ts > toUnixTimestamp64Milli(subtractDays(now64(), 1))
GROUP BY
    function_id
ORDER BY
    cancelled_count DESC`,
    explanation: 'View cancelled function runs within the past 24 hours.',
    templateKind: 'warning',
  },
  /*
  {
    id: 'recent-function-successes',
    name: 'Recent function successes',
    query: `SELECT
    data.function_id AS function_id,
    COUNT(*) as success_count
FROM
    events
WHERE
    name = 'inngest/function.finished'
    AND JSONExtractBool(data, 'result', 'success') = true
    AND ts > toUnixTimestamp64Milli(subtractDays(now64(), 1))
GROUP BY
    function_id
ORDER BY
    success_count DESC`,
    explanation: 'View successful function runs within the past 24 hours.',
    templateKind: 'success',
  },
  */
];
