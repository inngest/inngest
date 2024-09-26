import { Chart } from '@inngest/components/Chart/Chart';
import { Info } from '@inngest/components/Info/Info';
import { NewLink } from '@inngest/components/Link/Link';

import type { VolumeMetricsQuery } from '@/gql/graphql';
import type { EntityLookup } from './Dashboard';
import { getLineChartOptions, mapEntityLines } from './utils';

export const StepsThroughput = ({
  workspace,
  entities,
}: Partial<VolumeMetricsQuery> & { entities: EntityLookup }) => {
  const metrics = workspace && mapEntityLines(workspace.stepThroughput.metrics, entities);

  return (
    <div className="bg-canvasBase border-subtle relative flex h-[384px] w-full flex-col overflow-x-hidden rounded-lg border p-5">
      <div className="mb-2 flex flex-row items-center justify-between">
        <div className="text-subtle flex w-full flex-row items-center gap-x-2 text-lg">
          Total steps throughput{' '}
          <Info
            text="Total number of steps processed your env, app or function."
            action={
              <NewLink
                arrowOnHover
                className="text-sm"
                href="https://www.inngest.com/docs/features/inngest-functions/steps-workflows"
              >
                Learn more about step throughput.
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
