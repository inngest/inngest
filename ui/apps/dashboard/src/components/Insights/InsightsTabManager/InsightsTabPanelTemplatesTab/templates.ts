import type { QueryTemplate } from '@/components/Insights/types';

// TODO: account_id and workspace_id will be sent directly without the user specifying them.

function makeEventVolumePerHourQuery(event?: string) {
  return `SELECT
    toStartOfHour(toDateTime(event_ts / 1000)) AS hour_bucket,
    event_name,
    COUNT(*) AS event_count
FROM
    events
WHERE
    account_id = '{{ account_id }}'
    AND workspace_id = '{{ workspace_id }}'
    AND event_ts > {{ start_time }}${event ? `\n    AND event_name = '${event}'` : ''}
GROUP BY
    hour_bucket,
    event_name
ORDER BY
    hour_bucket,
    event_name DESC`;
}

const EVENT_TYPE_VOLUME_PER_HOUR_QUERY = makeEventVolumePerHourQuery();
const SPECIFIC_EVENT_PER_HOUR_QUERY = makeEventVolumePerHourQuery('{{ event_name }}');

const DEPLOYMENT_EVENTS_PER_HOUR_QUERY = `SELECT
    toStartOfHour(toDateTime(event_ts / 1000)) AS hour_bucket,
    countIf(event_name = 'api/deployment.succeeded') AS succeeded_count,
    countIf(event_name = 'api/deployment.failed') AS failed_count,
    countIf(event_name = 'api/deployment.skipped') AS skipped_count,
    COUNT(*) AS total_count
FROM
    events
WHERE
    account_id = '{{ account_id }}'
    AND workspace_id = '{{ workspace_id }}'
    AND event_name IN ('api/deployment.succeeded', 'api/deployment.failed', 'api/deployment.skipped')
    AND event_ts > {{ start_time }}
GROUP BY hour_bucket
ORDER BY hour_bucket DESC`;

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
    id: 'deployment-events-per-hour',
    name: 'Deployment events per hour',
    query: DEPLOYMENT_EVENTS_PER_HOUR_QUERY,
    explanation: 'Track failed, skipped, and successful deployments.',
    templateKind: 'time',
  },
];
