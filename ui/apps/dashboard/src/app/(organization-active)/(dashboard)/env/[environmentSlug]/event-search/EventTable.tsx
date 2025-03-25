import { useRef } from 'react';
import { Link } from '@inngest/components/Link';
import { Table } from '@inngest/components/Table';
import { Time } from '@inngest/components/Time';
import { createColumnHelper, getCoreRowModel } from '@tanstack/react-table';

import type { Event } from './types';

type Props = {
  events: Event[];
  onSelect: (eventID: string) => void;
  blankState: React.ReactNode;
};

export function EventTable({ events, onSelect, blankState }: Props) {
  const tableContainerRef = useRef<HTMLDivElement>(null);
  const columns = useColumns(onSelect);

  return (
    <main className="min-h-0 overflow-y-auto" ref={tableContainerRef}>
      <Table
        blankState={blankState}
        options={{ columns, data: events, getCoreRowModel: getCoreRowModel() }}
        tableContainerRef={tableContainerRef}
      />
    </main>
  );
}

function useColumns(onSelect: (eventID: string) => void) {
  const columnHelper = createColumnHelper<Event>();

  const columns = [
    columnHelper.accessor('id', {
      cell: (props) => {
        const { id } = props.row.original;
        return (
          <Link arrowOnHover onClick={() => onSelect(id)} href="">
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

  return columns;
}
