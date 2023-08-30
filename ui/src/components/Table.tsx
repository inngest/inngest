import { flexRender, useReactTable, type Row, type TableOptions } from '@tanstack/react-table';
import { useVirtual } from 'react-virtual';

import { IconChevron } from '@/icons';
import classNames from '@/utils/classnames';

const cellStyles = 'pl-6 pr-2 py-3 whitespace-nowrap';

type TableProps = {
  options: TableOptions<any>;
  blankState: React.ReactNode;
  customRowProps?: (row: Row<any>) => void;
  tableContainerRef: React.RefObject<HTMLDivElement>;
};

export default function Table({
  options,
  blankState,
  customRowProps,
  tableContainerRef,
}: TableProps) {
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
    <table className="w-full border-b border-slate-700/30">
      <thead className="text-left sticky top-0 z-[3]">
        {table.getHeaderGroups().map((headerGroup) => (
          <tr key={headerGroup.id}>
            {headerGroup.headers.map((header) => (
              <th
                className={classNames(
                  cellStyles,
                  'bg-slate-900 text-slate-500 font-medium',
                  header.column.getIsPinned() && 'sticky left-0 z-[4]',
                  header.column.getCanSort() && 'cursor-pointer',
                )}
                onClick={header.column.getToggleSortingHandler()}
                key={header.id}
                style={{ width: header.getSize() }}
              >
                <div className="flex items-center gap-2">
                  {header.isPlaceholder
                    ? null
                    : flexRender(header.column.columnDef.header, header.getContext())}
                  {header.column.getIsSorted() && options.data.length > 1 && (
                    <IconChevron
                      className={classNames(
                        'icon-xs transition-all duration-500',
                        header.column.getIsSorted() === 'asc' && '-rotate-180',
                      )}
                    />
                  )}
                </div>
              </th>
            ))}
          </tr>
        ))}
      </thead>
      <tbody className="divide-y divide-slate-800/30">
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
            return (
              <tr key={row.id} {...(customRowProps ? customRowProps(row) : {})}>
                {row.getVisibleCells().map((cell) => (
                  <td
                    className={classNames(
                      cellStyles,
                      'bg-slate-950',
                      cell.column.getIsPinned() && 'sticky left-0 z-[2]',
                    )}
                    key={cell.id}
                    style={{ width: cell.column.getSize() }}
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
