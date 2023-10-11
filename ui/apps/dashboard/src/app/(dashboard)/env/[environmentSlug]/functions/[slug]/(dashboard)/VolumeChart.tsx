import { getTimeRangeLabel } from '@/app/(dashboard)/env/[environmentSlug]/functions/[slug]/(dashboard)/DashboardTimeRangeFilter';
import type { TimeRange } from '@/app/(dashboard)/env/[environmentSlug]/functions/[slug]/logs/TimeRangeFilter';
import StackedBarChart from '@/components/Charts/StackedBarChart';
import { useFunctionUsage } from '@/queries';

export type UsageMetrics = { totalRuns: number; totalFailures: number };

type VolumeChartProps = {
  environmentSlug: string;
  functionSlug: string;
  timeRange: TimeRange;
};

export default function VolumeChart({
  environmentSlug,
  functionSlug,
  timeRange,
}: VolumeChartProps) {
  const [{ data: usage }] = useFunctionUsage({
    environmentSlug: environmentSlug,
    functionSlug: functionSlug,
    timeRange: timeRange,
  });

  const usageMetrics: UsageMetrics | undefined = usage?.reduce(
    (acc, u) => {
      acc.totalRuns += u.values.totalRuns;
      acc.totalFailures += u.values.failures;
      return acc;
    },
    {
      totalRuns: 0,
      totalFailures: 0,
    }
  );

  return (
    <StackedBarChart
      title="Usage"
      data={usage}
      total={usageMetrics?.totalRuns || 0}
      totalDescription={`${getTimeRangeLabel(timeRange)} Volume`}
      legend={[
        { name: 'Failures', dataKey: 'failures', color: '#ef4444' },
        {
          name: 'Runs',
          dataKey: 'successes',
          color: '#6266f1',
          default: true,
        },
      ]}
    />
  );
}
