'use client';

import { useState } from 'react';
import type { Route } from 'next';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { Button } from '@inngest/components/Button';
import { useQuery } from 'urql';

import { Time } from '@/components/Time';
import { graphql } from '@/gql';
import { FunctionRunStatus, FunctionRunTimeField } from '@/gql/graphql';
import LoadingIcon from '@/icons/LoadingIcon';
import CancelledIcon from '@/icons/status-icons/cancelled.svg';
import CompletedIcon from '@/icons/status-icons/completed.svg';
import FailedIcon from '@/icons/status-icons/failed.svg';
import RunningIcon from '@/icons/status-icons/running.svg';
import { useEnvironment } from '@/queries';
import cn from '@/utils/cn';
import { type TimeRange } from './TimeRangeFilter';

const functionRunStatusIcons = {
  [FunctionRunStatus.Cancelled]: { icon: CancelledIcon, color: 'text-gray-500' },
  [FunctionRunStatus.Completed]: { icon: CompletedIcon, color: 'text-teal-500' },
  [FunctionRunStatus.Failed]: { icon: FailedIcon, color: 'text-red-500' },
  [FunctionRunStatus.Running]: { icon: RunningIcon, color: 'text-sky-500' },
} as const satisfies Record<FunctionRunStatus, { icon: SVGComponent; color: `text-${string}-500` }>;

const GetFunctionRunsDocument = graphql(`
  query GetFunctionRuns(
    $environmentID: ID!
    $functionSlug: String!
    $functionRunStatuses: [FunctionRunStatus!]
    $functionRunCursor: String
    $timeRangeStart: Time!
    $timeRangeEnd: Time!
    $timeField: FunctionRunTimeField!
  ) {
    environment: workspace(id: $environmentID) {
      function: workflowBySlug(slug: $functionSlug) {
        id
        runs: runsV2(
          filter: {
            status: $functionRunStatuses
            lowerTime: $timeRangeStart
            upperTime: $timeRangeEnd
            timeField: $timeField
          }
          first: 20
          after: $functionRunCursor
        ) {
          edges {
            node {
              id
              status
              startedAt
              endedAt
            }
          }
          pageInfo {
            hasNextPage
            endCursor
          }
        }
      }
    }
  }
`);

type FunctionRunListResultPageProps = {
  environmentSlug: string;
  functionSlug: string;
  selectedStatuses: FunctionRunStatus[];
  selectedTimeRange: TimeRange;
  timeField: FunctionRunTimeField;
  functionRunCursor: string;
  isLastDisplayedPage: boolean;
  onLoadMore: (nextCursor: string) => void;
};

