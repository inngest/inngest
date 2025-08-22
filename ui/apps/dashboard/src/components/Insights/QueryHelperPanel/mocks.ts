import type { Query, QuerySnapshot } from '@/components/Insights/types';

type QueryRecord<T> = Record<string, T>;

export const MOCK_QUERY_SNAPSHOTS: QueryRecord<QuerySnapshot> = [1, 2, 3].reduce((acc, i) => {
  const id = `query-snapshot-${i}`;
  acc[id] = {
    id,
    createdAt: Date.now(),
    name: `Query Snapshot ${i}`,
    query: `Query snapshot ${i} query text`,
  };
  return acc;
}, {} as QueryRecord<QuerySnapshot>);

export const MOCK_SAVED_QUERIES: QueryRecord<Query> = [1, 2, 3].reduce((acc, i) => {
  const id = `saved-query-${i}`;
  acc[id] = {
    id,
    name: `Saved Query ${i}`,
    query: `Saved query ${i} query text`,
    saved: true,
  };
  return acc;
}, {} as QueryRecord<Query>);
