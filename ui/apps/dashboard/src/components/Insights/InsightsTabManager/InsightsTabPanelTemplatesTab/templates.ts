import type { QueryTemplate } from '@/components/Insights/types';

export const TEMPLATES: QueryTemplate[] = [
  // Events templates
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
    ALL
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
    ALL
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
    ALL
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
    ALL
ORDER BY
    cancelled_count DESC`,
    explanation: 'View cancelled function runs within the past 24 hours.',
    templateKind: 'warning',
  },
  // Runs templates
  {
    id: 'runs-count-by-status',
    name: 'Count runs by status',
    query: `SELECT
    function_id,
    status,
    COUNT(*) AS count
FROM
    runs
WHERE
    queued_at > now() - INTERVAL 1 DAY
GROUP BY
    ALL
ORDER BY
    count DESC`,
    explanation: 'Count runs by status in the last 24 hours.',
    templateKind: 'time',
  },
  {
    id: 'errors-by-message',
    name: 'Errors by message',
    query: `SELECT
    error.message AS error_message,
    COUNT(*) AS error_count
FROM
    runs
WHERE
    status = 'Failed'
    AND queued_at > now() - INTERVAL 7 DAY
GROUP BY
    ALL
ORDER BY
    error_count DESC
LIMIT 20`,
    explanation: 'Analyze errors by message to find common failure patterns.',
    templateKind: 'error',
  },
  {
    id: 'failed-runs-for-function',
    name: 'Failed runs for function',
    query: `SELECT
    toStartOfHour(queued_at) AS hour,
    left(error.message, 50) AS error_message_prefix,
    COUNT(*) AS count
FROM
    runs
WHERE
    app_id = '{{ app_id }}'
    AND function_id = '{{ function_id }}'
    AND status = 'Failed'
    AND queued_at > now() - INTERVAL 3 DAY
GROUP BY
    ALL
ORDER BY
    hour DESC,
    count DESC`,
    explanation:
      'Group failed runs by hour and error message for a specific function.',
    templateKind: 'error',
  },
  {
    id: 'duration-percentiles-by-hour',
    name: 'Duration percentiles by hour',
    query: `WITH
    ended_at - started_at AS duration
SELECT
    function_id,
    formatDateTime(queued_at, '%Y-%m-%d %H') AS bucket,
    quantile(0.5)(duration) AS p50_duration,
    quantile(0.9)(duration) AS p90_duration,
    quantile(0.99)(duration) AS p99_duration
FROM
    runs
WHERE
    queued_at > now() - INTERVAL 7 DAY
GROUP BY
    ALL
ORDER BY
    bucket DESC,
    function_id`,
    explanation: 'Analyze function duration percentiles by hour over 7 days.',
    templateKind: 'time',
  },
  {
    id: 'function-start-latency',
    name: 'Function start latency',
    query: `WITH
    started_at - fromUnixTimestamp64Milli(toUInt64(input.ts)) AS function_start_latency
SELECT
    function_id,
    formatDateTime(queued_at, '%Y-%m-%d') AS bucket,
    quantile(0.5)(function_start_latency) AS p50_latency,
    quantile(0.9)(function_start_latency) AS p90_latency,
    quantile(0.99)(function_start_latency) AS p99_latency
FROM
    runs
WHERE
    queued_at > now() - INTERVAL 5 MINUTE
GROUP BY
    ALL
ORDER BY
    bucket DESC,
    function_id`,
    explanation: 'Analyze function start latency percentiles.',
    templateKind: 'time',
  },
];
