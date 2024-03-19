'use client';

import React, { useRef } from 'react';
import { ExclamationCircleIcon } from '@heroicons/react/20/solid';
import { Link } from '@inngest/components/Link';
import { Table } from '@inngest/components/Table';
import { createColumnHelper, getCoreRowModel } from '@tanstack/react-table';

import { Time } from '@/components/Time';
import { graphql } from '@/gql';
import LoadingIcon from '@/icons/LoadingIcon';
import { useGraphQLQuery } from '@/utils/useGraphQLQuery';

const GetCancellationsDocument = graphql(`
  query GetCancellations($envID: ID!, $fnID: ID!) {
    environment: workspace(id: $envID) {
      function: workflow(id: $fnID) {
        cancellations {
          createdAt
          expression
          id
          name
          queuedAtMax
          queuedAtMin
        }
      }
    }
  }
`);

const columnHelper = createColumnHelper<{
  createdAt: Date;
  expression: string | null;
  name: string | null;
  queuedAtMax: Date;
  queuedAtMin: Date | null;
}>();

const columns = [
  columnHelper.accessor('name', {
    header: () => <span>Replay Name</span>,
    cell: (props) => {
      return props.getValue();
    },
  }),
  columnHelper.accessor('createdAt', {
    header: () => <span>Created At</span>,
    cell: (props) => {
      return <Time value={props.getValue()} />;
    },
  }),
  columnHelper.accessor('queuedAtMin', {
    header: () => <span>Queued At Minimum</span>,
    cell: (props) => {
      const value = props.getValue();
      if (!value) {
        return null;
      }

      return <Time value={value} />;
    },
  }),
  columnHelper.accessor('queuedAtMax', {
    header: () => <span>Queued At Maximum</span>,
    cell: (props) => {
      return <Time value={props.getValue()} />;
    },
  }),
  columnHelper.accessor('expression', {
    header: () => <span>Expression</span>,
    cell: (props) => {
      return props.getValue();
    },
  }),
];

type Props = {
  envID: string;
  fnID: string;
};

export function CancellationList({ envID, fnID }: Props) {
  const { data, isLoading, error } = useGraphQLQuery({
    query: GetCancellationsDocument,
    variables: {
      envID,
      fnID,
    },
    pollIntervalInMilliseconds: 5_000,
  });

  const tableContainerRef = useRef<HTMLDivElement>(null);

  if (isLoading && !data) {
    return (
      <div className="flex h-full w-full items-center justify-center">
        <LoadingIcon />
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex h-full w-full flex-col items-center justify-center gap-5">
        <div className="inline-flex items-center gap-2 text-red-600">
          <ExclamationCircleIcon className="h-4 w-4" />
          <h2 className="text-sm">Could not load replays</h2>
        </div>
      </div>
    );
  }

  const cancellations = data.environment.function?.cancellations?.map((c) => {
    return {
      ...c,
      createdAt: new Date(c.createdAt),
      queuedAtMax: new Date(c.queuedAtMax),
      queuedAtMin: c.queuedAtMin ? new Date(c.queuedAtMin) : null,
    };
  });
  if (!cancellations) {
    throw new Error('Cancellations not found');
  }

  return (
    <Table
      tableContainerRef={tableContainerRef}
      options={{
        data: cancellations,
        columns,
        getCoreRowModel: getCoreRowModel(),
        enableSorting: false,
      }}
      blankState={
        <p>
          You have no replays for this function.{' '}
          <Link className="inline-flex" href="https://inngest.com/docs/platform/replay">
            Learn about Replay
          </Link>
        </p>
      }
    />
  );
}
