'use client';

import { useRef } from 'react';
import { NewButton } from '@inngest/components/Button';
import { Table } from '@inngest/components/Table';
import { Time } from '@inngest/components/Time';
import { createColumnHelper, getCoreRowModel } from '@tanstack/react-table';

import { usePagination } from './usePagination';

type Cancellation = {
  createdAt: string;
  id: string;
  queuedAtMax: string;
  queuedAtMin: string | null;
};

type Props = {
  envSlug: string;
  fnSlug: string;
};

export function CancellationTable({ envSlug, fnSlug }: Props) {
  const tableContainerRef = useRef<HTMLDivElement>(null);

  const {
    data: items,
    fetchNextPage,
    hasNextPage,
    isFetching,
    isInitiallyFetching,
  } = usePagination({ envSlug, fnSlug });

  let blankSlate = <p>No results</p>;
  if (isInitiallyFetching) {
    blankSlate = <p>Loading...</p>;
  }

  return (
    <div className="flex flex-col items-center">
      <div className="mb-8 self-stretch">
        <Table
          blankState={blankSlate}
          options={{
            columns,
            data: items,
            enableSorting: false,
            getCoreRowModel: getCoreRowModel(),
          }}
          tableContainerRef={tableContainerRef}
        />
      </div>

      {!isInitiallyFetching && (
        <span>
          <NewButton
            appearance="outlined"
            disabled={isFetching || !hasNextPage}
            label="Load More"
            onClick={() => fetchNextPage()}
          />
        </span>
      )}
    </div>
  );
}

const columnHelper = createColumnHelper<Cancellation>();

// TODO: Add column for cancellation deletion
const columns = [
  columnHelper.accessor('createdAt', {
    header: () => <span>Created at</span>,
    cell: (props) => {
      return <Time value={props.getValue()} />;
    },
  }),
  columnHelper.accessor('id', {
    header: () => <span>ID</span>,
    cell: (props) => {
      return <div className="flex items-center gap-2">{props.getValue()}</div>;
    },
  }),
  columnHelper.accessor('queuedAtMin', {
    header: () => <span>Minimum queued at</span>,
    cell: (props) => {
      const value = props.getValue();
      if (!value) {
        return <span>-</span>;
      }

      return <Time value={value} />;
    },
  }),
  columnHelper.accessor('queuedAtMax', {
    header: () => <span>Maximum queued at</span>,
    cell: (props) => {
      return <Time value={props.getValue()} />;
    },
  }),
];
