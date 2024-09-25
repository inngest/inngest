import { Chart } from '@inngest/components/Chart/Chart';
import { Info } from '@inngest/components/Info/Info';
import { NewLink } from '@inngest/components/Link/Link';

import type { VolumeMetricsQuery } from '@/gql/graphql';
import type { EntityLookup } from './Dashboard';
import { getLineChartOptions, mapEntityLines } from './utils';

export type RunsThroughputMetricsType =
  VolumeMetricsQuery['workspace']['runsThroughput']['metrics'];

export const RunsThrougput = ({
  workspace,
  entities,
}: Partial<VolumeMetricsQuery> & { entities: EntityLookup }) => {
  const metrics = workspace && mapEntityLines(workspace.runsThroughput.metrics, entities);

  return (
    <div className="bg-canvasBase border-subtle relative flex h-[300px] w-full flex-col overflow-x-hidden rounded-lg p-5">
      <div className="mb-2 flex flex-row items-center justify-between">
        <div className="text-subtle flex w-full flex-row items-center gap-x-2 text-lg">
          Total runs throughput{' '}
          <Info
            text="Total number of runs processed your env, app or function."
            action={
              <NewLink
                arrowOnHover
                className="text-sm"
                href="https://www.inngest.com/docs/features/inngest-functions/steps-workflows"
              >
                Learn more about runs throughput.
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
