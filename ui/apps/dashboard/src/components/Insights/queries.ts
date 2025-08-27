import { ulid } from 'ulid';

import type { Query, QuerySnapshot, QueryTemplate } from './types';

type QueryRecord<T> = Record<string, T>;

export function isQuerySnapshot(q: Query | QuerySnapshot): q is QuerySnapshot {
  return !('saved' in q);
}

export function isQueryTemplate(q: Query | QuerySnapshot | QueryTemplate): q is QueryTemplate {
  return 'templateKind' in q;
}

export function getOrderedQuerySnapshots(
  querySnapshots: QueryRecord<QuerySnapshot>
): QuerySnapshot[] {
  return Object.values(querySnapshots).sort((a, b) => b.createdAt - a.createdAt);
}

export function getOrderedSavedQueries(queries: QueryRecord<Query>): Query[] {
  return Object.values(queries)
    .filter((query) => query.saved)
    .sort((a, b) => a.name.localeCompare(b.name));
}

export function makeQuerySnapshot(query: string, name?: string): QuerySnapshot {
  return {
    createdAt: Date.now(),
    id: ulid(),
    name: name ?? query,
    query,
  };
}
