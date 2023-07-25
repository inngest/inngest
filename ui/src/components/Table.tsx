import { useReactTable, flexRender } from '@tanstack/react-table';
import classNames from '@/utils/classnames';

const cellStyles = 'pl-6 pr-2 py-3';

export default function Table({ options }) {
  const table = useReactTable(options);

  return (
    <table className="w-full bg-slate-950 border-b border-slate-700/30">
      <thead className="bg-slate-900 text-left sticky top-0">
        {table.getHeaderGroups().map((headerGroup) => (
          <tr key={headerGroup.id}>
            {headerGroup.headers.map((header) => (
              <th
                className={classNames(cellStyles, 'text-slate-500 font-medium')}
                key={header.id}
              >
                {header.isPlaceholder
                  ? null
                  : flexRender(
                      header.column.columnDef.header,
                      header.getContext()
                    )}
              </th>
            ))}
          </tr>
        ))}
      </thead>
      <tbody className="divide-y divide-slate-800/30">
        {table.getRowModel().rows.map((row) => (
          <tr key={row.id}>
            {row.getVisibleCells().map((cell) => (
              <td className={cellStyles} key={cell.id}>
                {flexRender(cell.column.columnDef.cell, cell.getContext())}
              </td>
            ))}
          </tr>
        ))}
      </tbody>
    </table>
  );
}
