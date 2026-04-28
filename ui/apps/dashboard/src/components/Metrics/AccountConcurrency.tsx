import { Button } from '@inngest/components/Button';
import { Chart, type LineSeriesOption } from '@inngest/components/Chart/Chart';
import { Info } from '@inngest/components/Info/Info';
import { Link } from '@inngest/components/Link';
import { resolveColor } from '@inngest/components/utils/colors';
import { isDark } from '@inngest/components/utils/theme';

import type { VolumeMetricsQuery } from '@/gql/graphql';
import { pathCreator } from '@/utils/urls';
import { borderColor } from '@/utils/tailwind';
import { getLineChartOptions, lineColors, seriesOptions } from './utils';

type Props = {
  accountConcurrency: VolumeMetricsQuery['accountConcurrency'] | undefined;
  limit?: number;
  isMarketplace: boolean;
};

export function AccountConcurrency({
  accountConcurrency,
  limit,
  isMarketplace = false,
}: Props) {
  let option = {};
  if (accountConcurrency) {
    option = createChartOption({ limit, accountConcurrency });
  }

  return (
    <div className="bg-canvasBase border-subtle relative flex h-[384px] w-full flex-col overflow-x-hidden rounded-md border p-5">
      <div className="mb-2 flex flex-row items-center justify-between">
        <div className="text-subtle flex w-full flex-row items-center gap-x-2 text-lg">
          Account Concurrency
          <Info
            action={
              <Link
                className="text-sm"
                href="https://www.inngest.com/docs/guides/concurrency#concurrency-use-cases"
              >
                Learn more about concurrency.
              </Link>
            }
            text="The number of concurrently running steps across all environments"
          />
        </div>
        {!isMarketplace && (
          <Button
            appearance="outlined"
            href={pathCreator.billing({
              highlight: 'concurrency',
              ref: 'app-concurrency-chart',
            })}
            kind="secondary"
            label="Increase Concurrency"
          />
        )}
      </div>
      <div className="flex h-full flex-row items-center">
        <Chart
          className="h-full w-full"
          group="metricsDashboard"
          option={option}
        />
      </div>
    </div>
  );
}

function createChartOption({
  limit,
  accountConcurrency,
}: {
  limit: number | undefined;
  accountConcurrency: VolumeMetricsQuery['accountConcurrency'];
}): React.ComponentProps<typeof Chart>['option'] {
  const dark = isDark();

  const series: LineSeriesOption[] = [
    {
      ...seriesOptions,
      name: 'Account Concurrency',
      data: accountConcurrency.data.map(({ value }) => value),
      itemStyle: {
        color: resolveColor(lineColors[0][0], dark, lineColors[0]?.[1]),
      },
      areaStyle: { opacity: 0.3 },
    },
  ];

  if (limit) {
    series.push({
      ...seriesOptions,
      markLine: {
        animation: false,
        data: [{ yAxis: limit, name: 'Concurrency Limit', symbol: 'none' }],
        emphasis: {
          label: {
            color: 'inherit',
            formatter: ({ value }: any) => {
              return ` Plan Limit: ${value}\n\n`;
            },
            position: 'insideStartTop' as const,
            show: true,
          },
        },
        lineStyle: {
          type: 'solid' as any,
          color: resolveColor(lineColors[3][0], dark, lineColors[3][1]),
        },
        symbol: 'none',
        tooltip: {
          show: false,
        },
      },
    });
  }

  const xAxisData = accountConcurrency.data.map(({ bucket }) => bucket);

  return getLineChartOptions(
    {
      series,
      xAxis: {
        data: xAxisData,
      },
      yAxis: {
        max: ({ max }: { max: number }) => {
          if (limit && max < limit) {
            return Math.round(limit * 1.1);
          }
          return max;
        },
        splitLine: {
          lineStyle: {
            color: resolveColor(borderColor.subtle, dark, '#E2E2E2'),
          },
        },
      },
    },
    [{ name: 'Account Concurrency' }],
  );
}
