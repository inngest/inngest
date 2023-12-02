import cn from '@/utils/cn';

type Row = Record<string, any>;

type TableProps<T extends Row> = {
  columns: {
    key: string;
    label?: string;
    className?: string;
    render?: (row: T) => React.ReactNode;
  }[];
  data: Array<T>;
  empty: string;
};

export default function Table<T extends Row>({
  columns,
  data = [],
  empty = 'No results',
}: TableProps<T>) {
  return (
    <div className="w-full overflow-hidden rounded-lg border border-slate-200">
      <table className="w-full table-fixed divide-y divide-slate-200 bg-white text-sm text-slate-500">
        <thead className="h-full text-left">
          <tr>
            {columns.map((column, idx) => (
              <th key={idx} className={cn('p-3 font-semibold', column.className)} scope="col">
                {column.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="h-full divide-y divide-slate-200 text-slate-900">
          {data.length ? (
            data.map((row, idx) => (
              <tr className="truncate" key={`tr-${idx}`}>
                {columns.map((column, idx) => (
                  <td className={cn('p-3', column.className)} key={`td-${idx}`}>
                    {column.render ? column.render(row) : row[column.key]}
                  </td>
                ))}
              </tr>
            ))
          ) : (
            <tr>
              <td className="p-4 text-center" colSpan={columns.length + 1}>
                {empty}
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
