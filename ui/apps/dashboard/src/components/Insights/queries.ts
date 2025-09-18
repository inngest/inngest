import { ulid } from 'ulid';

import type { InsightsQuery } from '@/gql/graphql';
import type { QuerySnapshot, QueryTemplate } from './types';

type QueryRecord<T> = Record<string, T>;

export function isQuerySnapshot(q: InsightsQuery | QuerySnapshot): q is QuerySnapshot {
  return 'query' in q;
}

export function isQueryTemplate(
  q: InsightsQuery | QuerySnapshot | QueryTemplate
): q is QueryTemplate {
  return 'templateKind' in q;
}

export function getOrderedQuerySnapshots(
  querySnapshots: QueryRecord<QuerySnapshot>
): QuerySnapshot[] {
  return Object.values(querySnapshots).sort((a, b) => b.createdAt - a.createdAt);
}

export function getOrderedSavedQueries(
  queries: InsightsQuery[] | undefined
): undefined | InsightsQuery[] {
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
