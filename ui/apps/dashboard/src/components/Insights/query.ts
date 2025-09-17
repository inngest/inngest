import type { InsightsQuery as GQLInsightsQuery } from '@/gql/graphql';
import type { Query as LocalQuery } from './types';

export function toLocalQuery(q: GQLInsightsQuery): LocalQuery {
  return {
    id: q.id,
    name: q.name,
    query: q.sql,
    saved: true,
  };
}

export function toLocalQueryArray(
  queries: GQLInsightsQuery[] | undefined
): LocalQuery[] | undefined {
  if (!queries) return undefined;
  return queries.map(toLocalQuery);
}