function FunctionRunListResultPage({
  environmentSlug,
  functionSlug,
  selectedStatuses,
  selectedTimeRange,
  timeField,
  functionRunCursor,
  isLastDisplayedPage,
  onLoadMore,
}: FunctionRunListResultPageProps) {
  const [{ data: environment, fetching: isFetchingEnvironments }] = useEnvironment({
    environmentSlug,
  });

  const [{ data, fetching: isFetchingFunctionRuns }] = useQuery({
    query: GetFunctionRunsDocument,
    variables: {
      environmentID: environment?.id!,
      functionSlug,
      functionRunStatuses: selectedStatuses.length ? selectedStatuses : null,
      timeRangeStart: selectedTimeRange.start.toISOString(),
      timeRangeEnd: selectedTimeRange.end.toISOString(),
      timeField,
      functionRunCursor: functionRunCursor || null,
    },
    pause: !environment?.id,
  });

  const pathname = usePathname();

  if (isFetchingEnvironments || isFetchingFunctionRuns) {
    return (
      <div className="flex h-full w-full items-center justify-center">
        <LoadingIcon />
      </div>
    );
  }

  const function_ = data?.environment.function;

  if (!function_) {
    return (
      <div className="flex h-full w-full items-center justify-center">
        <h2 className="text-sm font-semibold text-gray-900">Function not found</h2>
      </div>
    );
  }

  const hasNextPage = function_.runs?.pageInfo.hasNextPage;
  const endCursor = function_.runs?.pageInfo.endCursor;
  const functionRuns = function_.runs?.edges?.map((edge) => edge?.node);

  if (!functionRuns || functionRuns.length === 0) {
    return (
      <div className="flex h-full w-full items-center justify-center">
        <h2 className="text-sm font-semibold text-gray-900">No function runs yet</h2>
      </div>
    );
  }

  return (
    <>
      {functionRuns?.map((functionRun, index) => {
        if (!functionRun) {
          return (
            <li key={index}>
              <div className="flex items-center gap-3 px-3 py-2.5 opacity-50">
                <FailedIcon className="h-6 w-6 shrink-0 text-red-500" />
                <div className="flex min-w-0 flex-col gap-1 text-ellipsis">
                  <p className="text-sm font-semibold text-slate-800">Error</p>
                  <p className="flex font-mono text-xs text-slate-500">
                    Could not load function run
                  </p>
                </div>
              </div>
            </li>
          );
        }
        const functionRunPathname = `/env/${environmentSlug}/functions/${encodeURIComponent(
          functionSlug
        )}/logs/${functionRun.id}`;
        const isActive = pathname === functionRunPathname;
        const StatusIcon = functionRunStatusIcons[functionRun.status].icon;

        let time: string;
        if (timeField === FunctionRunTimeField.EndedAt && functionRun.endedAt) {
          time = functionRun.endedAt;
        } else {
          time = functionRun.startedAt;
        }

        return (
          <li key={functionRun.id}>
            <Link
              href={functionRunPathname as Route}
              className={cn(
                'flex items-center gap-3 px-3 py-2.5 hover:bg-slate-100',
                isActive && 'bg-slate-100'
              )}
            >
              <StatusIcon
                className={cn(
                  functionRunStatusIcons[functionRun.status].color,
                  'h-6 w-6 shrink-0 text-teal-500'
                )}
              />
              <div className="flex min-w-0 flex-col gap-1 text-ellipsis">
                <Time
                  className="text-sm font-semibold text-slate-800"
                  format="relative"
                  value={new Date(time)}
                />

                <div className="flex items-center gap-2 font-mono text-xs text-slate-500">
                  {functionRun.id}
                </div>
              </div>
            </Link>
          </li>
        );
      })}
      {isLastDisplayedPage && hasNextPage && (
        <div className="flex justify-center py-2.5">
          <Button
            appearance="outlined"
            btnAction={() => onLoadMore(endCursor ?? '')}
            label="Load More"
          />
        </div>
      )}
    </>
  );
}

type FunctionRunListProps = {
  environmentSlug: string;
  functionSlug: string;
  selectedStatuses: FunctionRunStatus[];
  selectedTimeRange: TimeRange;
  timeField: FunctionRunTimeField;
};

export default function FunctionRunList({
  environmentSlug,
  functionSlug,
  selectedStatuses,
  selectedTimeRange,
  timeField,
}: FunctionRunListProps) {
  const [pageCursors, setPageCursors] = useState<string[]>(['']);

  // We reset the page cursors when the selected statuses or time range change, which resets the list to the first page.
  const [prevSelectedStatuses, setPrevSelectedStatuses] = useState(selectedStatuses);
  const [prevSelectedTimeRange, setPrevSelectedTimeRange] = useState(selectedTimeRange);
  if (selectedStatuses !== prevSelectedStatuses || selectedTimeRange !== prevSelectedTimeRange) {
    setPrevSelectedStatuses(selectedStatuses);
    setPrevSelectedTimeRange(selectedTimeRange);
    setPageCursors(['']);
  }

  return (
    <ul role="list" className="h-full divide-y divide-slate-100">
      {pageCursors.map((pageCursor, index) => (
        <FunctionRunListResultPage
          key={pageCursor}
          environmentSlug={environmentSlug}
          functionSlug={functionSlug}
          selectedStatuses={selectedStatuses}
          functionRunCursor={pageCursor}
          selectedTimeRange={selectedTimeRange}
          timeField={timeField}
          isLastDisplayedPage={index === pageCursors.length - 1}
          onLoadMore={(nextCursor) => setPageCursors([...pageCursors, nextCursor])}
        />
      ))}
    </ul>
  );
}
