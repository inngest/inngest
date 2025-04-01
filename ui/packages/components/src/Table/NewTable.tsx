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
  renderSubComponent?: (props: { row: Row<T> }) => React.ReactElement;
  enableExpanding?: boolean;
  columns: ColumnDef<T, any>[];
  getRowCanExpand?: (row: Row<T>) => boolean;
  onRowClick?: (row: Row<T>) => void;
  blankState?: React.ReactNode;
};

export default function Table<T>({
  data = [],
  isLoading,
  sorting,
  setSorting,
  enableExpanding = false,
  getRowCanExpand = () => false,
  renderSubComponent,
  onRowClick,
  blankState,
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
    manualSorting: true,
    onSortingChange: setSorting,
    state: {
      sorting,
    },
  });

  const tableStyles = 'w-full';
  const tableHeadStyles = 'bg-canvasSubtle';
  const tableColumnStyles = 'px-6';
  const expandedRowSideBorder =
    'before:bg-surfaceMuted relative before:absolute before:bottom-0 before:left-0 before:top-0 before:w-0.5';

  const isEmpty = data.length < 1 && !isLoading;

  return (
    <div className="">
      <table className={cn(tableStyles)}>
        <thead className={tableHeadStyles}>
          {table.getHeaderGroups().map((headerGroup) => (
            <tr key={headerGroup.id} className="h-9">
              {headerGroup.headers.map((header) => {
                const isIconOnlyColumn = header.column.columnDef.header === undefined;
                return (
                  <th
                    key={header.id}
                    style={{
                      width: header.getSize(),
                      maxWidth: header.getSize(),
                    }}
                    className={cn(
                      isIconOnlyColumn ? '' : tableColumnStyles,
                      'text-muted text-left text-xs font-medium'
                    )}
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
                          asc: <RiSortAsc className="text-light h-4 w-4" />,
                          desc: <RiSortDesc className="text-light h-4 w-4" />,
                        }[header.column.getIsSorted() as string] ?? null}
                      </div>
                    )}
                  </th>
                );
              })}
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
                {blankState}
              </td>
            </tr>
          )}
          {!isEmpty &&
            table.getRowModel().rows.map((row) => (
              <Fragment key={row.id}>
                <tr
                  className={cn(
                    row.getIsExpanded() ? 'h-12' : 'border-light h-12 border-b',
                    onRowClick ? 'hover:bg-canvasSubtle cursor-pointer' : ''
                  )}
                  onClick={() => {
                    const modalsContainer = document.getElementById('modals');
                    const hasModals = modalsContainer && modalsContainer.children.length > 0;
                    if (!hasModals) {
                      onRowClick?.(row);
                    }
                  }}
                >
                  {row.getVisibleCells().map((cell, i) => {
                    const isIconOnlyColumn = cell.column.columnDef.header === undefined;
                    return (
                      <td
                        key={cell.id}
                        style={{
                          width: cell.column.getSize(),
                          maxWidth: cell.column.getSize(),
                        }}
                        className={cn(
                          i === 0 && row.getIsExpanded() ? expandedRowSideBorder : '',
                          isIconOnlyColumn ? '' : tableColumnStyles
                        )}
                      >
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </td>
                    );
                  })}
                </tr>
                {row.getIsExpanded() && renderSubComponent && !isLoading && (
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
      </table>
    </div>
  );
}

const loadingRow = {
  isLoadingRow: true,
} as const;
