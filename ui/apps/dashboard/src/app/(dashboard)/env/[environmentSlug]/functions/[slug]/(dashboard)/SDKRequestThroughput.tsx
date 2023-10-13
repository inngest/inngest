'use client';

import { useQuery } from 'urql';

import type { TimeRange } from '@/app/(dashboard)/env/[environmentSlug]/functions/[slug]/logs/TimeRangeFilter';
import SimpleLineChart from '@/components/Charts/SimpleLineChart';
import { graphql } from '@/gql';
import { useEnvironment } from '@/queries';

const GetSDKReqMetricsDocument = graphql(`
  query GetSDKRequestMetrics(
    $environmentID: ID!
    $fnSlug: String!
    $startTime: Time!
    $endTime: Time!
  ) {
    environment: workspace(id: $environmentID) {
      function: workflowBySlug(slug: $fnSlug) {
        queued: metrics(
          opts: { name: "step_run_scheduled_total", from: $startTime, to: $endTime }
        ) {
          from
          to
          granularity
          data {
            bucket
            value
          }
        }
        started: metrics(opts: { name: "step_run_started_total", from: $startTime, to: $endTime }) {
          from
          to
          granularity
          data {
            bucket
            value
          }
        }

        ended: metrics(opts: { name: "step_run_ended_total", from: $startTime, to: $endTime }) {
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

type SDKReqThroughputChartProps = {
  environmentSlug: string;
  functionSlug: string;
  timeRange: TimeRange;
};

export default function SDKReqThroughputChart({
  environmentSlug,
  functionSlug,
  timeRange,
}: SDKReqThroughputChartProps) {
  const [{ data: environment }] = useEnvironment({
    environmentSlug,
  });

  const [{ data, fetching }] = useQuery({
    query: GetSDKReqMetricsDocument,
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
      title="SDK Request Throughput"
      desc="The number of requests to your SDKs over time executing the function and steps, including retries"
      data={metrics}
      legend={[
        { name: 'queued', dataKey: 'queued', color: '#fa8128' },
        { name: 'started', dataKey: 'started', color: '#82ca9d' },
        { name: 'ended', dataKey: 'ended', color: '#8884d8' },
      ]}
    />
  );
}
