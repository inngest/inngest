import { ulid } from 'ulid';

import type { InsightsQuery } from '@/gql/graphql';
import type { QuerySnapshot, QueryTemplate } from './types';

export function isQuerySnapshot(q: InsightsQuery | QuerySnapshot): q is QuerySnapshot {
  return 'isSnapshot' in q && q.isSnapshot;
}

export function isQueryTemplate(
  q: InsightsQuery | QuerySnapshot | QueryTemplate
): q is QueryTemplate {
  return 'templateKind' in q;
}

export function getOrderedSavedQueries(
  queries: InsightsQuery[] | undefined
): undefined | InsightsQuery[] {
  if (queries === undefined) return undefined;
  return [...queries].sort((a, b) => a.name.localeCompare(b.name));
}

export function makeQuerySnapshot(query: string, name?: string): QuerySnapshot {
  return {
    id: ulid(),
    isSnapshot: true,
    name: name ?? query,
    query,
  };
}
