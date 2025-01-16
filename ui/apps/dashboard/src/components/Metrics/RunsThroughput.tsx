import { Chart } from '@inngest/components/Chart/Chart';
import { Info } from '@inngest/components/Info/Info';
import { Link } from '@inngest/components/Link/Link';

import type { VolumeMetricsQuery } from '@/gql/graphql';
import type { EntityLookup } from './Dashboard';
import { getLineChartOptions, mapEntityLines } from './utils';

export type RunsThroughputMetricsType =
  VolumeMetricsQuery['workspace']['runsThroughput']['metrics'];

export const RunsThrougput = ({
  workspace,
  entities,
}: {
  workspace?: VolumeMetricsQuery['workspace'];
  entities: EntityLookup;
}) => {
  const metrics = workspace && mapEntityLines(workspace.runsThroughput.metrics, entities);

  return (
    <div className="bg-canvasBase border-subtle relative flex h-[384px] w-full flex-col overflow-visible rounded-md border p-5">
      <div className="mb-2 flex flex-row items-center justify-between">
        <div className="text-subtle flex w-full flex-row items-center gap-x-2 text-lg">
          Total runs throughput{' '}
          <Info
            text="Total number of runs processed your env, app or function."
            action={
              <Link
                arrowOnHover
                className="text-sm"
                href="https://www.inngest.com/docs/platform/monitor/observability-metrics#total-runs-throughput"
                target="_new"
              >
                Learn more about runs throughput.
              </Link>
            }
          />
        </div>
      </div>
      <div className="flex h-full flex-row items-center overflow-visible">
        <Chart
          option={metrics ? getLineChartOptions(metrics) : {}}
          className="relative h-full w-full overflow-visible"
          group="metricsDashboard"
        />
      </div>
    </div>
  );
};
