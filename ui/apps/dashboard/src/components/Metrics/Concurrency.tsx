import { Chart } from '@inngest/components/Chart/Chart';
import { Info } from '@inngest/components/Info/Info';
import { NewLink } from '@inngest/components/Link/Link';

import type { VolumeMetricsQuery } from '@/gql/graphql';
import type { FunctionLookup } from './Dashboard';
import { getLineChartOptions, mapFunctionLines } from './utils';

export type SdkThroughputMetricsType = VolumeMetricsQuery['workspace']['concurrency']['metrics'];

export const AccountConcurrency = ({
  workspace,
  functions,
}: Partial<VolumeMetricsQuery> & { functions: FunctionLookup }) => {
  const metrics =
    workspace && mapFunctionLines(workspace.concurrency.metrics, functions, { opacity: 1 });

  return (
    <div className="bg-canvasBase border-subtle relative flex h-[300px] w-full flex-col overflow-x-hidden rounded-lg p-5">
      <div className="mb-2 flex flex-row items-center justify-between">
        <div className="text-subtle flex w-full flex-row items-center gap-x-2 text-lg">
          Account Concurrency{' '}
          <Info
            text="Total number of compared to the account-level concurrency limits."
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
        <Chart option={metrics ? getLineChartOptions(metrics) : {}} className="h-full w-full" />
      </div>
    </div>
  );
};
