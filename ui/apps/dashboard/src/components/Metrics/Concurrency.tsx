import { Chart } from '@inngest/components/Chart/Chart';
import { Info } from '@inngest/components/Info/Info';
import { NewLink } from '@inngest/components/Link/Link';

import type { VolumeMetricsQuery } from '@/gql/graphql';
import type { EntityLookup } from './Dashboard';
import { NotFound } from './NotFound';
import { getLineChartOptions, mapEntityLines } from './utils';

export type SdkThroughputMetricsType = VolumeMetricsQuery['workspace']['concurrency']['metrics'];

export const AccountConcurrency = ({
  workspace,
  entities,
}: Partial<VolumeMetricsQuery> & { entities: EntityLookup }) => {
  const metrics =
    workspace && mapEntityLines(workspace.concurrency.metrics, entities, { opacity: 1 });

  const notFound = metrics && metrics.series.length === 0;
  if (notFound) {
    return <NotFound />;
  }

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
        <Chart option={metrics ? getLineChartOptions(metrics) : {}} className="h-full w-full" />
      </div>
    </div>
  );
};
