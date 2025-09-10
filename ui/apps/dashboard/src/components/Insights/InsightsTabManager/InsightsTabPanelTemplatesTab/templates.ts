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
    event_ts > {{ start_time }}${event ? `\n    AND event_name = '${event}'` : ''}
GROUP BY
    hour_bucket,
    event_name
ORDER BY
    hour_bucket,
    event_name DESC`;
}

const EVENT_TYPE_VOLUME_PER_HOUR_QUERY = makeEventVolumePerHourQuery();
const SPECIFIC_EVENT_PER_HOUR_QUERY = makeEventVolumePerHourQuery('{{ event_name }}');

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
];
