import colors from 'tailwindcss/colors';
import { useQuery } from 'urql';

import SimpleLineChart from '@/components/Charts/SimpleLineChart';
import { useEnvironment } from '@/components/Environments/environment-context';
import { graphql } from '@/gql';
import { mergeBacklogMetrics } from './mergeBacklogMetrics';

const GetStepBacklogDocument = graphql(`
  query GetStepBacklogMetrics(
    $environmentID: ID!
    $fnSlug: String!
    $startTime: Time!
    $endTime: Time!
  ) {
    environment: workspace(id: $environmentID) {
      function: workflowBySlug(slug: $fnSlug) {
        scheduled: metrics(opts: { name: "steps_scheduled", from: $startTime, to: $endTime }) {
          from
          to
          granularity
          data {
            bucket
            value
          }
        }
        sleeping: metrics(opts: { name: "steps_sleeping", from: $startTime, to: $endTime }) {
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

type StepBacklogChartProps = {
  functionSlug: string;
  startTime: string;
  endTime: string;
};

export default function StepBacklogChart({
  functionSlug,
  startTime,
  endTime,
}: StepBacklogChartProps) {
  const environment = useEnvironment();

  const [{ data, error: metricsError, fetching: isFetchingMetrics }] = useQuery(
    {
      query: GetStepBacklogDocument,
      variables: {
        environmentID: environment.id,
        fnSlug: functionSlug,
        startTime,
        endTime,
      },
    },
  );

  const scheduled = data?.environment.function?.scheduled.data ?? [];
  const sleeping = data?.environment.function?.sleeping.data ?? [];
  const granularity =
    data?.environment.function?.scheduled.granularity ??
    data?.environment.function?.sleeping.granularity ??
    '1m';
  // Both responses cover the same requested range, so use either one for the bounds.
  const rangeStart =
    data?.environment.function?.scheduled.from ?? startTime;
  const rangeEnd =
    data?.environment.function?.scheduled.to ?? endTime;
  const metrics = mergeBacklogMetrics(
    scheduled,
    sleeping,
    granularity,
    rangeStart,
    rangeEnd,
  );

  return (
    <SimpleLineChart
      title="Step Backlog - Point in Time"
      desc="The backlog status of steps for this function at point in time. This data shows the value at the time of instrumentation, and is different from throughput."
      data={metrics}
      legend={[
        { name: 'Queued', dataKey: 'scheduled', color: colors.slate['500'] },
        { name: 'Sleeping', dataKey: 'sleeping', color: colors.teal['500'] },
      ]}
      connectNulls
      isLoading={isFetchingMetrics}
      error={metricsError}
    />
  );
}
