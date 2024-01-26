'use client';

import colors from 'tailwindcss/colors';
import { useQuery } from 'urql';

import { useEnvironment } from '@/app/(dashboard)/env/[environmentSlug]/environment-context';
import type { TimeRange } from '@/app/(dashboard)/env/[environmentSlug]/functions/[slug]/logs/TimeRangeFilter';
import SimpleLineChart from '@/components/Charts/SimpleLineChart';
import { graphql } from '@/gql';

const GetFnRunMetricsDocument = graphql(`
  query GetFnMetrics($environmentID: ID!, $fnSlug: String!, $startTime: Time!, $endTime: Time!) {
    environment: workspace(id: $environmentID) {
      function: workflowBySlug(slug: $fnSlug) {
        queued: metrics(
          opts: { name: "function_run_scheduled_total", from: $startTime, to: $endTime }
        ) {
          from
          to
          granularity
          data {
            bucket
            value
          }
        }
        started: metrics(
          opts: { name: "function_run_started_total", from: $startTime, to: $endTime }
        ) {
          from
          to
          granularity
          data {
            bucket
            value
          }
        }
        ended: metrics(opts: { name: "function_run_ended_total", from: $startTime, to: $endTime }) {
          from
          to
          granularity
          data {
            bucket
            value
          }
        }
        concurrencyLimit: metrics(
          opts: { name: "concurrency_limit_reached_total", from: $startTime, to: $endTime }
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

type FunctionThroughputChartProps = {
  functionSlug: string;
  timeRange: TimeRange;
};

export default function FunctionThroughputChart({
  functionSlug,
  timeRange,
}: FunctionThroughputChartProps) {
  const environment = useEnvironment();

  const [{ data, error: metricsError, fetching: isFetchingMetrics }] = useQuery({
    query: GetFnRunMetricsDocument,
    variables: {
      environmentID: environment.id,
      fnSlug: functionSlug,
      startTime: timeRange.start.toISOString(),
      endTime: timeRange.end.toISOString(),
    },
  });

  const queued = data?.environment.function?.queued.data ?? [];
  const started = data?.environment.function?.started.data ?? [];
  const ended = data?.environment.function?.ended.data ?? [];
  const concurrencyLimit = data?.environment.function?.concurrencyLimit.data ?? [];

  const maxLength = Math.max(queued.length, started.length, ended.length, concurrencyLimit.length);

  const metrics = Array.from({ length: maxLength }).map((_, idx) => ({
    name:
      queued[idx]?.bucket ||
      started[idx]?.bucket ||
      ended[idx]?.bucket ||
      concurrencyLimit[idx]?.bucket ||
      '',
    values: {
      queued: queued[idx]?.value ?? 0,
      started: started[idx]?.value ?? 0,
      ended: ended[idx]?.value ?? 0,
      concurrencyLimit: Boolean(concurrencyLimit[idx]?.value),
    },
  }));

  return (
    <SimpleLineChart
      title="Function Throughput"
      desc="The number of function runs being processed over time."
      data={metrics}
      legend={[
        {
          name: 'Concurrency Limit',
          dataKey: 'concurrencyLimit',
          color: colors.amber['500'],
          referenceArea: true,
        },
        { name: 'Queued', dataKey: 'queued', color: colors.slate['500'] },
        { name: 'Started', dataKey: 'started', color: colors.sky['500'] },
        { name: 'Ended', dataKey: 'ended', color: colors.teal['500'] },
      ]}
      isLoading={isFetchingMetrics}
      error={metricsError}
    />
  );
}
