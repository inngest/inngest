import { useRef } from 'react';
import Link from 'next/link';
import { Table } from '@inngest/components/Table';
import { createColumnHelper, getCoreRowModel } from '@tanstack/react-table';

import { Time } from '@/components/Time';
import type { Event } from './types';

type Props = {
  events: Event[];
  onSelect: (eventID: string) => void;
};

export function EventTable({ events, onSelect }: Props) {
  const tableContainerRef = useRef<HTMLDivElement>(null);
  const columns = useColumns(onSelect);

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

function useColumns(onSelect: (eventID: string) => void) {
  const columnHelper = createColumnHelper<Event>();

  return [
    columnHelper.accessor('id', {
      cell: (props) => {
        const { id, name } = props.row.original;
        return <button onClick={() => onSelect(id)}>{id}</button>;
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
