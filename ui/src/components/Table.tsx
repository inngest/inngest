import { flexRender, useReactTable } from '@tanstack/react-table';

import classNames from '@/utils/classnames';

const cellStyles = 'pl-6 pr-2 py-3 whitespace-nowrap';

export default function Table({ options }) {
  const table = useReactTable(options);

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
        {table.getRowModel().rows.map((row) => (
          <tr key={row.id} {...(options.getRowProps ? options.getRowProps(row) : {})}>
            {row.getVisibleCells().map((cell) => (
              <td
                className={classNames(
                  cellStyles,
                  'bg-slate-950',
                  cell.column.getIsPinned() && 'sticky left-0 z-[2]',
                )}
                key={cell.id}
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
