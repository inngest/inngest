import { ulid } from 'ulid';

import type { InsightsQueryStatement } from '@/gql/graphql';
import type { QuerySnapshot, QueryTemplate } from './types';

export function isQuerySnapshot(
  q: InsightsQueryStatement | QuerySnapshot,
): q is QuerySnapshot {
  return 'isSnapshot' in q && q.isSnapshot;
}

export function isQueryTemplate(
  q: InsightsQueryStatement | QuerySnapshot | QueryTemplate,
): q is QueryTemplate {
  return 'templateKind' in q;
}

export function getOrderedSavedQueries(
  queries: InsightsQueryStatement[] | undefined,
): undefined | InsightsQueryStatement[] {
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
