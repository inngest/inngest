import type { Query, QuerySnapshot } from './types';

type QueryRecord<T> = Record<string, T>;

export function isQuerySnapshot(q: Query | QuerySnapshot): q is QuerySnapshot {
  return !('saved' in q);
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
