import { cn } from '@inngest/components/utils/classNames';
import { RiArrowDownSLine } from '@remixicon/react';
import { flexRender, useReactTable, type Row, type TableOptions } from '@tanstack/react-table';
import { useVirtual } from 'react-virtual';

const cellStyles = 'pl-4 pr-2 py-3 whitespace-nowrap';

type TableProps<T> = {
  options: TableOptions<T>;
  blankState: React.ReactNode;
  customRowProps?: (row: Row<T>) => void;
  tableContainerRef: React.RefObject<HTMLDivElement>;
  isVirtualized?: boolean;
};

export function Table<T>({
  options,
  blankState,
  customRowProps,
  tableContainerRef,
  isVirtualized = true,
}: TableProps<T>) {
  const table = useReactTable(options);
  const { rows } = table.getRowModel();

  const rowVirtualizer = useVirtual({
    parentRef: tableContainerRef,
    size: rows?.length,
    overscan: 10,
  });
  const { virtualItems: virtualRows, totalSize } = rowVirtualizer;
  const paddingTop = virtualRows.length > 0 ? virtualRows?.[0]?.start || 0 : 0;
  const paddingBottom =
    virtualRows.length > 0 ? totalSize - (virtualRows?.[virtualRows.length - 1]?.end || 0) : 0;

  // Calculates total colSpan of the table, to assign the colSpan of the blank state.
  // Might need to be changed if we implement the column visibity feature.
  let colSpanTotalSum = table.getHeaderGroups().reduce((sum, headerGroup) => {
    return (
      sum +
      headerGroup.headers.reduce((subSum, header) => {
        return subSum + header.colSpan;
      }, 0)
    );
  }, 0);

  return (
    <table className="bg-canvasBase border-subtle w-full border-b text-sm">
      <thead className="border-subtle sticky top-0 z-[3] border-b text-left">
        {table.getHeaderGroups().map((headerGroup) => (
          <tr key={headerGroup.id}>
            {headerGroup.headers.map((header) => (
              <th
                className={cn(
                  cellStyles,
                  'bg-canvasBase text-muted text-sm font-semibold',
                  header.column.getIsPinned() && 'sticky left-0 z-[4]',
                  header.column.getCanSort() && 'cursor-pointer'
                )}
                onClick={header.column.getToggleSortingHandler()}
                key={header.id}
                style={{
                  width: header.getSize() === Number.MAX_SAFE_INTEGER ? 'auto' : header.getSize(),
                }}
              >
                <div className="flex items-center gap-2">
                  {header.isPlaceholder
                    ? null
                    : flexRender(header.column.columnDef.header, header.getContext())}
                  {header.column.getIsSorted() && options.data.length > 1 && (
                    <RiArrowDownSLine
                      className={cn(
                        'h-3 w-3 transition-all duration-500',
                        header.column.getIsSorted() === 'asc' && '-rotate-180'
                      )}
                    />
                  )}
                </div>
              </th>
            ))}
          </tr>
        ))}
      </thead>
      <tbody className="divide-subtle text-basis divide-y">
        {options.data.length < 1 && (
          <tr>
            <td className={cn(cellStyles, 'text-center')} colSpan={colSpanTotalSum}>
              {blankState}
            </td>
          </tr>
        )}
        {isVirtualized && paddingTop > 0 && (
          <tr>
            <td style={{ height: `${paddingTop}px` }} />
          </tr>
        )}
        {isVirtualized &&
          virtualRows &&
          virtualRows.map((virtualRow) => {
            const row = table.getRowModel().rows[virtualRow.index];
            if (!row) return;
            return (
              <tr
                key={row.id}
                {...(customRowProps ? customRowProps(row) : {})}
                className="bg-canvasBase hover:bg-canvasSubtle/50"
              >
                {row.getVisibleCells().map((cell) => (
                  <td
                    className={cn(cellStyles, cell.column.getIsPinned() && 'sticky left-0 z-[2]')}
                    key={cell.id}
                    style={{
                      width:
                        cell.column.getSize() === Number.MAX_SAFE_INTEGER
                          ? 'auto'
                          : cell.column.getSize(),
                    }}
                  >
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </td>
                ))}
              </tr>
            );
          })}
        {isVirtualized && paddingBottom > 0 && (
          <tr>
            <td style={{ height: `${paddingBottom}px` }} />
          </tr>
        )}
        {!isVirtualized &&
          table.getRowModel().rows.map((row) => (
            <tr
              key={row.id}
              {...(customRowProps ? customRowProps(row) : {})}
              className="bg-canvaseBase hover:bg-canvasSubtle/50"
            >
              {row.getVisibleCells().map((cell) => (
                <td
                  className={cn(cellStyles, cell.column.getIsPinned() && 'sticky left-0 z-[2]')}
                  key={cell.id}
                  style={{
                    width:
                      cell.column.getSize() === Number.MAX_SAFE_INTEGER
                        ? 'auto'
                        : cell.column.getSize(),
                  }}
                >
                  {flexRender(cell.column.columnDef.cell, cell.getContext())}
                </td>
              ))}
            </tr>
          ))}
      </tbody>
    </table>
  );
}
