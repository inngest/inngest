import { useQuery } from 'urql';

import type { TimeRange } from '@/app/(dashboard)/env/[environmentSlug]/functions/[slug]/logs/TimeRangeFilter';
import StackedBarChart from '@/components/Charts/StackedBarChart';
import { graphql } from '@/gql';
import { useEnvironment } from '@/queries';

export type UsageMetrics = { totalRuns: number; totalFailures: number };

const GetFunctionRunsMetricsDocument = graphql(`
  query GetFunctionRunsMetrics(
    $environmentID: ID!
    $functionSlug: String!
    $startTime: Time!
    $endTime: Time!
  ) {
    environment: workspace(id: $environmentID) {
      function: workflowBySlug(slug: $functionSlug) {
        completed: usage(opts: { from: $startTime, to: $endTime }, event: "completed") {
          period
          total
          asOf
          data {
            slot
            count
          }
        }
        canceled: usage(opts: { from: $startTime, to: $endTime }, event: "cancelled") {
          period
          total
          asOf
          data {
            slot
            count
          }
        }
        failed: usage(opts: { from: $startTime, to: $endTime }, event: "errored") {
          period
          total
          asOf
          data {
            slot
            count
          }
        }
      }
    }
  }
`);

type FunctionRunsChartProps = {
  environmentSlug: string;
  functionSlug: string;
  timeRange: TimeRange;
};

export default function FunctionRunsChart({
  environmentSlug,
  functionSlug,
  timeRange,
}: FunctionRunsChartProps) {
  const [{ data: environment }] = useEnvironment({
    environmentSlug,
  });

  const [{ data }] = useQuery({
    query: GetFunctionRunsMetricsDocument,
    variables: {
      environmentID: environment?.id!,
      functionSlug,
      startTime: timeRange.start.toISOString(),
      endTime: timeRange.end.toISOString(),
    },
    pause: !environment?.id,
  });

  let metrics: {
    name: string;
    values: {
      completed: number;
      canceled: number;
      failed: number;
    };
  }[] = [];
  const completed = data?.environment.function?.completed;
  const canceled = data?.environment.function?.canceled;
  const failed = data?.environment.function?.failed;

  if (completed && canceled && failed) {
    metrics = completed.data.map((d, i) => ({
      name: d.slot,
      values: {
        completed: d.count,
        canceled: canceled.data[i]?.count ?? 0,
        failed: failed.data[i]?.count ?? 0,
      },
    }));
  }

  return (
    <StackedBarChart
      title="Function Runs"
      data={metrics}
      legend={[
        { name: 'Completed', dataKey: 'completed', color: '#14B8A6' },
        { name: 'Failed', dataKey: 'failed', color: '#EF4444' },
        { name: 'Canceled', dataKey: 'canceled', color: '#64748B' },
      ]}
      isLoading={isFetching}
    />
  );
}
