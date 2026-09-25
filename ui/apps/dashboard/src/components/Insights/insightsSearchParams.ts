export const MAX_INSIGHTS_QUERY_ID_BYTES = 128;
export const MAX_INSIGHTS_SQL_BYTES = 8 * 1024;
export const MAX_INSIGHTS_NAME_BYTES = 128;

export type InsightsDeepLinkIntent =
  | { kind: 'saved-query'; id: string }
  | {
      kind: 'sql-prefill';
      sql: string;
      name: string | undefined;
    };

export type InsightsSearchParams = {
  intent?: InsightsDeepLinkIntent;
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
      ? { intent: { kind: 'saved-query', id: queryID } }
      : {};
  }

  const sql = typeof search.sql === 'string' ? search.sql : undefined;
  if (!sql?.trim() || !isWithinByteLimit(sql, MAX_INSIGHTS_SQL_BYTES)) {
    return {};
  }

  const name = typeof search.name === 'string' ? search.name.trim() : undefined;
  const validName =
    name && isWithinByteLimit(name, MAX_INSIGHTS_NAME_BYTES) ? name : undefined;

  return { intent: { kind: 'sql-prefill', sql, name: validName } };
}

export function insightsURL(envSlug: string): string {
  return `/env/${encodeURIComponent(envSlug)}/insights`;
}

export function savedQueryInsightsURL(
  envSlug: string,
  queryID: string,
): string | undefined {
  const intent = validateInsightsSearch({ query_id: queryID }).intent;
  if (intent?.kind !== 'saved-query') return undefined;

  const params = new URLSearchParams({ query_id: intent.id });
  return `${insightsURL(envSlug)}?${params.toString()}`;
}

export function sqlPrefillInsightsURL(
  envSlug: string,
  sql: string,
  name?: string,
): string | undefined {
  const intent = validateInsightsSearch({ sql, name }).intent;
  if (intent?.kind !== 'sql-prefill') return undefined;

  const params = new URLSearchParams({ sql: intent.sql });
  if (intent.name) params.set('name', intent.name);
  return `${insightsURL(envSlug)}?${params.toString()}`;
}

export function consumeSQLPrefillURL(href: string): string {
  return updateInsightsURL(href, (params) => {
    params.delete('intent');
    params.delete('sql');
    params.delete('name');
  });
}

export function syncSavedQueryURL(
  href: string,
  queryID: string | undefined,
): string | undefined {
  const intent = queryID
    ? validateInsightsSearch({ query_id: queryID }).intent
    : undefined;
  if (queryID && intent?.kind !== 'saved-query') return undefined;

  return updateInsightsURL(href, (params) => {
    params.delete('intent');
    params.delete('query_id');
    params.delete('sql');
    params.delete('name');
    if (intent?.kind === 'saved-query') params.set('query_id', intent.id);
  });
}

function updateInsightsURL(
  href: string,
  update: (params: URLSearchParams) => void,
): string {
  const url = new URL(href, 'https://app.inngest.local');
  update(url.searchParams);
  return `${url.pathname}${url.search}${url.hash}`;
}
