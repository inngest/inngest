'use client';

import colors from 'tailwindcss/colors';
import { useQuery } from 'urql';

import type { TimeRange } from '@/types/TimeRangeFilter';
import SimpleLineChart from '@/components/Charts/SimpleLineChart';
import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';

const GetFunctionRateLimitDocument = graphql(`
  query GetFunctionRateLimitDocument(
    $environmentID: ID!
    $fnSlug: String!
    $startTime: Time!
    $endTime: Time!
  ) {
    environment: workspace(id: $environmentID) {
      function: workflowBySlug(slug: $fnSlug) {
        ratelimit: metrics(
          opts: { name: "function_run_rate_limited_total", from: $startTime, to: $endTime }
        ) {
          from
          to
          granularity
          data {
            bucket
            value
          }
        }
      }
    }
  }
`);

type FunctionRateLimitChartProps = {
  functionSlug: string;
  timeRange: TimeRange;
};

export default function FunctionRunRateLimitChart({
  functionSlug,
  timeRange,
}: FunctionRateLimitChartProps) {
  const environment = useEnvironment();

  const [{ data, error: metricsError, fetching: isFetchingMetrics }] = useQuery({
    query: GetFunctionRateLimitDocument,
    variables: {
      environmentID: environment.id,
      fnSlug: functionSlug,
      startTime: timeRange.start.toISOString(),
      endTime: timeRange.end.toISOString(),
    },
  });

  const ratelimit = data?.environment.function?.ratelimit.data ?? [];

  const metrics = Array.from({ length: ratelimit.length }).map((_, idx) => ({
    name: ratelimit[idx]?.bucket || '',
    values: {
      ratelimit: ratelimit[idx]?.value ?? 0,
    },
  }));

  return (
    <SimpleLineChart
      title="Function RateLimit"
      desc="The number of runs that got dropped due to rate limit settings"
      data={metrics}
      legend={[{ name: 'Rate Limited', dataKey: 'ratelimit', color: colors.slate['500'] }]}
      isLoading={isFetchingMetrics}
      error={metricsError}
    />
  );
}
