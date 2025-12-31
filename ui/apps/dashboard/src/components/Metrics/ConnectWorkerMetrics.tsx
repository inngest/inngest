import { Chart } from '@inngest/components/Chart/Chart';
import { Info } from '@inngest/components/Info/Info';
import { Link } from '@inngest/components/Link';

import type { VolumeMetricsQuery } from '@/gql/graphql';
import type { EntityLookup } from './Dashboard';
import { getLineChartOptions, mapEntityLines } from './utils';

export const ConnectWorkerPercentage = ({
  workspace,
  entities,
}: {
  workspace?: VolumeMetricsQuery['workspace'];
  entities: EntityLookup;
}) => {
  const metrics =
    workspace &&
    mapEntityLines(workspace.workerPercentageUsed.metrics, entities);

  // Override yAxis for percentage formatting
  const chartData = metrics
    ? {
        ...metrics,
        yAxis: {
          type: 'value' as const,
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
            text="Percentage of Connect worker capacity currently being used. For workers with no concurrency limit, this will be 0%."
            action={
              <Link
                className="text-sm"
                href="https://www.inngest.com/docs/setup/connect"
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
          option={
            chartData
              ? getLineChartOptions(chartData, chartData.legendData)
              : {}
          }
          className="relative h-full w-full overflow-visible"
          group="metricsDashboard"
        />
      </div>
    </div>
  );
};

export const ConnectWorkerTotalCapacity = ({
  workspace,
  entities,
}: {
  workspace?: VolumeMetricsQuery['workspace'];
  entities: EntityLookup;
}) => {
  const chartData =
    workspace &&
    mapEntityLines(workspace.workerTotalCapacity.metrics, entities);

  return (
    <div className="bg-canvasBase border-subtle relative flex h-[384px] w-full flex-col overflow-visible rounded-md border p-5">
      <div className="mb-2 flex flex-row items-center justify-between">
        <div className="text-subtle flex w-full flex-row items-center gap-x-2 text-lg">
          Connect Worker Total Capacity{' '}
          <Info
            text="Total capacity available across all Connect workers. For workers with no concurrency limit, this will be 0."
            action={
              <Link
                className="text-sm"
                href="https://www.inngest.com/docs/setup/connect"
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
          option={
            chartData
              ? getLineChartOptions(chartData, chartData.legendData)
              : {}
          }
          className="relative h-full w-full overflow-visible"
          group="metricsDashboard"
        />
      </div>
    </div>
  );
};
