'use client';

import colors from 'tailwindcss/colors';
import { useQuery } from 'urql';

import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
import type { TimeRange } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/functions/[slug]/logs/TimeRangeFilter';
import SimpleLineChart from '@/components/Charts/SimpleLineChart';
import { graphql } from '@/gql';

const GetStepsRunningDocument = graphql(`
  query GetStepsRunningMetrics(
    $environmentID: ID!
    $fnSlug: String!
    $startTime: Time!
    $endTime: Time!
  ) {
    environment: workspace(id: $environmentID) {
      function: workflowBySlug(slug: $fnSlug) {
        running: metrics(opts: { name: "steps_running", from: $startTime, to: $endTime }) {
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

type StepsRunningChartProps = {
  functionSlug: string;
  timeRange: TimeRange;
};

export default function StepsRunningChart({ functionSlug, timeRange }: StepsRunningChartProps) {
  const environment = useEnvironment();

  const [{ data, error: metricsError, fetching: isFetchingMetrics }] = useQuery({
    query: GetStepsRunningDocument,
    variables: {
      environmentID: environment.id,
      fnSlug: functionSlug,
      startTime: timeRange.start.toISOString(),
      endTime: timeRange.end.toISOString(),
    },
  });

  const running = data?.environment.function?.running.data ?? [];
  const metrics = Array.from({ length: running.length }).map((_, idx) => ({
    name: running[idx]?.bucket || '',
    values: {
      running: running[idx]?.value ?? 0,
    },
  }));

  return (
    <SimpleLineChart
      title="Step Running"
      desc="The # of steps running for this function"
      data={metrics}
      legend={[{ name: 'Running', dataKey: 'running', color: colors.blue['500'] }]}
      isLoading={isFetchingMetrics}
      error={metricsError}
    />
  );
}
