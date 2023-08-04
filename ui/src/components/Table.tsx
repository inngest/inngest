import { flexRender, useReactTable, type Row, type TableOptions } from '@tanstack/react-table';

import classNames from '@/utils/classnames';

const cellStyles = 'pl-6 pr-2 py-3 whitespace-nowrap';

type TableProps = {
  options: TableOptions<any>;
  blankState: React.ReactNode;
  customRowProps?: (row: Row<any>) => void;
};

export default function Table({ options, blankState, customRowProps }: TableProps) {
  const table = useReactTable(options);

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
                )}
                key={header.id}
                style={{ width: header.getSize() }}
              >
                {header.isPlaceholder
                  ? null
                  : flexRender(header.column.columnDef.header, header.getContext())}
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
        {table.getRowModel().rows.map((row) => (
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
        ))}
      </tbody>
    </table>
  );
}
