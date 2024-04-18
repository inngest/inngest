import { ArrowDownIcon, ArrowUpIcon } from '@heroicons/react/20/solid';
import { FunctionRunStatusIcon } from '@inngest/components/FunctionRunStatusIcon';
import { IDCell, StatusCell, TextCell, TimeCell } from '@inngest/components/Table';
import { type FunctionRunStatus } from '@inngest/components/types/functionRun';
import { cn } from '@inngest/components/utils/classNames';
import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  useReactTable,
  type OnChangeFn,
  type SortingState,
} from '@tanstack/react-table';

import { Time } from '@/components/Time';

export type Run = {
  status: FunctionRunStatus;
  duration: string;
  id: string;
  queuedAt: string;
  endedAt: string;
};

type RunsTableProps = {
  data: Run[] | undefined;
  sorting?: SortingState;
  setSorting?: OnChangeFn<SortingState>;
  isLoading?: boolean;
};

export default function RunsTable({ data = [], isLoading, sorting, setSorting }: RunsTableProps) {
  const columns = useColumns();

  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
    manualSorting: true,
    onSortingChange: setSorting,
    state: {
      sorting,
    },
  });

  // TODO: pass loading to column cells for skeletons
  if (isLoading) return;

  const tableStyles = 'w-full border-y border-slate-200';
  const tableHeadStyles = 'border-b border-slate-200';
  const tableBodyStyles = 'divide-y divide-slate-200';
  const tableColumnStyles = 'px-4';

  return (
    <table className={tableStyles}>
      <thead className={tableHeadStyles}>
        {table.getHeaderGroups().map((headerGroup) => (
          <tr key={headerGroup.id} className="h-12">
            {headerGroup.headers.map((header) => (
              <th
                key={header.id}
                className={cn(tableColumnStyles, 'text-sm font-semibold text-slate-500 ')}
              >
                {header.isPlaceholder ? null : (
                  <div
                    className={cn(
                      header.column.getCanSort() &&
                        'flex cursor-pointer select-none items-center gap-1'
                    )}
                    onClick={header.column.getToggleSortingHandler()}
                  >
                    {flexRender(header.column.columnDef.header, header.getContext())}
                    {{
                      asc: <ArrowDownIcon className="h-4 w-4" />,
                      desc: <ArrowUpIcon className="h-4 w-4" />,
                    }[header.column.getIsSorted() as string] ?? null}
                  </div>
                )}
              </th>
            ))}
          </tr>
        ))}
      </thead>
      <tbody className={tableBodyStyles}>
        {table.getRowModel().rows.map((row) => (
          <tr key={row.id} className="h-12">
            {row.getVisibleCells().map((cell) => (
              <td className={tableColumnStyles} key={cell.id}>
                {flexRender(cell.column.columnDef.cell, cell.getContext())}
              </td>
            ))}
          </tr>
        ))}
      </tbody>
      <tfoot>
        {table.getFooterGroups().map((footerGroup) => (
          <tr key={footerGroup.id}>
            {footerGroup.headers.map((header) => (
              <th key={header.id}>
                {header.isPlaceholder
                  ? null
                  : flexRender(header.column.columnDef.footer, header.getContext())}
              </th>
            ))}
          </tr>
        ))}
      </tfoot>
    </table>
  );
}

function useColumns() {
  const columnHelper = createColumnHelper<Run>();

  return [
    columnHelper.accessor('status', {
      cell: (info) => {
        const status = info.getValue();

        return (
          <div className="flex items-center">
            <StatusCell status={status}>
              <FunctionRunStatusIcon status={status} className="h-5 w-5" />
            </StatusCell>
          </div>
        );
      },
      header: 'Status',
    }),
    columnHelper.accessor('id', {
      cell: (info) => {
        const id = info.getValue();

        return (
          <div className="flex items-center">
            <IDCell>{id}</IDCell>
          </div>
        );
      },
      header: 'Run ID',
    }),
    columnHelper.accessor('queuedAt', {
      cell: (info) => {
        const time = info.getValue();

        return (
          <div className="flex items-center">
            <TimeCell>
              <Time value={new Date(time)} />
            </TimeCell>
          </div>
        );
      },
      header: 'Queued At',
    }),
    columnHelper.accessor('endedAt', {
      cell: (info) => {
        const time = info.getValue();

        return (
          <div className="flex items-center">
            <TimeCell>
              <Time value={new Date(time)} />
            </TimeCell>
          </div>
        );
      },
      header: 'Ended At',
    }),
    columnHelper.accessor('duration', {
      cell: (info) => {
        const duration = info.getValue();

        return (
          <div className="flex items-center">
            <TextCell>
              <p>{duration || '-'}</p>
            </TextCell>
          </div>
        );
      },
      header: 'Duration',
    }),
  ];
}
