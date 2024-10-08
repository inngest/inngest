import { Chart } from '@inngest/components/Chart/Chart';
import { Info } from '@inngest/components/Info/Info';
import { NewLink } from '@inngest/components/Link/Link';
import { resolveColor } from '@inngest/components/utils/colors';
import { isDark } from '@inngest/components/utils/theme';

import type { VolumeMetricsQuery } from '@/gql/graphql';
import type { EntityLookup } from './Dashboard';
import { dateFormat, getLineChartOptions, lineColors, seriesOptions, timeDiff } from './utils';

const zeroID = '00000000-0000-0000-0000-000000000000';

export const mapConcurrency = (
  {
    concurrency: { metrics: limitMetrics },
    stepRunning: { metrics: runningMetrics },
  }: VolumeMetricsQuery['workspace'],
  entities: EntityLookup,
  concurrencyLimit: number
) => {
  const dark = isDark();

  const diff = timeDiff(limitMetrics[0]?.data[0]?.bucket, limitMetrics[0]?.data.at(-1)?.bucket);
  const dataLength = limitMetrics[0]?.data?.length || 30;

  const metrics = {
    yAxis: {
      max: ({ max }: { max: number }) => (max > concurrencyLimit ? max : concurrencyLimit),
    },

    xAxis: {
      type: 'category' as const,
      boundaryGap: true,
      data: runningMetrics[0]?.data.map(({ bucket }) => bucket) || ['No Data Found'],
      axisPointer: { show: true, type: 'line' as const },
      axisLabel: {
        interval: dataLength <= 40 ? 2 : dataLength / (dataLength / 12),
        formatter: (value: string) => dateFormat(value, diff),
        margin: 10,
      },
    },

    series: [
      ...runningMetrics
        .filter(({ id }) => id !== zeroID)
        .map((f, i) => ({
          ...{ ...seriesOptions, stack: 'Total' },
          silent: true,
          name: entities[f.id]?.name,
          data: f.data.map(({ value }) => value),
          itemStyle: {
            color: resolveColor(lineColors[i % lineColors.length]![0]!, dark, lineColors[0]?.[1]),
          },
          areaStyle: { opacity: 1 },
        })),
      {
        ...seriesOptions,
        markLine: {
          symbol: 'none',
          barMinHeight: '100%',
          large: true,
          animation: false,
          lineStyle: {
            type: 'solid' as any,
            color: resolveColor(lineColors[3]![0]!, dark, lineColors[3]![1]),
          },
          data: [{ yAxis: concurrencyLimit, name: 'Concurrency Limit', symbol: 'none' }],

          emphasis: {
            label: {
              show: true,
              color: 'inherit',
              position: 'insideStartTop' as const,
              formatter: ({ value }: any) => {
                return ` Plan Limit: ${value}\n\n`;
              },
            },
          },
        },
      },
    ],
  };

  return getLineChartOptions(
    metrics,
    runningMetrics.length
      ? runningMetrics.map(({ id }) => ({ name: entities[id]?.name }))
      : ['No Data Found']
  );
};

export const AccountConcurrency = ({
  workspace,
  entities,
  concurrencyLimit,
}: {
  workspace?: VolumeMetricsQuery['workspace'];
  entities: EntityLookup;
  concurrencyLimit: number;
}) => {
  const chartOptions = workspace && mapConcurrency(workspace, entities, concurrencyLimit);

  return (
    <div className="bg-canvasBase border-subtle relative flex h-[384px] w-full flex-col overflow-x-hidden rounded-lg border p-5">
      <div className="mb-2 flex flex-row items-center justify-between">
        <div className="text-subtle flex w-full flex-row items-center gap-x-2 text-lg">
          Account Concurrency{' '}
          <Info
            text="Total number of steps running compared to the account-level concurrency limits."
            action={
              <NewLink
                arrowOnHover
                className="text-sm"
                href="https://www.inngest.com/docs/guides/concurrency#concurrency-use-cases"
              >
                Learn more about concurrency.
              </NewLink>
            }
          />
        </div>
      </div>
      <div className="flex h-full flex-row items-center">
        <Chart
          option={chartOptions ? chartOptions : {}}
          className="h-full w-full"
          group="metricsDashboard"
        />
      </div>
    </div>
  );
};
