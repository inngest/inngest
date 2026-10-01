const MAX_INSIGHTS_SQL_BYTES = 8 * 1024;

type RunsInsightsParams = {
  envSlug: string;
  celQuery: string;
  functionSlug: string | null;
};

export function runsInsightsHref({
  envSlug,
  celQuery,
  functionSlug,
}: RunsInsightsParams): string | undefined {
  // Insights' CEL macro requires a backtick-delimited argument and has no
  // escaping syntax for a literal backtick.
  if (celQuery.includes('`')) return undefined;

  const filters = [
    `cel(\`${toInsightsCEL(celQuery)}\`)`,
    'queued_at > now() - INTERVAL 3 DAY',
  ];
  if (functionSlug) {
    filters.push(`function_id = ${sqlString(functionSlug)}`);
  }

  const sql = `SELECT
    *
FROM
    runs
WHERE
    ${filters.join('\n    AND ')}`;

  // Keep generated SQL within Insights' accepted deep-link limit.
  if (new TextEncoder().encode(sql).byteLength > MAX_INSIGHTS_SQL_BYTES) {
    return undefined;
  }

  const params = new URLSearchParams({ sql, name: 'Runs search' });
  return `/env/${encodeURIComponent(envSlug)}/insights?${params.toString()}`;
}

function toInsightsCEL(celQuery: string): string {
  const expressions = celQuery.split('\n');
  if (expressions.length === 1) return celQuery;

  // Runs treats each editor line as a separate expression and ANDs them.
  // Insights parses the macro argument as one expression, so make that
  // implicit conjunction explicit while preserving each line verbatim.
  return expressions.map((expression) => `(${expression})`).join(' &&\n');
}

function sqlString(value: string): string {
  return `'${value.replaceAll('\\', '\\\\').replaceAll("'", "\\'")}'`;
}
