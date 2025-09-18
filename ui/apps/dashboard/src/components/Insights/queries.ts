import { ulid } from 'ulid';

import type { InsightsQuery as GQLInsightsQuery } from '@/gql/graphql';
import type { Query as LocalQuery, Query, QuerySnapshot, QueryTemplate } from './types';

type QueryRecord<T> = Record<string, T>;

export function isQuerySnapshot(q: Query | QuerySnapshot): q is QuerySnapshot {
  return !('savedQueryId' in q);
}

export function isQueryTemplate(q: Query | QuerySnapshot | QueryTemplate): q is QueryTemplate {
  return 'templateKind' in q;
}

export function getOrderedQuerySnapshots(
  querySnapshots: QueryRecord<QuerySnapshot>
): QuerySnapshot[] {
  return Object.values(querySnapshots).sort((a, b) => b.createdAt - a.createdAt);
}

export function getOrderedSavedQueries(queries: Query[] | undefined): undefined | Query[] {
  if (queries === undefined) return undefined;
  return [...queries].sort((a, b) => a.name.localeCompare(b.name));
}

export function makeQuerySnapshot(query: string, name?: string): QuerySnapshot {
  return {
    createdAt: Date.now(),
    id: ulid(),
    name: name ?? query,
    query,
  };
}

export function toLocalQuery(q: GQLInsightsQuery): LocalQuery {
  return {
    id: q.id,
    name: q.name,
    query: q.sql,
    savedQueryId: q.id,
  };
}

export function toLocalQueryArray(
  queries: GQLInsightsQuery[] | undefined
): LocalQuery[] | undefined {
  if (!queries) return undefined;
  return queries.map(toLocalQuery);
}

export function isSavedQuery(q: LocalQuery): boolean {
  return q.savedQueryId !== undefined;
}
