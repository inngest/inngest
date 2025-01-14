import { Fragment, useMemo } from 'react';
import { Skeleton } from '@inngest/components/Skeleton';
import { cn } from '@inngest/components/utils/classNames';
import { RiSortAsc, RiSortDesc } from '@remixicon/react';
import {
  flexRender,
  getCoreRowModel,
  getExpandedRowModel,
  getSortedRowModel,
  useReactTable,
  type ColumnDef,
  type OnChangeFn,
  type Row,
  type SortingState,
} from '@tanstack/react-table';

type TableProps<T> = {
  data: T[] | undefined;
  sorting?: SortingState;
  setSorting?: OnChangeFn<SortingState>;
  isLoading?: boolean;
  renderSubComponent: (props: { row: Row<T> }) => React.ReactElement;
  enableExpanding: boolean;
  columns: ColumnDef<T>[];
  getRowCanExpand: (row: Row<T>) => boolean;
};

export default function CompactPaginatedTable<T>({
  data = [],
  isLoading,
  sorting,
  setSorting,
  enableExpanding,
  getRowCanExpand,
  renderSubComponent,
  columns,
}: TableProps<T>) {
  // Render empty lines for skeletons when data is loading
  const tableData = useMemo(() => {
    if (isLoading) {
      return Array(columns.length)
        .fill(null)
        .map((_, index) => {
          return {
            ...loadingRow,

            // Need an ID to avoid "missing key" errors when rendering rows
            id: index,
          };
        }) as unknown as T[]; // Casting is bad but we need to do this for the loading skeleton
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
    getRowCanExpand,
    getExpandedRowModel: getExpandedRowModel(),
    enableExpanding,
    getSortedRowModel: getSortedRowModel(),
    onSortingChange: setSorting,
    state: {
      sorting,
    },
  });

  const tableStyles =
    'w-full border-subtle rounded-md border border-separate border-spacing-0 overflow-hidden';
  const tableHeadStyles = 'bg-canvasSubtle';
  const tableBodyStyles = 'divide-y divide-subtle';
  const tableColumnStyles = 'px-4';

  const isEmpty = data.length < 1 && !isLoading;

  return (
    <table className={cn(tableStyles)}>
      <thead className={tableHeadStyles}>
        {table.getHeaderGroups().map((headerGroup) => (
          <tr key={headerGroup.id} className="h-12">
            {headerGroup.headers.map((header) => (
              <th
                key={header.id}
                style={{
                  width: header.getSize(),
                }}
                className={cn(tableColumnStyles, 'text-muted text-left text-sm font-normal')}
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
            <td
              className="text-muted h-12 text-center text-sm font-normal"
              colSpan={table.getVisibleFlatColumns().length}
            >
              No results were found.
            </td>
          </tr>
        )}
        {!isEmpty &&
          table.getRowModel().rows.map((row) => (
            <Fragment key={row.id}>
              <tr className="h-12">
                {row.getVisibleCells().map((cell) => {
                  // TO DO: Add left border without layout shift
                  return (
                    <td
                      key={cell.id}
                      style={{
                        width: cell.column.getSize(),
                      }}
                      className={tableColumnStyles}
                    >
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </td>
                  );
                })}
              </tr>
              {row.getIsExpanded() && !isLoading && (
                <tr>
                  <td
                    colSpan={row.getVisibleCells().length}
                    className={cn(
                      row.getIsExpanded()
                        ? 'before:bg-surfaceMuted relative before:absolute before:bottom-0 before:left-0 before:top-0 before:w-0.5'
                        : ''
                    )}
                  >
                    {renderSubComponent({ row })}
                  </td>
                </tr>
              )}
            </Fragment>
          ))}
      </tbody>
      {/* TO DO: Add pagination footer */}
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

const loadingRow = {
  isLoadingRow: true,
} as const;
