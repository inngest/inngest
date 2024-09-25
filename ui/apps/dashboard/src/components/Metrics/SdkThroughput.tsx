import { Chart } from '@inngest/components/Chart/Chart';
import { Info } from '@inngest/components/Info/Info';
import { NewLink } from '@inngest/components/Link/Link';

import type { VolumeMetricsQuery } from '@/gql/graphql';
import type { EntityLookup } from './Dashboard';
import { getLineChartOptions, mapEntityLines } from './utils';

export type SdkThroughputMetricsType = VolumeMetricsQuery['workspace']['sdkThroughput']['metrics'];

export const SdkThroughput = ({
  workspace,
  entities,
}: Partial<VolumeMetricsQuery> & { entities: EntityLookup }) => {
  const metrics =
    workspace && mapEntityLines(workspace.sdkThroughput.metrics, entities, { opacity: 0.1 });

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
