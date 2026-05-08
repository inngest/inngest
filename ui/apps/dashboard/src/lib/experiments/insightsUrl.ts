/**
 * URL to open the Insights explorer for a given SQL query. The SQL itself is
 * built server-side (see `experimentInsightsQuery` on the GraphQL API) so
 * that column and table names stay in sync with the insights schema.
 */
export function insightsUrl(envSlug: string, sql: string): string {
  const params = new URLSearchParams({ sql });
  return `/env/${encodeURIComponent(envSlug)}/insights?${params.toString()}`;
}
