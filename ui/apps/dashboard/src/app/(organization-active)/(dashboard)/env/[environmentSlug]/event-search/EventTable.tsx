import { useRef } from 'react';
import { Table } from '@inngest/components/Table';
import { Time } from '@inngest/components/Time';
import { RiArrowRightLine } from '@remixicon/react';
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
          <button
            className="transition-color group flex cursor-pointer items-center gap-1 text-indigo-400 underline decoration-transparent decoration-2 underline-offset-4 duration-300 hover:decoration-indigo-400"
            onClick={() => onSelect(id)}
          >
            {id}
            <RiArrowRightLine className="h-3 w-3 -translate-x-3 text-indigo-400 opacity-0 transition-all group-hover:translate-x-0 group-hover:opacity-100" />
          </button>
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
