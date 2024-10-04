import { Chart } from '@inngest/components/Chart/Chart';
import { Info } from '@inngest/components/Info/Info';
import { NewLink } from '@inngest/components/Link/Link';
import { resolveColor } from '@inngest/components/utils/colors';
import { isDark } from '@inngest/components/utils/theme';

import type { ScopedMetric, VolumeMetricsQuery } from '@/gql/graphql';
import { dateFormat, getLineChartOptions, lineColors, seriesOptions, timeDiff } from './utils';

const accumulator: { value: number }[] = [];

//
// Throughput metrics are aggregated by function or app like so
// [{id: x, data: {value}}, {id: x, data: {value}}]
// all we care about are totals so sum values per interval across all
const sum = (metrics: ScopedMetric[]) =>
  metrics.reduce(
    (acc, metric: ScopedMetric) =>
      metric.data.map((item, index) => ({
        value: (acc[index]?.value || 0) + item.value,
      })),
    accumulator
  );

export const mapSdkThroughput = (
  {
    sdkThroughputScheduled,
    sdkThroughputStarted,
    sdkThroughputEnded,
  }: VolumeMetricsQuery['workspace'],
  areaStyle?: { opacity: number }
) => {
  const dark = isDark();

  const intervals = sdkThroughputScheduled.metrics[0]?.data || [];
  const diff = timeDiff(intervals[0]?.bucket, intervals.at(-1)?.bucket);
  const dataLength = intervals.length || 30;

  const metrics = [
    {
      name: 'Queued',
      data: sum(sdkThroughputScheduled.metrics),
    },
    { name: 'Started', data: sum(sdkThroughputStarted.metrics) },
    { name: 'Ended', data: sum(sdkThroughputEnded.metrics) },
  ];

  return {
    xAxis: {
      type: 'category' as const,
      boundaryGap: true,
      data: intervals.length ? intervals.map(({ bucket }) => bucket) : ['No Data Found'],
      axisLabel: {
        interval: dataLength <= 40 ? 2 : dataLength / (dataLength / 12),
        formatter: (value: string) => dateFormat(value, diff),
        margin: 10,
      },
    },
    series: metrics.map((m, i) => {
      return {
        ...seriesOptions,
        name: m.name,
        data: m.data.map(({ value }) => value),
        itemStyle: {
          color: resolveColor(lineColors[i % lineColors.length]![0]!, dark, lineColors[0]?.[1]),
        },
        areaStyle,
      };
    }),
  };
};

export const SdkThroughput = ({ workspace }: { workspace?: VolumeMetricsQuery['workspace'] }) => {
  const metrics = workspace && mapSdkThroughput(workspace, { opacity: 0.1 });

  return (
    <div className="bg-canvasBase border-subtle relative flex h-[384px] w-full flex-col overflow-x-hidden rounded-lg border p-5 md:w-[75%]">
      <div className="mb-2 flex w-full flex-row items-center justify-between p-0">
        <div className="text-subtle flex w-full flex-row items-center gap-x-2 text-lg">
          SDK request throughput{' '}
          <Info
            text="Total number of requests to Inngest SDKs from functions in your apps."
            action={
              <NewLink
                arrowOnHover
                className="text-sm"
                href="https://www.inngest.com/docs/sdk/overview"
              >
                Learn more about SDK throughput.
              </NewLink>
            }
          />
        </div>
      </div>
      <div className="flex h-full flex-row items-center">
        <Chart option={metrics ? getLineChartOptions(metrics) : {}} className="h-full w-full" />
      </div>
    </div>
  );
};
