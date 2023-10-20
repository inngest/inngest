import { useMemo } from 'react';
import Link from 'next/link';
import { ArrowRightIcon, ChartBarIcon, ExclamationCircleIcon } from '@heroicons/react/20/solid';
import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  useReactTable,
} from '@tanstack/react-table';

import MiniStackedBarChart from '@/components/Charts/MiniStackedBarChart';
import { Pill } from '@/components/Pill/Pill';
import TriggerPill, { TRIGGER_TYPE, type Trigger } from '@/components/Pill/TriggerPill';
import Placeholder from '@/components/Placeholder';
import cn from '@/utils/cn';

export type FunctionTableRow = {
  appName: string | null;
  name: string;
  isArchived: boolean;
  isActive: boolean;
  slug: string;
  triggers: Trigger[];
  failureRate: number | undefined;
  usage:
    | {
        total: number;
        slots: { failureCount: number; startCount: number }[];
      }
    | undefined;
};

type Props = {
  environmentSlug: string;
  rows: FunctionTableRow[] | undefined;
};

export function FunctionTable({ environmentSlug, rows = [] }: Props) {
  const columns = useMemo(() => {
    return createColumns(environmentSlug);
  }, [environmentSlug]);

  const table = useReactTable({
    data: rows,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getRowId: (row) => row.slug,
  });

  return (
    <main className="flex min-h-0 flex-col overflow-y-auto bg-slate-100">
      <table className="border-b border-slate-200 bg-white">
        <thead className="shadow-outline-primary-light sticky top-0 z-10 bg-white">
          {table.getHeaderGroups().map((headerGroup) => (
            <tr key={headerGroup.id}>
              {headerGroup.headers.map((header, index) => (
                <th
                  key={header.id}
                  className={cn(
                    'w-fit whitespace-nowrap px-2 py-3 text-left text-xs font-semibold text-slate-600',
                    index === 0 && 'pl-6',
                    index === columns.length - 1 && 'w-0'
                  )}
                >
                  {header.isPlaceholder
                    ? null
                    : flexRender(header.column.columnDef.header, header.getContext())}
                </th>
              ))}
            </tr>
          ))}
        </thead>
        <tbody className="divide-y divide-slate-100">
          {table.getRowModel().rows.map((row) => (
            <tr key={row.id} className="hover:bg-slate-100">
              {row.getVisibleCells().map((cell, index) => (
                <td
                  key={cell.id}
                  className={cn('whitespace-nowrap', index === columns.length - 1 && 'pr-6')}
                >
                  {flexRender(cell.column.columnDef.cell, cell.getContext())}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>

      {rows.length === 0 && (
        <div className="p-3 text-center text-sm text-slate-700">No functions</div>
      )}
    </main>
  );
}

function Shimmer() {
  return (
    <div className="flex">
      <Placeholder className="mx-4 my-4 h-2.5 w-full bg-slate-200" />
    </div>
  );
}

const columnHelper = createColumnHelper<FunctionTableRow>();

function createColumns(environmentSlug: string) {
  const columns = [
    columnHelper.accessor('name', {
      cell: (info) => {
        const name = info.getValue();
        const { isActive, isArchived, slug } = info.row.original;

        return (
          <div className="flex items-center pl-6">
            <div
              className={cn(
                'h-2.5 w-2.5 rounded-full',
                isArchived ? 'bg-slate-300' : isActive ? 'bg-teal-500' : 'bg-amber-500'
              )}
            />
            <Link
              key="name"
              href={`/env/${environmentSlug}/functions/${encodeURIComponent(slug)}`}
              className="group flex w-full items-center gap-2 px-2 py-3 text-sm font-medium text-slate-700  hover:text-indigo-600"
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
      cell: (info) => {
        return info.getValue().map((trigger) => {
          return (
            <TriggerPill
              key={trigger.value}
              href={
                trigger.type === TRIGGER_TYPE.event
                  ? `/env/${environmentSlug}/events/${encodeURIComponent(trigger.value)}`
                  : undefined
              }
              trigger={trigger}
            />
          );
        });
      },
      header: 'Triggers',
    }),
    columnHelper.accessor('appName', {
      cell: (info) => {
        return (
          <div className="px-2 py-3 text-sm font-medium text-slate-700">{info.getValue()}</div>
        );
      },
      header: 'App',
    }),
    columnHelper.accessor('failureRate', {
      cell: (info) => {
        const value = info.getValue();
        if (value === undefined) {
          return <Shimmer />;
        }

        let icon;
        if (value > 0) {
          icon = <ExclamationCircleIcon className="-ml-1 mr-1 h-4 w-4 text-red-600" />;
        }

        return (
          <Pill className="bg-white px-2.5 text-slate-600">
            {icon}
            {value}%
          </Pill>
        );
      },
      header: 'Failure Rate (24hr)',
    }),
    columnHelper.accessor('usage', {
      cell: (info) => {
        const value = info.getValue();
        if (value === undefined) {
          return <Shimmer />;
        }

        return (
          <div className="flex items-center justify-end gap-2">
            <span
              key="volume-count"
              className="overflow-hidden whitespace-nowrap text-xs text-slate-600"
            >
              <Pill className="gap-1 bg-white align-middle text-slate-600">
                <ChartBarIcon className="-ml-0.5 h-3.5 w-3.5 shrink-0 text-indigo-500" />
                {value.total.toLocaleString(undefined, {
                  notation: 'compact',
                  compactDisplay: 'short',
                })}
              </Pill>
            </span>

            <MiniStackedBarChart key="volume-chart" className="shrink-0" data={value.slots} />
          </div>
        );
      },
      header: 'Volume (24hr)',
    }),
  ];

  return columns;
}
