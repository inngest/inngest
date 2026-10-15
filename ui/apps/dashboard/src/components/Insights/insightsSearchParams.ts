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

export type InsightsDeepLinkError = 'sql-too-large';

export type InsightsSearchParams = {
  query_id?: string;
  sql?: string;
  name?: string;
  deep_link_error?: InsightsDeepLinkError;
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
  if (!sql?.trim()) return {};
  if (!isWithinByteLimit(sql, MAX_INSIGHTS_SQL_BYTES)) {
    return { deep_link_error: 'sql-too-large' };
  }

  const name = typeof search.name === 'string' ? search.name.trim() : undefined;
  const validName =
    name && isWithinByteLimit(name, MAX_INSIGHTS_NAME_BYTES) ? name : undefined;

  return validName ? { sql, name: validName } : { sql };
}

export function insightsDeepLinkIntent(
  search: InsightsSearchParams,
): InsightsDeepLinkIntent | undefined {
  if (search.query_id) {
    return { kind: 'saved-query', id: search.query_id };
  }
  if (search.sql) {
    return { kind: 'sql-prefill', sql: search.sql, name: search.name };
  }
}

export function insightsDeepLinkError(
  search: InsightsSearchParams,
): InsightsDeepLinkError | undefined {
  return search.deep_link_error;
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

export function consumeSQLPrefillURL(href: string): string {
  return updateInsightsURL(href, (params) => {
    params.delete('intent');
    params.delete('sql');
    params.delete('name');
    params.delete('deep_link_error');
  });
}

export function syncSavedQueryURL(
  href: string,
  queryID: string | undefined,
): string | undefined {
  const search = queryID ? validateInsightsSearch({ query_id: queryID }) : {};
  if (queryID && !search.query_id) return undefined;

  return updateInsightsURL(href, (params) => {
    params.delete('intent');
    params.delete('query_id');
    params.delete('sql');
    params.delete('name');
    params.delete('deep_link_error');
    if (search.query_id) params.set('query_id', search.query_id);
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
