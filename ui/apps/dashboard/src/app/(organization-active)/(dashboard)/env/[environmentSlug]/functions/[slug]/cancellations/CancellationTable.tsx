'use client';

import { useRef } from 'react';
import { Table } from '@inngest/components/Table';
import { Time } from '@inngest/components/Time';
import { createColumnHelper, getCoreRowModel } from '@tanstack/react-table';

type Cancellation = {
  createdAt: string;
  id: string;
  queuedAtMax: string;
  queuedAtMin: string | null;
};

type Props = {
  data: Cancellation[];
};

export function CancellationTable({ data }: Props) {
  const tableContainerRef = useRef<HTMLDivElement>(null);

  return (
    <Table
      blankState={<p>No results</p>}
      options={{
        columns,
        data,
        enableSorting: false,
        getCoreRowModel: getCoreRowModel(),
      }}
      tableContainerRef={tableContainerRef}
    />
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
    header: () => <span>Queued at min</span>,
    cell: (props) => {
      const value = props.getValue();
      if (!value) {
        return <span>-</span>;
      }

      return <Time value={value} />;
    },
  }),
  columnHelper.accessor('queuedAtMax', {
    header: () => <span>Queued at max</span>,
    cell: (props) => {
      return <Time value={props.getValue()} />;
    },
  }),
];
