import { Chart } from '@inngest/components/Chart/Chart';
import { Info } from '@inngest/components/Info/Info';
import { Link } from '@inngest/components/Link/Link';
import { resolveColor } from '@inngest/components/utils/colors';
import { isDark } from '@inngest/components/utils/theme';

import type { VolumeMetricsQuery } from '@/gql/graphql';
import { getLineChartOptions, getXAxis, lineColors, seriesOptions } from './utils';

export const ConnectWorkerPercentage = ({
  data,
}: {
  data?: VolumeMetricsQuery['workerPercentageUsedTimeSeries'];
}) => {
  const dark = isDark();

  const chartData = data
    ? {
        xAxis: getXAxis(data),
        series: [
          {
            ...seriesOptions,
            name: 'Worker Percentage Used',
            data: data.data.map(({ value }) => value),
            itemStyle: {
              color: resolveColor(lineColors[0]![0]!, dark, lineColors[0]?.[1]),
            },
          },
        ],
        yAxis: {
          type: 'value',
          max: 100,
          min: 0,
          axisLabel: {
            formatter: '{value}%',
          },
        },
      }
    : null;

  return (
    <div className="bg-canvasBase border-subtle relative flex h-[384px] w-full flex-col overflow-visible rounded-md border p-5">
      <div className="mb-2 flex flex-row items-center justify-between">
        <div className="text-subtle flex w-full flex-row items-center gap-x-2 text-lg">
          Connect Worker Usage{' '}
          <Info
            text="Percentage of Connect worker capacity currently being used."
            action={
              <Link
                className="text-sm"
                href="https://www.inngest.com/docs/platform/connect"
                target="_new"
              >
                Learn more about Connect Workers.
              </Link>
            }
          />
        </div>
      </div>
      <div className="flex h-full flex-row items-center overflow-visible">
        <Chart
          option={chartData ? getLineChartOptions(chartData) : {}}
          className="relative h-full w-full overflow-visible"
          group="metricsDashboard"
        />
      </div>
    </div>
  );
};

export const ConnectWorkerTotalCapacity = ({
  data,
}: {
  data?: VolumeMetricsQuery['workerTotalCapacityTimeSeries'];
}) => {
  const dark = isDark();

  const chartData = data
    ? {
        xAxis: getXAxis(data),
        series: [
          {
            ...seriesOptions,
            name: 'Total Capacity',
            data: data.data.map(({ value }) => value),
            itemStyle: {
              color: resolveColor(lineColors[1]![0]!, dark, lineColors[1]?.[1]),
            },
          },
        ],
      }
    : null;

  return (
    <div className="bg-canvasBase border-subtle relative flex h-[384px] w-full flex-col overflow-visible rounded-md border p-5">
      <div className="mb-2 flex flex-row items-center justify-between">
        <div className="text-subtle flex w-full flex-row items-center gap-x-2 text-lg">
          Connect Worker Total Capacity{' '}
          <Info
            text="Total capacity available across all Connect workers."
            action={
              <Link
                className="text-sm"
                href="https://www.inngest.com/docs/platform/connect"
                target="_new"
              >
                Learn more about Connect Workers.
              </Link>
            }
          />
        </div>
      </div>
      <div className="flex h-full flex-row items-center overflow-visible">
        <Chart
          option={chartData ? getLineChartOptions(chartData) : {}}
          className="relative h-full w-full overflow-visible"
          group="metricsDashboard"
        />
      </div>
    </div>
  );
};
