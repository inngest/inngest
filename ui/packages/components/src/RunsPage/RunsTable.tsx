import { Fragment, useMemo, useState } from 'react';
import { Button } from '@inngest/components/Button';
import { Skeleton } from '@inngest/components/Skeleton';
import { cn } from '@inngest/components/utils/classNames';
import { RiSortAsc, RiSortDesc } from '@remixicon/react';
import {
  flexRender,
  getCoreRowModel,
  getExpandedRowModel,
  useReactTable,
  type OnChangeFn,
  type Row,
  type SortingState,
  type VisibilityState,
} from '@tanstack/react-table';

import { useScopedColumns } from './columns';
import type { Run, ViewScope } from './types';

type RunsTableProps = {
  data: Run[] | undefined;
  sorting?: SortingState;
  setSorting?: OnChangeFn<SortingState>;
  isLoading?: boolean;
  error?: Error | null;
  onRefresh?: () => void;
  renderSubComponent: (props: Run) => React.ReactElement;
  getRowCanExpand: (row: Row<Run>) => boolean;
  visibleColumns?: VisibilityState;
  scope: ViewScope;
};

export default function RunsTable({
  data = [],
  isLoading,
  error,
  onRefresh,
  sorting,
  setSorting,
  getRowCanExpand,
  renderSubComponent,
  visibleColumns: columnVisibility,
  scope,
}: RunsTableProps) {
  const columns = useScopedColumns(scope);

  // Manually track expanded rows because getIsExpanded seems to be index-based,
  // which means polling can shift the expanded row. We may be able to switch
  // back to getIsExpanded when we replace polling with websockets
  const [expandedRunIDs, setExpandedRunIDs] = useState<string[]>([]);
  const numberOfVisibleColumns =
    columnVisibility && Object.values(columnVisibility).filter((value) => value === true).length;
  // Render 8 empty lines for skeletons when data is loading
  const tableData = useMemo(() => {
    if (isLoading) {
      return Array(numberOfVisibleColumns || columns.length)
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
            cell: () => <Skeleton className="my-4 block h-3" />,
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
      columnVisibility,
    },
  });

  const tableStyles = 'w-full';
  const tableHeadStyles = 'bg-tableHeader sticky top-0 z-[2]';
  const tableBodyStyles = 'divide-y divide-light';
  const tableColumnStyles = 'px-4';

  const isEmpty = data.length < 1 && !isLoading;
  const hasError = error && !isLoading;

  const isTimeout = hasError && isTimeoutError(error);

  const errorMessage = isTimeout
    ? 'Error: request timed out'
    : `Error: ${error?.message || 'An error occurred'}`;

  return (
    <table className={cn(tableStyles, isEmpty && 'border-b-0')}>
      <thead className={tableHeadStyles}>
        {table.getHeaderGroups().map((headerGroup) => (
          <tr key={headerGroup.id} className="h-9">
            {headerGroup.headers.map((header) => (
              <th
                key={header.id}
                className={cn(tableColumnStyles, 'text-muted text-left text-xs font-medium')}
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
        {hasError && (
          <tr>
            <td
              className="pt-28 text-center align-top"
              colSpan={numberOfVisibleColumns || table.getVisibleFlatColumns().length}
            >
              <div className="flex flex-col items-center gap-4">
                <p className="text-muted font-medium">{errorMessage}</p>
                {onRefresh && (
                  <Button label="Retry" kind="primary" appearance="outlined" onClick={onRefresh} />
                )}
              </div>
            </td>
          </tr>
        )}
        {isEmpty && !hasError && (
          <tr>
            <td
              className="text-muted pt-28 text-center align-top font-medium"
              colSpan={numberOfVisibleColumns || table.getVisibleFlatColumns().length}
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
                className="hover:bg-canvasSubtle h-[42px] cursor-pointer"
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
                    {renderSubComponent({ ...row.original })}
                  </td>
                </tr>
              )}
            </Fragment>
          ))}
      </tbody>
      {!isEmpty && (
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
      )}
    </table>
  );
}

/**
 * Check if an error is a timeout error by examining HTTP status codes or error message
 * Handles both 408 (Request Timeout) and 504 (Gateway Timeout) status codes
 */
function isTimeoutError(error: Error): boolean {
  // Check if this is a urql CombinedError with networkError
  const networkError = (error as any)?.networkError;
  const status = networkError?.response?.status;
  if (status === 408 || status === 504) {
    return true;
  }
  // Fallback to message check
  return error.message.toLowerCase().includes('timeout');
}

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
