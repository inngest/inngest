import type { QueryTemplate } from '@/components/Insights/types';

// TODO: account_id will be derived by the BE.

function makeEventVolumePerHourQuery(event?: string) {
  return `SELECT
    toStartOfHour(toDateTime(event_ts / 1000)) AS hour_bucket,
    event_name,
    COUNT(*) AS event_count
FROM
    events
WHERE
    event_ts > toUnixTimestamp(subtractDays(now(), 3)) * 1000${
      event ? `\n    AND event_name = '${event}'` : ''
    }
GROUP BY
    hour_bucket,
    event_name
ORDER BY
    hour_bucket,
    event_name DESC`;
}

const EVENT_TYPE_VOLUME_PER_HOUR_QUERY = makeEventVolumePerHourQuery();
const SPECIFIC_EVENT_PER_HOUR_QUERY = makeEventVolumePerHourQuery('{{ event_name }}');

const SPECIFIC_FAILED_FUNCTION_INVOCATIONS_QUERY = `SELECT
    COUNT(*) as failed_invocations
FROM
    events
WHERE
    event_name = 'inngest/function.failed'
    AND simpleJSONExtractString(event_data, 'function_id') = '{{ function_id }}'
    AND event_ts > toUnixTimestamp(addDays(now(), -1)) * 1000`;

export const TEMPLATES: QueryTemplate[] = [
  {
    id: 'event-type-volume-per-hour',
    name: 'Events by type per hour',
    query: EVENT_TYPE_VOLUME_PER_HOUR_QUERY,
    explanation: 'Examine hourly volume by event type.',
    templateKind: 'time',
  },
  {
    id: 'specific-event-per-hour',
    name: 'Specific event per hour',
    query: SPECIFIC_EVENT_PER_HOUR_QUERY,
    explanation: 'View hourly volume of a specific event.',
    templateKind: 'time',
  },
  {
    id: 'recent-function-failures',
    name: 'Recent function failures',
    query: SPECIFIC_FAILED_FUNCTION_INVOCATIONS_QUERY,
    explanation: 'View failed function invocations in the past 24 hours.',
    templateKind: 'error',
  },
];
