import { sqlPrefillInsightsURL } from '@/components/Insights/insightsSearchParams';

/**
 * URL to open the Insights explorer for a given SQL query. The SQL itself is
 * built server-side (see `experimentInsightsQuery` on the GraphQL API) so
 * that column and table names stay in sync with the insights schema.
 */
export function insightsUrl(envSlug: string, sql: string): string | undefined {
  return sqlPrefillInsightsURL(envSlug, sql);
}
