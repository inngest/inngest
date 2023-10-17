import { IconChevron } from '@inngest/components/icons/Chevron';
import { classNames } from '@inngest/components/utils/classNames';
import { flexRender, useReactTable, type Row, type TableOptions } from '@tanstack/react-table';
import { useVirtual } from 'react-virtual';

const cellStyles = 'pl-6 pr-2 py-3 whitespace-nowrap';

type TableProps = {
  options: TableOptions<any>;
  blankState: React.ReactNode;
  customRowProps?: (row: Row<any>) => void;
  tableContainerRef: React.RefObject<HTMLDivElement>;
};

export function Table({ options, blankState, customRowProps, tableContainerRef }: TableProps) {
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
    <table className="dark:bg-slate-910 w-full border-b border-slate-200 bg-white dark:border-slate-700/30">
      <thead className="sticky top-0 z-[3] text-left shadow">
        {table.getHeaderGroups().map((headerGroup) => (
          <tr key={headerGroup.id}>
            {headerGroup.headers.map((header) => (
              <th
                className={classNames(
                  cellStyles,
                  'bg-white font-medium text-slate-600 dark:bg-slate-900 dark:text-slate-500',
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
                    <IconChevron
                      className={classNames(
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
      <tbody className="divide-y divide-slate-100 text-slate-700 dark:divide-slate-800/30 dark:text-slate-400">
        {options.data.length < 1 && (
          <tr>
            <td className={classNames(cellStyles, 'text-center')} colSpan={colSpanTotalSum}>
              {blankState}
            </td>
          </tr>
        )}
        {paddingTop > 0 && (
          <tr>
            <td style={{ height: `${paddingTop}px` }} />
          </tr>
        )}
        {virtualRows &&
          virtualRows.map((virtualRow) => {
            const row = table.getRowModel().rows[virtualRow.index];
            if (!row) return;
            return (
              <tr
                key={row.id}
                {...(customRowProps ? customRowProps(row) : {})}
                className=" dark:bg-slate-910 bg-white hover:bg-slate-100 dark:hover:bg-slate-900"
              >
                {row.getVisibleCells().map((cell) => (
                  <td
                    className={classNames(
                      cellStyles,
                      cell.column.getIsPinned() && 'sticky left-0 z-[2]'
                    )}
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
        {paddingBottom > 0 && (
          <tr>
            <td style={{ height: `${paddingBottom}px` }} />
          </tr>
        )}
      </tbody>
    </table>
  );
}
