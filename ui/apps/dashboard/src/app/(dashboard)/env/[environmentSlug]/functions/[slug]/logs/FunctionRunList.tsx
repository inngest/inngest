'use client';

import { useCallback, useRef, useState } from 'react';
import { FunctionRunStatusIcon } from '@inngest/components/FunctionRunStatusIcon';
import { Table } from '@inngest/components/Table';
import { createColumnHelper, getCoreRowModel, type Row } from '@tanstack/react-table';
import { useQuery } from 'urql';

import { graphql } from '@/gql';
import { FunctionRunStatus, FunctionRunTimeField, type RunListItem } from '@/gql/graphql';
import { useEnvironment } from '@/queries';
import { type TimeRange } from './TimeRangeFilter';

const columnHelper = createColumnHelper<RunListItem>();

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

const columns = [
  columnHelper.accessor('id', {
    header: () => <span>ID</span>,
    cell: (props) => <p className="text-sm font-medium leading-7">{props.getValue()}</p>,
  }),
  columnHelper.accessor('status', {
    header: () => <span>Status</span>,
    cell: (props) => (
      <div className="flex items-center gap-2 lowercase">
        <FunctionRunStatusIcon status={props.getValue()} className="h-5 w-5" />
        <p className="first-letter:capitalize">{props.getValue()}</p>
      </div>
    ),
  }),
  columnHelper.accessor('startedAt', {
    header: () => <span>Scheduled At</span>,
    cell: (props) => <time>{props.getValue()}</time>,
  }),
  columnHelper.accessor('endedAt', {
    header: () => <span>Ended At</span>,
    cell: (props) => <time>{props.getValue()}</time>,
  }),
];

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
  const [functionRuns, setFunctionRuns] = useState<Array<RunListItem>>([]);

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

  const fetchMoreOnScroll = useCallback((containerRefElement?: HTMLDivElement | null) => {
    if (containerRefElement && runs?.length > 0) {
      const { scrollHeight, scrollTop, clientHeight } = containerRefElement;
      // Check if scrolled to the bottom
      const reachedBottom = scrollHeight - scrollTop - clientHeight < 50;

      if (reachedBottom && endCursor) {
        setPageCursors([...pageCursors, endCursor]);
      }
    }
  }, []);

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
    <div
      className="min-h-0 w-full overflow-y-auto pb-10"
      onScroll={(e) => fetchMoreOnScroll(e.target as HTMLDivElement)}
      ref={tableContainerRef}
    >
      <Table
        options={{
          data: runs ?? [],
          columns,
          getCoreRowModel: getCoreRowModel(),
          enableSorting: false,
          enablePinning: true,
          initialState: {
            columnPinning: {
              left: ['createdAt'],
            },
          },
          defaultColumn: {
            minSize: 0,
            size: Number.MAX_SAFE_INTEGER,
            maxSize: Number.MAX_SAFE_INTEGER,
          },
        }}
        tableContainerRef={tableContainerRef}
        blankState={<p>No function runs</p>}
      />
    </div>
  );
}
