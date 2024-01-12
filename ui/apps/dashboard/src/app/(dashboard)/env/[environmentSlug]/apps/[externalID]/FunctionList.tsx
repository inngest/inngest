import { useRef } from 'react';
import Link from 'next/link';
import ArrowRightIcon from '@heroicons/react/20/solid/ArrowRightIcon';
import { Table } from '@inngest/components/Table';
import type { Function } from '@inngest/components/types/function';
import { createColumnHelper, getCoreRowModel } from '@tanstack/react-table';

import TriggerPill from '@/components/Pill/TriggerPill';

type Fn = Pick<Function, 'name' | 'slug' | 'triggers'>;

type Props = {
  envSlug: string;
  functions: Fn[];
};

export function FunctionList({ envSlug, functions }: Props) {
  const tableContainerRef = useRef<HTMLDivElement>(null);
  const columns = useColumns({ envSlug });

  const sortedFunctions = [...functions].sort((a, b) => {
    return a.name.localeCompare(b.name);
  });

  return (
    <main
      className="min-h-0 overflow-y-auto rounded-lg border border-slate-300"
      ref={tableContainerRef}
    >
      <Table
        blankState={<p>No functions.</p>}
        options={{
          columns,
          data: sortedFunctions,
          getCoreRowModel: getCoreRowModel(),
          enableSorting: false,
        }}
        tableContainerRef={tableContainerRef}
        isVirtualized={false}
      />
    </main>
  );
}

function useColumns({ envSlug }: { envSlug: string }) {
  const columnHelper = createColumnHelper<Fn>();

  return [
    columnHelper.accessor('name', {
      cell: (info) => {
        const name = info.getValue();

        return (
          <div className="flex items-center">
            <Link
              className="group flex w-full items-center gap-2 text-sm font-medium text-slate-700  hover:text-indigo-600"
              href={`/env/${envSlug}/functions/${encodeURIComponent(info.row.original.slug)}`}
            >
              {name}
              <ArrowRightIcon className="h-3 w-3 -translate-x-3 text-indigo-600 opacity-0 transition-all group-hover:translate-x-0 group-hover:opacity-100" />
            </Link>
          </div>
        );
      },
      header: 'Function Name',
    }),
    columnHelper.accessor('triggers', {
      cell: (props) => {
        const triggers = props.getValue();
        return (
          <div className="flex gap-1">
            {triggers.map((trigger) => {
              return (
                <TriggerPill
                  href={
                    trigger.type === 'EVENT'
                      ? `/env/${envSlug}/events/${encodeURIComponent(trigger.value)}`
                      : undefined
                  }
                  key={trigger.type + trigger.value}
                  trigger={{
                    type: trigger.type === 'EVENT' ? 'event' : 'schedule',
                    value: trigger.value,
                  }}
                />
              );
            })}
          </div>
        );
      },
      header: () => <span>Triggers</span>,
    }),
  ];
}
