import { useRef } from 'react';
import Link from 'next/link';
import { Table } from '@inngest/components/Table';
import { createColumnHelper, getCoreRowModel } from '@tanstack/react-table';

import { Time } from '@/components/Time';
import type { Event } from './types';

type Props = {
  environmentSlug: string;
  events: Event[];
};

export function EventTable({ environmentSlug, events }: Props) {
  const tableContainerRef = useRef<HTMLDivElement>(null);
  const columns = useColumns({ environmentSlug });

  return (
    <main className="min-h-0 overflow-y-auto" ref={tableContainerRef}>
      <Table
        blankState={<></>}
        options={{ columns, data: events, getCoreRowModel: getCoreRowModel() }}
        tableContainerRef={tableContainerRef}
      />
    </main>
  );
}

function useColumns({ environmentSlug }: { environmentSlug: string }) {
  const columnHelper = createColumnHelper<Event>();

  return [
    columnHelper.accessor('id', {
      cell: (props) => {
        const { id, name } = props.row.original;
        return (
          <Link
            href={`/env/${environmentSlug}/events/${encodeURIComponent(name)}/logs/${id}`}
            rel="noopener noreferrer"
            target="_blank"
          >
            {id}
          </Link>
        );
      },
      header: () => <span>ID</span>,
    }),
    columnHelper.accessor('name', {
      header: () => <span>Name</span>,
    }),
    columnHelper.accessor('receivedAt', {
      cell: (props) => {
        return <Time value={props.getValue()} />;
      },
      header: () => <span>Received At</span>,
    }),
  ];
}
