export const MAX_INSIGHTS_QUERY_ID_BYTES = 128;
export const MAX_INSIGHTS_SQL_BYTES = 8 * 1024;
export const MAX_INSIGHTS_NAME_BYTES = 128;

export type InsightsSearchParams = {
  query_id?: string;
  sql?: string;
  name?: string;
};

const encoder = new TextEncoder();

function isWithinByteLimit(value: string, limit: number): boolean {
  return encoder.encode(value).byteLength <= limit;
}

export function validateInsightsSearch(
  search: Record<string, unknown>,
): InsightsSearchParams {
  const queryID =
    typeof search.query_id === 'string' ? search.query_id.trim() : undefined;

  // A saved-query link and a SQL-prefill link are mutually exclusive. Treat
  // any non-empty query_id as the selected mode, even when it is invalid, so
  // malformed query_id links cannot fall through to a co-supplied SQL value.
  if (queryID) {
    return isWithinByteLimit(queryID, MAX_INSIGHTS_QUERY_ID_BYTES)
      ? { query_id: queryID }
      : {};
  }

  const sql = typeof search.sql === 'string' ? search.sql : undefined;
  if (!sql?.trim() || !isWithinByteLimit(sql, MAX_INSIGHTS_SQL_BYTES)) {
    return {};
  }

  const name = typeof search.name === 'string' ? search.name.trim() : undefined;
  if (name && isWithinByteLimit(name, MAX_INSIGHTS_NAME_BYTES)) {
    return { sql, name };
  }

  return { sql };
}

export function insightsURL(envSlug: string): string {
  return `/env/${encodeURIComponent(envSlug)}/insights`;
}

export function sqlPrefillInsightsURL(
  envSlug: string,
  sql: string,
  name?: string,
): string | undefined {
  const search = validateInsightsSearch({ sql, name });
  if (!search.sql) return undefined;

  const params = new URLSearchParams({ sql: search.sql });
  if (search.name) params.set('name', search.name);
  return `${insightsURL(envSlug)}?${params.toString()}`;
}
