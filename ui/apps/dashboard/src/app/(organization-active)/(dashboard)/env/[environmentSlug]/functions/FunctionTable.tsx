'use client';

import { useMemo } from 'react';
import { type Route } from 'next';
import { Link } from '@inngest/components/Link';
import { Pill, PillContent } from '@inngest/components/Pill';
import { type Trigger } from '@inngest/components/types/trigger';
import { RiBarChart2Fill, RiErrorWarningLine } from '@remixicon/react';
import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  useReactTable,
} from '@tanstack/react-table';

import { useEnvironment } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/environment-context';
import MiniStackedBarChart from '@/components/Charts/MiniStackedBarChart';
import Placeholder from '@/components/Placeholder';
import cn from '@/utils/cn';

export type FunctionTableRow = {
  appName: string | null;
  name: string;
  isArchived: boolean;
  isActive: boolean;
  isPaused: boolean;
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
  rows: FunctionTableRow[] | undefined;
};

export function FunctionTable({ rows = [] }: Props) {
  const env = useEnvironment();

  const columns = useMemo(() => {
    return createColumns(env.slug);
  }, [env.slug]);

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

function Shimmer({ className }: { className?: string }) {
  return (
    <div className={`flex ${className}`}>
      <Placeholder className="my-4 h-2.5 w-full bg-slate-200" />
    </div>
  );
}

const columnHelper = createColumnHelper<FunctionTableRow>();

function createColumns(environmentSlug: string) {
  const columns = [
    columnHelper.accessor('name', {
      cell: (info) => {
        const name = info.getValue();
        const { isPaused, isArchived, slug } = info.row.original;

        return (
          <div className="flex items-center pl-6">
            <div
              className={cn(
                'h-2.5 w-2.5 rounded-full',
                isArchived ? 'bg-slate-300' : isPaused ? 'bg-amber-500' : 'bg-teal-500'
              )}
            />
            <Link
              key="name"
              href={`/env/${environmentSlug}/functions/${encodeURIComponent(slug)}` as Route}
              internalNavigation
              className="w-full px-2 py-3 text-sm font-medium"
            >
              {name}
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
            <Pill
              href={
                trigger.type === 'EVENT'
                  ? (`/env/${environmentSlug}/events/${encodeURIComponent(trigger.value)}` as Route)
                  : undefined
              }
              key={trigger.value}
            >
              <PillContent type={trigger.type}>{trigger.value}</PillContent>
            </Pill>
          );
        });
      },
      header: 'Triggers',
    }),
    columnHelper.accessor('appName', {
      cell: (info) => {
        const appExternalID = info.getValue();
        if (!appExternalID) {
          return null;
        }

        return (
          <Pill href={`/env/${environmentSlug}/apps/${encodeURIComponent(appExternalID)}` as Route}>
            <PillContent type="APP">{appExternalID}</PillContent>
          </Pill>
        );
      },
      header: 'App',
    }),
    columnHelper.accessor('failureRate', {
      cell: (info) => {
        const value = info.getValue();
        if (value === undefined) {
          return <Shimmer className="w-12 px-2.5" />;
        }

        let icon;
        if (value > 0) {
          icon = <RiErrorWarningLine className="-ml-1 mr-1 h-4 w-4 text-rose-500" />;
        }

        return (
          <div className="flex items-center gap-1 px-2.5 text-sm text-slate-600">
            {icon}
            {value}%
          </div>
        );
      },
      header: 'Failure Rate (24hr)',
    }),
    columnHelper.accessor('usage', {
      cell: (info) => {
        const value = info.getValue();
        if (value === undefined) {
          return <Shimmer className="w-[212px] px-2.5" />;
        }

        return (
          <div className="flex min-w-[212px] items-center justify-end gap-2">
            <span
              key="volume-count"
              className="overflow-hidden whitespace-nowrap text-xs text-slate-600"
            >
              <div className="flex items-center gap-1 align-middle text-sm text-slate-600">
                <RiBarChart2Fill className="-ml-0.5 h-3.5 w-3.5 shrink-0 text-indigo-500" />
                {value.total.toLocaleString(undefined, {
                  notation: 'compact',
                  compactDisplay: 'short',
                })}
              </div>
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
