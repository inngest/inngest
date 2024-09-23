import React from 'react';
import { NewButton } from '@inngest/components/Button';
import { Chart } from '@inngest/components/Chart/Chart';
import { Info } from '@inngest/components/Info/Info';
import { NewLink } from '@inngest/components/Link/Link';
import { OptionalTooltip } from '@inngest/components/Tooltip/OptionalTooltip';
import { formatDistanceToNow } from '@inngest/components/utils/date';
import { RiArrowRightUpLine } from '@remixicon/react';

import { useEnvironment } from '@/components/Environments/environment-context';
import type { FunctionStatusMetricsQuery } from '@/gql/graphql';
import { pathCreator } from '@/utils/urls';
import type { FunctionLookup } from './Dashboard';
import { getLineChartOptions, mapFunctionLines, sum, type LineChartData } from './utils';

export type CompletedType = FunctionStatusMetricsQuery['workspace']['completed'];
export type CompletedMetricsType = FunctionStatusMetricsQuery['workspace']['completed']['metrics'];

export type Rate = {
  name: string;
  lastOccurence?: string;
  totalFailures: number;
  failureRate: number;
};

export type RateListData = {
  rateList: Rate[];
};

export type MappedData = RateListData & LineChartData;

const filter = ({ metrics }: CompletedType) =>
  metrics.filter(({ tagValue }) => tagValue === 'Failed');

const sort = (metrics: CompletedMetricsType) =>
  metrics.sort(({ data: data1 }, { data: data2 }) => sum(data2) - sum(data1));

//
// Completion metrics for this function are spread across this:
// [{id: x, data: {value}}, {id: x, data: {value}}]
// flatten & sum all the values
const getRate = (id: string, totalFailures: number, completed: CompletedType) => {
  const totalCompleted = sum(
    completed.metrics.filter((m) => m.id === id).flatMap(({ data }) => data)
  );

  return totalFailures && totalCompleted ? totalFailures / totalCompleted : 0;
};

const mapRateList = (
  failed: CompletedMetricsType,
  completed: CompletedType,
  functions: FunctionLookup
): Rate[] => {
  return failed.map((f) => {
    const failures = f.data.filter((d) => d.value > 0);
    const totalFailures = sum(failures);
    const lastOccurence = failures.at(-1)?.bucket;
    return {
      name: functions[f.id] || f.id,
      lastOccurence,
      totalFailures,
      failureRate: getRate(f.id, totalFailures, completed),
    };
  });
};

const mapFailed = (
  { completed }: FunctionStatusMetricsQuery['workspace'],
  functions: FunctionLookup
) => {
  const failed = sort(filter(completed));
  const rateList = mapRateList(failed, completed, functions);

  return {
    ...mapFunctionLines(failed, functions),
    rateList,
  };
};

export const FailedFunctions = ({
  workspace,
  functions,
}: Partial<FunctionStatusMetricsQuery> & { functions: FunctionLookup }) => {
  const env = useEnvironment();

  const metrics = workspace && mapFailed(workspace, functions);

  return (
    <div className="bg-canvasBase border-subtle overflowx-hidden relative flex h-[300px] w-full flex-col rounded-lg p-5">
      <div className="mb-2 flex flex-row items-center justify-between">
        <div className="text-subtle flex w-full flex-row items-center gap-x-2 text-lg">
          Failed Functions{' '}
          <Info
            text="Total number of failed runs in your environment, app or function."
            action={
              <NewLink
                arrowOnHover
                className="text-sm"
                href="https://www.inngest.com/docs/features/inngest-functions?ref=app-metrics"
              >
                Learn more about Inngest functions.
              </NewLink>
            }
          />
        </div>
        <NewButton
          size="small"
          kind="secondary"
          appearance="outlined"
          icon={<RiArrowRightUpLine />}
          iconSide="left"
          label="View all"
          href={pathCreator.functions({ envSlug: env.slug })}
        />
      </div>
      <div className="flex h-full flex-row items-center">
        <Chart option={metrics ? getLineChartOptions(metrics) : {}} className="h-[100%] w-[75%]" />
        <FailedList rateList={metrics?.rateList} />
      </div>
    </div>
  );
};

export const FailedList = ({ rateList }: { rateList: Rate[] | undefined }) => {
  return (
    <div className="border-subtle my-5 mb-5 ml-4 mt-8 flex h-full w-[25%] min-w-[220px] flex-col items-start justify-start border-l pl-4">
      <div className="pt flex w-full flex-row items-center justify-between gap-x-3 text-xs font-medium leading-none">
        <div>Recently failed</div>
        <div className="flex flex-row gap-x-3 justify-self-end">
          <div className="justify-self-end">Failed Runs</div>
          <div>Rate</div>
        </div>
      </div>
      {rateList?.map((r, i) => (
        <React.Fragment key={`function-failed-list-${i}`}>
          <div className="mt-3 flex w-full flex-row items-center justify-between gap-x-3 text-xs font-light leading-none">
            <div>{r.name}</div>
            <div className="flex flex-row justify-end gap-x-4">
              <div className="justify-self-end">{r.totalFailures}</div>
              <div className="text-tertiary-moderate">
                {r.failureRate.toLocaleString(undefined, {
                  style: 'percent',
                  minimumFractionDigits: 0,
                })}
              </div>
            </div>
          </div>

          <OptionalTooltip tooltip={r.lastOccurence}>
            <div
              className={`text-disabled leading none text-xs ${
                r.lastOccurence && 'cursor-pointer'
              }`}
            >
              {r.lastOccurence && formatDistanceToNow(r.lastOccurence, { addSuffix: true })}
            </div>
          </OptionalTooltip>
        </React.Fragment>
      ))}
    </div>
  );
};
