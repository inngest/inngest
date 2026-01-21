import { Fragment, useMemo } from 'react';
import { Skeleton } from '@inngest/components/Skeleton';
import { cn } from '@inngest/components/utils/classNames';
import { RiSortAsc, RiSortDesc } from '@remixicon/react';
import {
  flexRender,
  getCoreRowModel,
  useReactTable,
  type ColumnDef,
  type OnChangeFn,
  type Row,
  type SortingState,
} from '@tanstack/react-table';

interface WithId {
  id: string;
}

type ExpandableTableProps<T> = {
  renderSubComponent: (props: { row: Row<T> }) => React.ReactElement;
  expandedIDs: string[];
};

type BaseTableProps<T> = {
  data: T[] | undefined;
  sorting?: SortingState;
  setSorting?: OnChangeFn<SortingState>;
  isLoading?: boolean;
  columns: ColumnDef<T, any>[];
  onRowClick?: (row: Row<T>) => void;
  getRowHref?: (row: Row<T>) => string;
  blankState?: React.ReactNode;
  cellClassName?: string;
  noHeader?: boolean;
};

type TableProps<T> = BaseTableProps<T> &
  (T extends WithId
    ? Partial<ExpandableTableProps<T>>
    : { renderSubComponent?: never; expandedIDs?: never });

export function Table<T>({
  data = [],
  isLoading,
  sorting,
  setSorting,
  renderSubComponent,
  onRowClick,
  getRowHref,
  blankState,
  columns,
  expandedIDs = [],
  cellClassName,
  noHeader = false,
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
            cell: () => <Skeleton className="my-2 block h-3" />,
          }))
        : columns,
    [isLoading]
  );

  const table = useReactTable({
    data: tableData,
    columns: tableColumns,
    getCoreRowModel: getCoreRowModel(),
    manualSorting: true,
    onSortingChange: setSorting,
    state: {
      sorting,
    },
  });

  const tableStyles = 'w-full';
  const tableHeadStyles = 'bg-tableHeader sticky top-0 z-[2]';
  const tableColumnStyles = 'px-4';
  const expandedRowSideBorder =
    'before:bg-surfaceMuted relative before:absolute before:bottom-0 before:left-0 before:top-0 before:w-0.5';

  const isEmpty = data.length < 1 && !isLoading;

  // Type guard to check if a row has an id property
  const hasId = <T,>(obj: T): obj is T & WithId => {
    return typeof (obj as any).id === 'string';
  };

  return (
    <div className="">
      <table className={cn(tableStyles)}>
        {!noHeader && (
          <thead className={tableHeadStyles}>
            {table.getHeaderGroups().map((headerGroup) => (
              <tr key={headerGroup.id} className="h-9">
                {headerGroup.headers.map((header) => {
                  const isIconOnlyColumn = header.column.columnDef.header === undefined;
                  return (
                    <th
                      key={header.id}
                      className={cn(
                        isIconOnlyColumn ? '' : tableColumnStyles,
                        'text-muted text-nowrap text-left text-xs font-medium'
                      )}
                    >
                      {header.isPlaceholder ? null : (
                        <div
                          className={cn(
                            header.column.getCanSort()
                              ? 'flex cursor-pointer select-none items-center gap-1'
                              : header.column.getIsSorted()
                              ? 'flex items-center gap-1'
                              : ''
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
        )}
        <tbody>
          {isEmpty && (
            <tr>
              <td
                className="text-muted h-[42px] text-center text-sm font-normal"
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
                  role={getRowHref ? 'link' : undefined}
                  tabIndex={onRowClick ? 0 : undefined}
                  aria-label={getRowHref ? 'View details' : undefined}
                  className={cn(
                    hasId(row.original) && expandedIDs.includes(row.original.id)
                      ? 'h-[42px]'
                      : 'border-light box-border h-[42px] border-b',
                    onRowClick ? 'hover:bg-canvasSubtle cursor-pointer' : ''
                  )}
                  onClick={(e) => {
                    const modalsContainer = document.getElementById('modals');
                    const hasModals = modalsContainer && modalsContainer.children.length > 0;
                    if (hasModals) return;

                    const url = getRowHref?.(row);
                    if (url && (e.metaKey || e.ctrlKey || e.button === 1)) {
                      // Simulate native link behavior
                      window.open(url, '_blank');
                      return;
                    }
                    onRowClick?.(row);
                  }}
                >
                  {row.getVisibleCells().map((cell, i) => {
                    const isIconOnlyColumn = cell.column.columnDef.header === undefined;
                    return (
                      <td
                        key={cell.id}
                        className={cn(
                          i === 0 && hasId(row.original) && expandedIDs.includes(row.original.id)
                            ? expandedRowSideBorder
                            : '',
                          isIconOnlyColumn ? '' : tableColumnStyles,
                          cellClassName ?? ''
                        )}
                      >
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </td>
                    );
                  })}
                </tr>
                {hasId(row.original) &&
                  expandedIDs.includes(row.original.id) &&
                  renderSubComponent &&
                  !isLoading && (
                    <tr>
                      <td
                        colSpan={row.getVisibleCells().length}
                        className={cn(expandedRowSideBorder, 'border-light border-b')}
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
