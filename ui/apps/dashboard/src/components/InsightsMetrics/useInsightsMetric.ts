import { useQuery } from 'urql';

import { InsightsMetricDocument } from './queries';
import type { MetricTable } from './types';

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
  },
): { data: MetricTable; fetching: boolean; error: unknown } {
  const [{ data, fetching, error }] = useQuery({
    query: InsightsMetricDocument,
    variables: {
      workspaceID: opts.workspaceID,
      functionIDs: opts.functionIDs ?? null,
      key,
      range: opts.range,
      limit: opts.limit ?? null,
    },
  });
  return { data: data?.insightsMetric, fetching, error };
}
