import { Fragment, useMemo, useState } from 'react';
import { Skeleton } from '@inngest/components/Skeleton';
import { IDCell, StatusCell, TextCell, TimeCell } from '@inngest/components/Table';
import { type FunctionRunStatus } from '@inngest/components/types/functionRun';
import { cn } from '@inngest/components/utils/classNames';
import { formatMilliseconds } from '@inngest/components/utils/date';
import { RiSortAsc, RiSortDesc } from '@remixicon/react';
import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  getExpandedRowModel,
  useReactTable,
  type OnChangeFn,
  type Row,
  type SortingState,
} from '@tanstack/react-table';

export type Run = {
  status: FunctionRunStatus;
  durationMS: number | null;
  id: string;
  queuedAt: string;
  endedAt: string | null;
  startedAt: string | null;
};

type RunsTableProps = {
  data: Run[] | undefined;
  sorting?: SortingState;
  setSorting?: OnChangeFn<SortingState>;
  isLoading?: boolean;
  renderSubComponent: (props: { id: string }) => React.ReactElement;
  getRowCanExpand: (row: Row<Run>) => boolean;
};

export default function RunsTable({
  data = [],
  isLoading,
  sorting,
  setSorting,
  getRowCanExpand,
  renderSubComponent,
}: RunsTableProps) {
  // Manually track expanded rows because getIsExpanded seems to be index-based,
  // which means polling can shift the expanded row. We may be able to switch
  // back to getIsExpanded when we replace polling with websockets
  const [expandedRunIDs, setExpandedRunIDs] = useState<string[]>([]);

  // Render 8 empty lines for skeletons when data is loading
  const tableData = useMemo(() => {
    if (isLoading) {
      return Array(8)
        .fill(null)
        .map((_, index) => {
          return {
            ...loadingRow,

            // Need an ID to avoid "missing key" errors when rendering rows
            id: index,
          };
        }) as unknown as Run[]; // Casting is bad but we need to do this for the loading skeleton
    }

    return data;
  }, [isLoading, data]);

  const tableColumns = useMemo(
    () =>
      isLoading
        ? columns.map((column) => ({
            ...column,
            cell: () => <Skeleton className="my-4 block h-4" />,
          }))
        : columns,
    [isLoading]
  );

  const table = useReactTable({
    data: tableData,
    columns: tableColumns,
    getCoreRowModel: getCoreRowModel(),
    getExpandedRowModel: getExpandedRowModel(),
    getRowCanExpand,
    manualSorting: true,
    onSortingChange: setSorting,
    state: {
      sorting,
    },
  });

  const tableStyles = 'w-full border-y border-subtle';
  const tableHeadStyles = 'border-b border-subtle';
  const tableBodyStyles = 'divide-y divide-subtle';
  const tableColumnStyles = 'px-4';

  const isEmpty = data.length < 1 && !isLoading;

  return (
    <table className={cn(isEmpty && 'h-full', tableStyles)}>
      <thead className={tableHeadStyles}>
        {table.getHeaderGroups().map((headerGroup) => (
          <tr key={headerGroup.id} className="h-12">
            {headerGroup.headers.map((header) => (
              <th
                key={header.id}
                className={cn(tableColumnStyles, 'text-subtle text-left text-sm font-semibold')}
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
                      asc: <RiSortDesc className="h-4 w-4" />,
                      desc: <RiSortAsc className="h-4 w-4" />,
                    }[header.column.getIsSorted() as string] ?? null}
                  </div>
                )}
              </th>
            ))}
          </tr>
        ))}
      </thead>
      <tbody className={tableBodyStyles}>
        {isEmpty && (
          <tr>
            {/* TODO: when we introduce column visibility options, this colSpan has to be dinamically calculated depending on # visible columns */}
            <td
              className="text-subtle pt-28 text-center align-top font-medium"
              colSpan={table.getAllColumns().length}
            >
              No results were found.
            </td>
          </tr>
        )}
        {!isEmpty &&
          table.getRowModel().rows.map((row) => (
            <Fragment key={row.original.id}>
              <tr
                key={row.original.id}
                className="hover:bg-canvasSubtle/50 h-12 cursor-pointer"
                onClick={() => {
                  if (expandedRunIDs.includes(row.original.id)) {
                    setExpandedRunIDs((prev) => {
                      return prev.filter((id) => id !== row.original.id);
                    });
                  } else {
                    setExpandedRunIDs((prev) => {
                      return [...prev, row.original.id];
                    });
                  }
                }}
              >
                {row.getVisibleCells().map((cell) => (
                  <td className={tableColumnStyles} key={cell.id}>
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </td>
                ))}
              </tr>
              {expandedRunIDs.includes(row.original.id) && !isLoadingRow(row.original) && (
                // Overrides tableStyles divider color
                <tr className="!border-transparent">
                  <td colSpan={row.getVisibleCells().length}>
                    {renderSubComponent({ id: row.original.id })}
                  </td>
                </tr>
              )}
            </Fragment>
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

const columnHelper = createColumnHelper<Run>();

const columns = [
  columnHelper.accessor('status', {
    cell: (info) => {
      const status = info.getValue();

      return (
        <div className="flex items-center">
          <StatusCell status={status} />
        </div>
      );
    },
    header: 'Status',
    enableSorting: false,
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
    enableSorting: false,
  }),
  columnHelper.accessor('queuedAt', {
    cell: (info) => {
      const time = info.getValue();

      return (
        <div className="flex items-center">
          <TimeCell date={new Date(time)} />
        </div>
      );
    },
    header: 'Queued at',
    enableSorting: false,
  }),
  columnHelper.accessor('startedAt', {
    cell: (info) => {
      const time = info.getValue();

      return (
        <div className="flex items-center">
          {time ? <TimeCell date={new Date(time)} /> : <TextCell>-</TextCell>}
        </div>
      );
    },
    header: 'Started at',
    enableSorting: false,
  }),
  columnHelper.accessor('endedAt', {
    cell: (info) => {
      const time = info.getValue();

      return (
        <div className="flex items-center">
          {time ? <TimeCell date={new Date(time)} /> : <TextCell>-</TextCell>}
        </div>
      );
    },
    header: 'Ended at',
    enableSorting: false,
  }),
  columnHelper.accessor('durationMS', {
    cell: (info) => {
      const duration = info.getValue();

      return (
        <div className="flex items-center">
          <TextCell>{duration ? formatMilliseconds(duration) : '-'}</TextCell>
        </div>
      );
    },
    header: 'Duration',
    enableSorting: false,
  }),
];

const loadingRow = {
  isLoadingRow: true,
} as const;

/**
 * Whether a row is a loading skeleton object. This is important because the
 * loading skeleton rows don't have fields, but the row schema requires them
 */
function isLoadingRow(row: Run): boolean {
  return (row as any).isLoadingRow;
}
