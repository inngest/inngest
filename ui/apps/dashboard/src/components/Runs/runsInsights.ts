import { RunsOrderByField } from '@/gql/graphql';
import {
  MAX_INSIGHTS_SQL_BYTES,
  sqlPrefillInsightsURL,
} from '@/components/Insights/insightsSearchParams';

const TIME_COLUMNS = {
  [RunsOrderByField.QueuedAt]: 'queued_at',
  [RunsOrderByField.StartedAt]: 'started_at',
  [RunsOrderByField.EndedAt]: 'ended_at',
} as const;

type RunsInsightsParams = {
  envSlug: string;
  celQuery: string;
  startTime: string;
  endTime: string | null;
  timeField: RunsOrderByField;
  appIDs: string[] | null | undefined;
  functionSlug: string | null;
};

export function runsInsightsHref({
  envSlug,
  celQuery,
  startTime,
  endTime,
  timeField,
  appIDs,
  functionSlug,
}: RunsInsightsParams): string | undefined {
  // Insights' CEL macro requires a backtick-delimited argument and has no
  // escaping syntax for a literal backtick.
  if (celQuery.includes('`')) return undefined;

  const timeColumn = TIME_COLUMNS[timeField];
  const filters = [
    `cel(\`${celQuery}\`)`,
    `${timeColumn} >= ${sqlString(insightsDateTime(startTime))}`,
  ];

  if (endTime) {
    filters.push(`${timeColumn} < ${sqlString(insightsDateTime(endTime))}`);
  }
  if (functionSlug) {
    filters.push(`function_id = ${sqlString(functionSlug)}`);
  } else if (appIDs?.length) {
    filters.push(`app_id IN (${appIDs.map(sqlString).join(', ')})`);
  }

  const sql = `SELECT
    *
FROM
    runs
WHERE
    ${filters.join('\n    AND ')}
ORDER BY
    ${timeColumn} DESC
LIMIT 1000`;

  if (new TextEncoder().encode(sql).byteLength > MAX_INSIGHTS_SQL_BYTES) {
    return undefined;
  }

  return sqlPrefillInsightsURL(envSlug, sql, 'Runs search');
}

function insightsDateTime(value: string): string {
  return new Date(value).toISOString().replace('T', ' ').replace('Z', '');
}

function sqlString(value: string): string {
  return `'${value.replaceAll('\\', '\\\\').replaceAll("'", "\\'")}'`;
}
