import type { SortingState } from '@tanstack/react-table';
import { useQuery } from 'urql';

import { InsightsMetricOrderByDirection } from '@/gql/graphql';
import { InsightsMetricDocument } from './queries';
import type { MetricTable } from './types';

// sortingToOrderBy converts a RankedTable's tanstack SortingState into this
// hook's own `orderBy` input — null (the registry's own default order) when
// nothing's been clicked yet. Shared by every AI Overview widget that wires
// a sortable RankedTable up to a server-sorted useInsightsMetric call.
export function sortingToOrderBy(
  sorting: SortingState,
): { column: string; direction: 'asc' | 'desc' } | null {
  const [sort] = sorting;
  return sort ? { column: sort.id, direction: sort.desc ? 'desc' : 'asc' } : null;
}

// One InsightsMetric request per widget, keyed by the registry key it wants
// — see queries.ts for why this replaced the combined/aliased query. urql
// dedupes identical (query, variables) pairs, so multiple widgets reading
// the same key (e.g. modelDistribution feeding both "Tokens by model" and
// "Cost by model") still share one network request.
export function useInsightsMetric(
  key: string,
  opts: {
    workspaceID: string;
    functionIDs?: string[] | null;
    range: { from: string; to: string };
    limit?: number;
    // Requests the backend re-sort this metric's rows by a different
    // column/direction than the registry's own default (e.g. a RankedTable
    // column the user clicked) — validated server-side against that
    // registry entry's orderableColumns (see pkg/applogic/dashboards/
    // query.go's Get). Lowercase to match tanstack's own SortingState
    // convention (`desc: boolean`), converted to the GraphQL enum here so
    // callers don't need to import it. Changing this changes the query's
    // variables, so urql automatically re-fetches with the new order.
    orderBy?: { column: string; direction: 'asc' | 'desc' } | null;
    // Skips firing the query — e.g. while a caller's own functionID lookup is
    // still resolving, where an unpaused call would fire with an empty
    // functionIDs list and get back env-wide (not "no data yet") rows.
    pause?: boolean;
  },
): { data: MetricTable; fetching: boolean; error: unknown } {
  const [{ data, fetching, error }] = useQuery({
    query: InsightsMetricDocument,
    pause: opts.pause,
    variables: {
      workspaceID: opts.workspaceID,
      functionIDs: opts.functionIDs ?? null,
      key,
      range: opts.range,
      limit: opts.limit ?? null,
      orderBy: opts.orderBy
        ? {
            column: opts.orderBy.column,
            direction:
              opts.orderBy.direction === 'desc'
                ? InsightsMetricOrderByDirection.Desc
                : InsightsMetricOrderByDirection.Asc,
          }
        : null,
    },
  });
  return { data: data?.insightsMetric, fetching, error };
}
