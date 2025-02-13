import { Fragment, useMemo } from 'react';
import { Skeleton } from '@inngest/components/Skeleton';
import { cn } from '@inngest/components/utils/classNames';
import { RiSortAsc, RiSortDesc } from '@remixicon/react';
import {
  flexRender,
  getCoreRowModel,
  getExpandedRowModel,
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
  columns: ColumnDef<T, any>[];
  getRowCanExpand: (row: Row<T>) => boolean;
  footer?: React.ReactElement;
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
  footer,
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
    manualSorting: true,
    onSortingChange: setSorting,
    state: {
      sorting,
    },
  });

  const tableStyles = 'w-full';
  const tableHeadStyles = 'bg-canvasSubtle';
  const tableColumnStyles = 'px-4';
  const expandedRowSideBorder =
    'before:bg-surfaceMuted relative before:absolute before:bottom-0 before:left-0 before:top-0 before:w-0.5';

  const isEmpty = data.length < 1 && !isLoading;

  return (
    <div className="border-subtle overflow-hidden rounded-md border">
      <table className={cn(tableStyles)}>
        <thead className={tableHeadStyles}>
          {table.getHeaderGroups().map((headerGroup) => (
            <tr key={headerGroup.id} className="border-subtle h-12 border-b">
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
                        asc: <RiSortAsc className="h-4 w-4" />,
                        desc: <RiSortDesc className="h-4 w-4" />,
                      }[header.column.getIsSorted() as string] ?? null}
                    </div>
                  )}
                </th>
              ))}
            </tr>
          ))}
        </thead>
        <tbody>
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
                <tr
                  className={
                    row.getIsExpanded() ? 'h-12' : 'border-subtle h-12 border-b last:border-b-0'
                  }
                >
                  {row.getVisibleCells().map((cell, i) => {
                    return (
                      <td
                        key={cell.id}
                        style={{
                          width: cell.column.getSize(),
                        }}
                        className={cn(
                          i === 0 && row.getIsExpanded() ? expandedRowSideBorder : '',
                          tableColumnStyles
                        )}
                      >
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </td>
                    );
                  })}
                </tr>
                {row.getIsExpanded() && !isLoading && (
                  <tr className="border-subtle border-b last:border-b-0">
                    <td
                      colSpan={row.getVisibleCells().length}
                      className={cn(row.getIsExpanded() ? expandedRowSideBorder : '')}
                    >
                      {renderSubComponent({ row })}
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
                <td colSpan={table.getAllColumns().length}>{footer}</td>
              </tr>
            ))}
          </tfoot>
        )}
      </table>
    </div>
  );
}

const loadingRow = {
  isLoadingRow: true,
} as const;
