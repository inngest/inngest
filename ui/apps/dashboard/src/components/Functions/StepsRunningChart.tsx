import colors from 'tailwindcss/colors';
import { useQuery } from 'urql';

import SimpleLineChart from '@/components/Charts/SimpleLineChart';
import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';
import { mergeStepsRunningMetrics } from './mergeStepsRunningMetrics';

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

type StepsRunningChartProps = {
  functionSlug: string;
  startTime: string;
  endTime: string;
};

export default function StepsRunningChart({
  functionSlug,
  startTime,
  endTime,
}: StepsRunningChartProps) {
  const environment = useEnvironment();

  const [{ data, error: metricsError, fetching: isFetchingMetrics }] = useQuery(
    {
      query: GetStepsRunningDocument,
      variables: {
        environmentID: environment.id,
        fnSlug: functionSlug,
        startTime,
        endTime,
      },
    },
  );

  const running = data?.environment.function?.running.data ?? [];
  const concurrencyLimit =
    data?.environment.function?.concurrencyLimit.data ?? [];
  const granularity =
    data?.environment.function?.running.granularity ??
    data?.environment.function?.concurrencyLimit.granularity ??
    '1m';
  // Both responses cover the same requested range, so use either one for the bounds.
  const rangeStart = data?.environment.function?.running.from ?? startTime;
  const rangeEnd = data?.environment.function?.running.to ?? endTime;
  const metrics = mergeStepsRunningMetrics(
    running,
    concurrencyLimit,
    granularity,
    rangeStart,
    rangeEnd,
  );

  return (
    <SimpleLineChart
      title="Step Running - Point in Time"
      desc="The number of steps running for this function at point in time. This data shows the value at the time of instrumentation, and is different from throughput. The chart background changes color when the function's concurrency limit is reached."
      data={metrics}
      legend={[
        {
          name: 'Concurrency Limit Hit',
          dataKey: 'concurrencyLimit',
          color: colors.amber['500'],
          referenceArea: true,
        },
        { name: 'Running', dataKey: 'running', color: colors.blue['500'] },
      ]}
      connectNulls
      isLoading={isFetchingMetrics}
      error={metricsError}
    />
  );
}
