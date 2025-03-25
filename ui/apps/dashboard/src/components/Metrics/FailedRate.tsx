import React from 'react';
import { Link } from '@inngest/components/Link/Link';
import { OptionalTooltip } from '@inngest/components/Tooltip/OptionalTooltip';
import { formatDistanceToNow } from '@inngest/components/utils/date';

import type { FunctionStatusMetricsQuery } from '@/gql/graphql';
import { pathCreator } from '@/utils/urls';
import { useEnvironment } from '../Environments/environment-context';
import type { EntityLookup } from './Dashboard';
import { sum } from './utils';

export type CompletedByFunctionType =
  FunctionStatusMetricsQuery['workspace']['completedByFunction'];
export type CompletedByFunctionMetricsType =
  FunctionStatusMetricsQuery['workspace']['completedByFunction']['metrics'];

export type Rate = {
  slug: string;
  name: string;
  lastOccurence?: string;
  totalFailures: number;
  failureRate: number;
};

export type RateListData = {
  rateList: Rate[];
};

const filter = ({ metrics }: CompletedByFunctionType) =>
  metrics.filter(({ tagValue }) => tagValue === 'Failed');

const sort = (metrics: CompletedByFunctionMetricsType) =>
  metrics.sort(({ data: data1 }, { data: data2 }) => sum(data2) - sum(data1));

//
// Completion metrics for this function are spread across this:
// [{id: x, data: {value}}, {id: x, data: {value}}]
// flatten & sum all the values
const getRate = (id: string, totalFailures: number, completed: CompletedByFunctionType) => {
  const totalCompleted = sum(
    completed.metrics.filter((m) => m.id === id).flatMap(({ data }) => data)
  );

  return totalFailures && totalCompleted ? totalFailures / totalCompleted : 0;
};

const mapRateList = (
  failed: CompletedByFunctionMetricsType,
  completed: CompletedByFunctionType,
  functions: EntityLookup
): Rate[] => {
  return failed.slice(0, 6).map((f) => {
    const failures = f.data.filter((d) => d.value > 0);
    const totalFailures = sum(failures);
    const lastOccurence = failures.at(-1)?.bucket;
    return {
      slug: functions[f.id]?.slug || '',
      name: functions[f.id]?.name || f.id,
      lastOccurence,
      totalFailures,
      failureRate: getRate(f.id, totalFailures, completed),
    };
  });
};

export const FailedRate = ({
  workspace,
  functions,
}: {
  workspace?: FunctionStatusMetricsQuery['workspace'];
  functions: EntityLookup;
}) => {
  const env = useEnvironment();
  const failed = workspace && sort(filter(workspace.completedByFunction));
  const rateList = failed && mapRateList(failed, workspace.completedByFunction, functions);

  return (
    <div className="border-subtle my-5 mb-5 ml-4 mt-8 hidden h-full w-[25%] min-w-[220px] flex-col items-start justify-start border-l pl-4 md:flex">
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
            <Link
              className="text-basis text-xs font-light leading-none hover:no-underline"
              href={`${pathCreator.function({ envSlug: env.slug, functionSlug: r.slug })}/runs`}
            >
              <div className="w-[136px] overflow-hidden text-ellipsis text-nowrap">{r.name}</div>
            </Link>
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
              className={`text-disabled text-xs leading-none ${
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
