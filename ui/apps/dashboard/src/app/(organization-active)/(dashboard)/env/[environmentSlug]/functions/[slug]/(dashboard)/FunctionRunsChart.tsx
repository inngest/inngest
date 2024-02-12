'use client';

import colors from 'tailwindcss/colors';
import { useQuery } from 'urql';

import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
import type { TimeRange } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/functions/[slug]/logs/TimeRangeFilter';
import StackedBarChart from '@/components/Charts/StackedBarChart';
import { graphql } from '@/gql';

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
          data {
            slot
            count
          }
        }
        canceled: usage(opts: { from: $startTime, to: $endTime }, event: "cancelled") {
          period
          total
          data {
            slot
            count
          }
        }
        failed: usage(opts: { from: $startTime, to: $endTime }, event: "errored") {
          period
          total
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
  functionSlug: string;
  timeRange: TimeRange;
};

export default function FunctionRunsChart({ functionSlug, timeRange }: FunctionRunsChartProps) {
  const environment = useEnvironment();

  const [{ data, error: functionRunsMetricsError, fetching: isFetchingFunctionRunsMetrics }] =
    useQuery({
      query: GetFunctionRunsMetricsDocument,
      variables: {
        environmentID: environment.id,
        functionSlug,
        startTime: timeRange.start.toISOString(),
        endTime: timeRange.end.toISOString(),
      },
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
        { name: 'Completed', dataKey: 'completed', color: colors.teal['500'] },
        { name: 'Failed', dataKey: 'failed', color: colors.red['500'] },
        { name: 'Canceled', dataKey: 'canceled', color: colors.slate['500'] },
      ]}
      isLoading={isFetchingFunctionRunsMetrics}
      error={functionRunsMetricsError}
    />
  );
}
