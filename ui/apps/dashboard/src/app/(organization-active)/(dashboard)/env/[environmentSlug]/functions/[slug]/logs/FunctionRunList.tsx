'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import { type Route } from 'next';
import { Button } from '@inngest/components/Button';
import { FunctionRunStatusIcon } from '@inngest/components/FunctionRunStatusIcon';
import { Link } from '@inngest/components/Link';
import { Table } from '@inngest/components/Table';
import { createColumnHelper, getCoreRowModel } from '@tanstack/react-table';
import { useQuery } from 'urql';

import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
import { Time } from '@/components/Time';
import { graphql } from '@/gql';
import { FunctionRunStatus, FunctionRunTimeField, type RunListItem } from '@/gql/graphql';
import { type TimeRange } from './TimeRangeFilter';

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
function createColumns({
  environmentSlug,
  functionSlug,
}: {
  environmentSlug: string;
  functionSlug: string;
}) {
  const columnHelper = createColumnHelper<RunListItem>();

  return [
    columnHelper.accessor('id', {
      header: () => <span>ID</span>,
      cell: (props) => {
        return (
          <Link
            className="text-sm font-medium leading-7"
            internalNavigation
            //@ts-ignore
            scroll={false}
            href={
              `/env/${environmentSlug}/functions/${encodeURIComponent(
                functionSlug
              )}/logs/${props.getValue()}` as Route
            }
          >
            {props.getValue()}
          </Link>
        );
      },
    }),
    columnHelper.accessor('status', {
      header: () => <span>Status</span>,
      cell: (props) => (
        <div className="flex items-center gap-2 lowercase">
          <FunctionRunStatusIcon status={props.getValue()} className="h-5 w-5" />
          <p className="first-letter:capitalize">{props.getValue()}</p>
        </div>
      ),
      size: 300,
      minSize: 300,
    }),
    columnHelper.accessor('startedAt', {
      header: () => <span>Queued At</span>,
      cell: (props) => <Time value={new Date(props.getValue())} />,
      size: 300,
      minSize: 300,
    }),
    columnHelper.accessor('endedAt', {
      header: () => <span>Ended At</span>,
      cell: (props) => {
        const endedAt = props.getValue();
        return <>{endedAt ? <Time value={new Date(endedAt)} /> : <p>-</p>}</>;
      },
      size: 300,
      minSize: 300,
    }),
  ];
}

type FunctionRunListProps = {
  functionSlug: string;
  selectedStatuses: FunctionRunStatus[];
  selectedTimeRange: TimeRange;
  timeField: FunctionRunTimeField;
};

export default function FunctionRunList({
  functionSlug,
  selectedStatuses,
  selectedTimeRange,
  timeField,
}: FunctionRunListProps) {
  const env = useEnvironment();

  const columns = useMemo(() => {
    return createColumns({ environmentSlug: env.slug, functionSlug });
  }, [env.slug, functionSlug]);

  const [pageCursors, setPageCursors] = useState<string[]>(['']);
  const [aggregatedFunctionRuns, setAggregatedFunctionRuns] = useState<RunListItem[]>([]);

  const tableContainerRef = useRef<HTMLDivElement>(null);
  // We reset the page cursors when the selected statuses or time range change, which resets the list to the first page.
  const [prevSelectedStatuses, setPrevSelectedStatuses] = useState(selectedStatuses);
  const [prevSelectedTimeRange, setPrevSelectedTimeRange] = useState(selectedTimeRange);
  const [prevSelectedTimeField, setPrevSelectedTimeField] = useState(timeField);

  const environment = useEnvironment();

  const [{ data, fetching }] = useQuery({
    query: GetFunctionRunsDocument,
    variables: {
      environmentID: environment.id,
      functionSlug,
      functionRunStatuses: selectedStatuses.length ? selectedStatuses : null,
      timeRangeStart: selectedTimeRange.start.toISOString(),
      timeRangeEnd: selectedTimeRange.end.toISOString(),
      timeField,
      functionRunCursor: pageCursors[pageCursors.length - 1] || null,
    },
  });

  const runs = data?.environment.function?.runs?.edges?.map((edge) => edge?.node) ?? [];
  const endCursor = data?.environment.function?.runs?.pageInfo.endCursor;
  const hasNextPage = data?.environment.function?.runs?.pageInfo.hasNextPage;
  const isLoading = fetching || (runs.length > 0 && aggregatedFunctionRuns.length === 0);

  useEffect(() => {
    if (
      selectedStatuses !== prevSelectedStatuses ||
      selectedTimeRange !== prevSelectedTimeRange ||
      timeField !== prevSelectedTimeField
    ) {
      setPrevSelectedStatuses(selectedStatuses);
      setPrevSelectedTimeRange(selectedTimeRange);
      setPrevSelectedTimeField(timeField);
      setPageCursors(['']);
      setAggregatedFunctionRuns([]);
    } else {
      setAggregatedFunctionRuns((prevFunctionRuns) => {
        const updatedFunctionRuns = prevFunctionRuns.map((prevRun) => {
          const matchingRun = runs.find((run) => run?.id === prevRun.id);
          return matchingRun ?? prevRun;
        });
        return [
          ...updatedFunctionRuns,
          ...runs.filter((run) => !prevFunctionRuns.some((prevRun) => prevRun.id === run?.id)),
        ].filter(Boolean) as RunListItem[];
      });
    }
  }, [data, selectedStatuses, selectedTimeRange, timeField]);

  return (
    <div className="min-h-0 w-full overflow-y-auto pb-10" ref={tableContainerRef}>
      <Table
        options={{
          data: aggregatedFunctionRuns,
          columns,
          getCoreRowModel: getCoreRowModel(),
          enableSorting: false,
          enablePinning: false,
          state: {
            columnOrder:
              timeField === FunctionRunTimeField.StartedAt
                ? ['id', 'status', 'startedAt', 'endedAt']
                : ['id', 'status', 'endedAt', 'startedAt'],
          },
          defaultColumn: {
            minSize: 0,
            size: Number.MAX_SAFE_INTEGER,
            maxSize: Number.MAX_SAFE_INTEGER,
          },
        }}
        tableContainerRef={tableContainerRef}
        blankState={isLoading ? <p>Loading...</p> : <p>No function runs</p>}
      />
      {hasNextPage && aggregatedFunctionRuns.length > 0 && (
        <div className="flex justify-center pt-4">
          <Button
            label="Load More"
            appearance="outlined"
            loading={fetching}
            btnAction={() => endCursor && setPageCursors([...pageCursors, endCursor])}
          />
        </div>
      )}
    </div>
  );
}
