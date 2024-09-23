import { Chart } from '@inngest/components/Chart/Chart';
import { Info } from '@inngest/components/Info/Info';
import { NewLink } from '@inngest/components/Link/Link';

import type { VolumeMetricsQuery } from '@/gql/graphql';
import type { FunctionLookup } from './Dashboard';
import { getLineChartOptions, mapFunctionLines } from './utils';

export type SdkThroughputMetricsType = VolumeMetricsQuery['workspace']['sdkThroughput']['metrics'];

export const SdkThroughput = ({
  workspace,
  functions,
}: Partial<VolumeMetricsQuery> & { functions: FunctionLookup }) => {
  const metrics =
    workspace && mapFunctionLines(workspace.sdkThroughput.metrics, functions, { opacity: 0.1 });

  return (
    <div className="bg-canvasBase border-subtle relative flex h-[300px] w-full flex-col overflow-x-hidden rounded-lg p-5 md:w-[75%]">
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
