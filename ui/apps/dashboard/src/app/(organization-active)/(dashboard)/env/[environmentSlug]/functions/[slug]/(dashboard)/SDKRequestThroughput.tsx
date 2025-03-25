'use client';

import colors from 'tailwindcss/colors';
import { useQuery } from 'urql';

import type { TimeRange } from '@/types/TimeRangeFilter';
import SimpleLineChart from '@/components/Charts/SimpleLineChart';
import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';

const GetSDKReqMetricsDocument = graphql(`
  query GetSDKRequestMetrics(
    $environmentID: ID!
    $fnSlug: String!
    $startTime: Time!
    $endTime: Time!
  ) {
    environment: workspace(id: $environmentID) {
      function: workflowBySlug(slug: $fnSlug) {
        queued: metrics(opts: { name: "sdk_req_scheduled_total", from: $startTime, to: $endTime }) {
          from
          to
          granularity
          data {
            bucket
            value
          }
        }
        started: metrics(opts: { name: "sdk_req_started_total", from: $startTime, to: $endTime }) {
          from
          to
          granularity
          data {
            bucket
            value
          }
        }

        ended: metrics(opts: { name: "sdk_req_ended_total", from: $startTime, to: $endTime }) {
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
  functionSlug: string;
  timeRange: TimeRange;
};

export default function SDKReqThroughputChart({
  functionSlug,
  timeRange,
}: SDKReqThroughputChartProps) {
  const environment = useEnvironment();

  const [{ data, error: metricsError, fetching: isFetchingMetrics }] = useQuery({
    query: GetSDKReqMetricsDocument,
    variables: {
      environmentID: environment.id,
      fnSlug: functionSlug,
      startTime: timeRange.start.toISOString(),
      endTime: timeRange.end.toISOString(),
    },
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
    metrics = queued.data.map((d, idx) => ({
      name: d.bucket,
      values: {
        queued: d.value,
        started: started.data[idx]?.value || 0,
        ended: ended.data[idx]?.value || 0,
      },
    }));
  }

  return (
    <SimpleLineChart
      title="SDK Request Throughput"
      desc="The number of requests to your SDKs over time running the function and steps, including retries."
      data={metrics}
      legend={[
        { name: 'Queued', dataKey: 'queued', color: colors.slate['500'] },
        { name: 'Started', dataKey: 'started', color: colors.sky['500'] },
        { name: 'Ended', dataKey: 'ended', color: colors.teal['500'] },
      ]}
      isLoading={isFetchingMetrics}
      error={metricsError}
    />
  );
}
