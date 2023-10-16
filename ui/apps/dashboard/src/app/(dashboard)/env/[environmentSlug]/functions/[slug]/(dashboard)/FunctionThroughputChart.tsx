'use client';

import { useQuery } from 'urql';

import type { TimeRange } from '@/app/(dashboard)/env/[environmentSlug]/functions/[slug]/logs/TimeRangeFilter';
import SimpleLineChart from '@/components/Charts/SimpleLineChart';
import { graphql } from '@/gql';
import { useEnvironment } from '@/queries';

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
      }
    }
  }
`);

type FunctionThroughputChartProps = {
  environmentSlug: string;
  functionSlug: string;
  timeRange: TimeRange;
};

export default function FunctionThroughputChart({
  environmentSlug,
  functionSlug,
  timeRange,
}: FunctionThroughputChartProps) {
  const [{ data: environment, fetching: isFetchingEnvironment }] = useEnvironment({
    environmentSlug,
  });

  const [{ data, fetching: isFetchingMetrics }] = useQuery({
    query: GetFnRunMetricsDocument,
    variables: {
      environmentID: environment?.id!,
      fnSlug: functionSlug,
      startTime: timeRange.start.toISOString(),
      endTime: timeRange.end.toISOString(),
    },
    pause: !environment?.id,
  });

  let metrics: {
    name: string;
    values: {
      queued: number;
      started: number;
      ended: number;
    };
  }[] = [];
  const queued = data?.environment.function?.queued;
  const started = data?.environment.function?.started;
  const ended = data?.environment.function?.ended;

  if (queued && started && ended) {
    metrics = queued.data.map((d, idx) => {
      const startedCount = started.data[idx]?.value || 0;
      const endedCount = ended.data[idx]?.value || 0;

      return {
        name: d.bucket,
        values: {
          queued: d.value,
          started: startedCount,
          ended: endedCount,
        },
      };
    });
  }

  return (
    <SimpleLineChart
      title="Function Throughput"
      desc="The number of functions being processed over time"
      data={metrics}
      legend={[
        { name: 'queued', dataKey: 'queued', color: '#fa8128' },
        { name: 'started', dataKey: 'started', color: '#82ca9d' },
        { name: 'ended', dataKey: 'ended', color: '#8884d8' },
      ]}
      isLoading={isFetchingEnvironment || isFetchingMetrics}
    />
  );
}
