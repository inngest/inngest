import { Button } from '@inngest/components/Button';
import { Chart } from '@inngest/components/Chart/Chart';
import { Info } from '@inngest/components/Info/Info';
import { Link } from '@inngest/components/Link/Link';
import { resolveColor } from '@inngest/components/utils/colors';
import { isDark } from '@inngest/components/utils/theme';
import resolveConfig from 'tailwindcss/resolveConfig';

import type { VolumeMetricsQuery } from '@/gql/graphql';
import { pathCreator } from '@/utils/urls';
import tailwindConfig from '../../../tailwind.config';
import type { EntityLookup } from './Dashboard';
import { getLineChartOptions, getXAxis, lineColors, seriesOptions } from './utils';

const zeroID = '00000000-0000-0000-0000-000000000000';
const {
  theme: { borderColor },
} = resolveConfig(tailwindConfig);

export const mapConcurrency = (
  { stepRunning: { metrics: runningMetrics } }: VolumeMetricsQuery['workspace'],
  entities: EntityLookup
) => {
  const dark = isDark();

  const metrics = {
    yAxis: {
      splitLine: {
        lineStyle: { color: resolveColor(borderColor.subtle, dark, '#E2E2E2') },
      },
    },
    xAxis: getXAxis(runningMetrics),
    series: [
      ...runningMetrics
        .filter(({ id }) => id !== zeroID)
        .map((f, i) => ({
          ...seriesOptions,
          name: entities[f.id]?.name,
          data: f.data.map(({ value }) => value),
          itemStyle: {
            color: resolveColor(lineColors[i % lineColors.length]![0]!, dark, lineColors[0]?.[1]),
          },
        })),
    ],
  };

  return getLineChartOptions(
    metrics,
    runningMetrics.length
      ? runningMetrics.map(({ id }) => ({ name: entities[id]?.name }))
      : ['No Data Found']
  );
};

export const Concurrency = ({
  workspace,
  entities,
  isMarketplace = false,
}: {
  workspace?: VolumeMetricsQuery['workspace'];
  entities: EntityLookup;
  isMarketplace?: boolean;
}) => {
  const chartOptions = workspace && mapConcurrency(workspace, entities);

  return (
    <div className="bg-canvasBase border-subtle relative flex h-[384px] w-full flex-col overflow-x-hidden rounded-md border p-5">
      <div className="mb-2 flex flex-row items-center justify-between">
        <div className="text-subtle flex w-full flex-row items-center gap-x-2 text-lg">
          Concurrency{' '}
          <Info
            text="The number of concurrently running steps within this environment"
            action={
              <Link
                arrowOnHover
                className="text-sm"
                href="https://www.inngest.com/docs/guides/concurrency#concurrency-use-cases"
              >
                Learn more about concurrency.
              </Link>
            }
          />
        </div>
        {!isMarketplace && (
          <Button
            label="Increase Concurrency"
            kind="secondary"
            appearance="outlined"
            href={pathCreator.billing({ ref: 'app-concurrency-chart', highlight: 'concurrency' })}
          />
        )}
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
