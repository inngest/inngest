'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import { Button } from '@inngest/components/Button';
import { FunctionRunStatusIcon } from '@inngest/components/FunctionRunStatusIcon';
import { Link } from '@inngest/components/Link';
import { Table } from '@inngest/components/Table';
import { createColumnHelper, getCoreRowModel, type ColumnOrderState } from '@tanstack/react-table';
import { useQuery } from 'urql';

import { graphql } from '@/gql';
import { FunctionRunStatus, FunctionRunTimeField, type RunListItem } from '@/gql/graphql';
import { useEnvironment } from '@/queries';
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
            href={`/env/${environmentSlug}/functions/${encodeURIComponent(
              functionSlug
            )}/logs/${props.getValue()}`}
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
      header: () => <span>Scheduled At</span>,
      cell: (props) => <time>{props.getValue()}</time>,
      size: 300,
      minSize: 300,
    }),
    columnHelper.accessor('endedAt', {
      header: () => <span>Ended At</span>,
      cell: (props) => <time>{props.getValue()}</time>,
      size: 300,
      minSize: 300,
    }),
  ];
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
  const columns = useMemo(() => {
    return createColumns({ environmentSlug, functionSlug });
  }, [environmentSlug, functionSlug]);

  const [pageCursors, setPageCursors] = useState<string[]>(['']);
  const [functionRuns, setFunctionRuns] = useState<RunListItem[]>([]);
  const [prevSelectedTimeField, setPrevSelectedTimeField] = useState(timeField);

  const tableContainerRef = useRef<HTMLDivElement>(null);
  // We reset the page cursors when the selected statuses or time range change, which resets the list to the first page.
  const [prevSelectedStatuses, setPrevSelectedStatuses] = useState(selectedStatuses);
  const [prevSelectedTimeRange, setPrevSelectedTimeRange] = useState(selectedTimeRange);
  if (selectedStatuses !== prevSelectedStatuses || selectedTimeRange !== prevSelectedTimeRange) {
    setPrevSelectedStatuses(selectedStatuses);
    setPrevSelectedTimeRange(selectedTimeRange);
    setPageCursors(['']);
    setFunctionRuns([]);
  }

  const [{ data: environment, fetching: isFetchingEnvironments }] = useEnvironment({
    environmentSlug,
  });

  const [{ data, fetching }] = useQuery({
    query: GetFunctionRunsDocument,
    variables: {
      environmentID: environment?.id!,
      functionSlug,
      functionRunStatuses: selectedStatuses.length ? selectedStatuses : null,
      timeRangeStart: selectedTimeRange.start.toISOString(),
      timeRangeEnd: selectedTimeRange.end.toISOString(),
      timeField,
      functionRunCursor: pageCursors[pageCursors.length - 1] || null,
    },
    pause: !environment?.id,
  });

  const runs = data?.environment?.function?.runs?.edges?.map((edge) => edge?.node) ?? [];
  const endCursor = data?.environment?.function?.runs?.pageInfo.endCursor;
  const hasNextPage = data?.environment?.function?.runs?.pageInfo.hasNextPage;
  const isLoading = isFetchingEnvironments || fetching;

  useEffect(() => {
    if (timeField !== prevSelectedTimeField) {
      const validRuns = runs ? (runs.filter(Boolean) as RunListItem[]) : [];
      setFunctionRuns(validRuns);
      setPrevSelectedTimeField(timeField);
    } else {
      setFunctionRuns(
        (prevFunctionRuns) => [...prevFunctionRuns, ...runs].filter(Boolean) as RunListItem[]
      );
    }
  }, [data]);

  if (
    !data ||
    !data?.environment ||
    !data?.environment?.function ||
    !data?.environment?.function?.runs ||
    !functionRuns
  ) {
    return;
  }

  return (
    <div className="min-h-0 w-full overflow-y-auto pb-10" ref={tableContainerRef}>
      <Table
        options={{
          data: functionRuns ?? [],
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
      {hasNextPage && (
        <div className="flex justify-center pt-4">
          <Button
            label="Load More"
            appearance="outlined"
            loading={fetching}
            btnAction={() =>
              pageCursors && endCursor && setPageCursors([...pageCursors, endCursor])
            }
          />
        </div>
      )}
    </div>
  );
}
