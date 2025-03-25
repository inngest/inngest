'use client';

import { useMemo } from 'react';
import { type Route } from 'next';
import { Link } from '@inngest/components/Link';
import { HorizontalPillList, Pill, PillContent } from '@inngest/components/Pill';
import { Skeleton } from '@inngest/components/Skeleton/Skeleton';
import { type Trigger } from '@inngest/components/types/trigger';
import { cn } from '@inngest/components/utils/classNames';
import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  useReactTable,
} from '@tanstack/react-table';

import MiniStackedBarChart from '@/components/Charts/MiniStackedBarChart';
import { useEnvironment } from '@/components/Environments/environment-context';

export type FunctionTableRow = {
  appName: string | null;
  name: string;
  isArchived: boolean;
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
    <main className="bg-canvasBase flex min-h-0 flex-col overflow-y-auto">
      <table className="border-subtle border-b">
        <thead className="shadow-subtle sticky top-0 z-10 shadow-[0_1px_0]">
          {table.getHeaderGroups().map((headerGroup) => (
            <tr key={headerGroup.id} className="h-12">
              {headerGroup.headers.map((header, index) => (
                <th
                  key={header.id}
                  className={cn(
                    'text-muted w-fit whitespace-nowrap px-4 text-left text-sm font-semibold ',
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
        <tbody className="divide-subtle divide-y">
          {table.getRowModel().rows.map((row) => (
            <tr key={row.id} className="hover:bg-canvasSubtle/50">
              {row.getVisibleCells().map((cell, index) => (
                <td
                  key={cell.id}
                  className={cn('whitespace-nowrap', index === columns.length - 1 && 'pr-4')}
                >
                  {flexRender(cell.column.columnDef.cell, cell.getContext())}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>

      {rows.length === 0 && <div className="text-subtle p-3 text-center text-sm">No functions</div>}
    </main>
  );
}

function Shimmer({ className }: { className?: string }) {
  return (
    <div className={`flex ${className}`}>
      <Skeleton className="block h-5 w-full" />
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
          <div className="flex items-center pl-4">
            <div
              className={cn(
                'h-2.5 w-2.5 rounded-full',
                isArchived
                  ? 'bg-surfaceMuted'
                  : isPaused
                  ? 'bg-accent-subtle'
                  : 'bg-primary-moderate'
              )}
            />
            <Link
              key="name"
              href={`/env/${environmentSlug}/functions/${encodeURIComponent(slug)}` as Route}
              arrowOnHover
              className="w-full px-2 py-3 text-sm font-medium"
            >
              {name}
            </Link>
          </div>
        );
      },
      header: 'Function name',
    }),
    columnHelper.accessor('triggers', {
      cell: (props) => {
        const triggers = props.getValue();
        return (
          <HorizontalPillList
            alwaysVisibleCount={2}
            pills={triggers.map((trigger) => {
              return (
                <Pill
                  appearance="outlined"
                  href={
                    trigger.type === 'EVENT'
                      ? (`/env/${environmentSlug}/events/${encodeURIComponent(
                          trigger.value
                        )}` as Route)
                      : undefined
                  }
                  key={trigger.type + trigger.value}
                >
                  <PillContent type={trigger.type}>{trigger.value}</PillContent>
                </Pill>
              );
            })}
          />
        );
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
          <div className="flex items-center">
            <Pill
              appearance="outlined"
              href={`/env/${environmentSlug}/apps/${encodeURIComponent(appExternalID)}` as Route}
            >
              <PillContent type="APP">{appExternalID}</PillContent>
            </Pill>
          </div>
        );
      },
      header: 'App',
    }),
    columnHelper.accessor('failureRate', {
      cell: (info) => {
        const value = info.getValue();
        if (value === undefined) {
          return <Shimmer className="px-2.5" />;
        }

        if (value === 0) {
          return <div className="text-light px-2.5 text-sm">-</div>;
        }

        return (
          <div className={'text-tertiary-intense flex items-center gap-1 px-2.5 text-sm'}>
            {value}%
          </div>
        );
      },
      header: 'Failure rate (24hr)',
    }),
    columnHelper.accessor('usage', {
      cell: (info) => {
        const value = info.getValue();
        if (value === undefined) {
          return <Shimmer className="px-2.5" />;
        }

        return (
          <div className="flex min-w-[212px] items-center justify-end gap-2">
            <span
              key="volume-count"
              className="text-subtle overflow-hidden whitespace-nowrap text-xs"
            >
              <div className="flex items-center gap-1 align-middle text-sm">
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
