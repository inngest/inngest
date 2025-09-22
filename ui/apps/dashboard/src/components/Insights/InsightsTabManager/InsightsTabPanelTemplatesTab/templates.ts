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

const COUNT_ALIAS_MAP: Record<'failed' | 'cancelled' | 'finished', string> = {
  failed: 'failed_count',
  cancelled: 'cancelled_count',
  finished: 'success_count',
};

function makeFunctionStatusQuery(outcome: 'failed' | 'cancelled' | 'finished') {
  const base = `SELECT
    simpleJSONExtractString(event_data, 'function_id') as function_id,
    COUNT(*) as ${COUNT_ALIAS_MAP[outcome]}
FROM
    events
WHERE
    event_name = 'inngest/function.${outcome}'`;

  const successFilter =
    outcome === 'finished'
      ? `
    AND JSONExtractBool(event_data, 'result', 'success') = true`
      : '';

  return `${base}${successFilter}
    AND event_ts > toUnixTimestamp(addDays(now(), -1)) * 1000
GROUP BY
    function_id
ORDER BY
    ${COUNT_ALIAS_MAP[outcome]} DESC`;
}

const RECENT_FAILED_FUNCTION_COUNT = makeFunctionStatusQuery('failed');
const RECENT_CANCELLED_FUNCTION_COUNT = makeFunctionStatusQuery('cancelled');
const RECENT_SUCCESSFUL_FUNCTION_COUNT = makeFunctionStatusQuery('finished');

export const TEMPLATES: QueryTemplate[] = [
  {
    id: 'recent-events',
    name: 'Recent events',
    query: 'SELECT * FROM events',
    explanation: 'View recents events subject to row and plan history limit restrictions.',
    templateKind: 'time',
  },
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
    query: RECENT_FAILED_FUNCTION_COUNT,
    explanation: 'View failed function runs within the past 24 hours.',
    templateKind: 'error',
  },
  {
    id: 'recent-function-cancellations',
    name: 'Recent function cancellations',
    query: RECENT_CANCELLED_FUNCTION_COUNT,
    explanation: 'View cancelled function runs within the past 24 hours.',
    templateKind: 'warning',
  },
  {
    id: 'recent-function-successes',
    name: 'Recent function successes',
    query: RECENT_SUCCESSFUL_FUNCTION_COUNT,
    explanation: 'View successful function runs within the past 24 hours.',
    templateKind: 'success',
  },
];
