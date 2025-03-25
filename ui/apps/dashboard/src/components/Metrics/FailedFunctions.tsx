import { Button } from '@inngest/components/Button';
import { Chart } from '@inngest/components/Chart/Chart';
import { Info } from '@inngest/components/Info/Info';
import { Link } from '@inngest/components/Link/Link';
import { RiArrowRightUpLine } from '@remixicon/react';

import { useEnvironment } from '@/components/Environments/environment-context';
import type { FunctionStatusMetricsQuery } from '@/gql/graphql';
import { pathCreator } from '@/utils/urls';
import type { EntityLookup } from './Dashboard';
import { FailedRate } from './FailedRate';
import { getLineChartOptions, mapEntityLines, sum } from './utils';

export type CompletedType = FunctionStatusMetricsQuery['workspace']['completed'];
export type CompletedMetricsType = FunctionStatusMetricsQuery['workspace']['completed']['metrics'];

const filter = ({ metrics }: CompletedType) =>
  metrics.filter(({ tagValue }) => tagValue === 'Failed');

const sort = (metrics: CompletedMetricsType) =>
  metrics.sort(({ data: data1 }, { data: data2 }) => sum(data2) - sum(data1));

const mapFailed = (
  { completed }: FunctionStatusMetricsQuery['workspace'],
  entities: EntityLookup
) => {
  const failed = sort(filter(completed));
  return mapEntityLines(failed, entities);
};

export const FailedFunctions = ({
  workspace,
  entities,
  functions,
}: {
  workspace?: FunctionStatusMetricsQuery['workspace'];
  entities: EntityLookup;
  functions: EntityLookup;
}) => {
  const env = useEnvironment();

  const metrics = workspace && mapFailed(workspace, entities);

  return (
    <div className="bg-canvasBase border-subtle overflowx-hidden relative flex h-[384px] w-full flex-col rounded-md border p-5">
      <div className="mb-2 flex flex-row items-center justify-between gap-x-2">
        <div className="text-subtle flex w-full flex-row items-center gap-x-2 text-lg">
          Failed Functions{' '}
          <Info
            text="Total number of failed runs in your environment, app or function."
            action={
              <Link
                arrowOnHover
                className="text-sm"
                href="https://www.inngest.com/docs/platform/monitor/observability-metrics#failed-functions"
                target="_new"
              >
                Learn more about Inngest functions.
              </Link>
            }
          />
        </div>
        <Button
          size="small"
          kind="secondary"
          appearance="outlined"
          icon={<RiArrowRightUpLine />}
          iconSide="left"
          label="View all"
          href={`${pathCreator.runs({ envSlug: env.slug })}?filterStatus=%5B"FAILED"%5D`}
        />
      </div>
      <div className="flex h-full flex-row items-center">
        <Chart
          option={metrics ? getLineChartOptions(metrics) : {}}
          className="h-[100%] w-full md:w-[75%]"
          group="metricsDashboard"
        />
        <FailedRate workspace={workspace} functions={functions} />
      </div>
    </div>
  );
};
