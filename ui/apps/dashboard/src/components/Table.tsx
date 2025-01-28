import { cn } from '@inngest/components/utils/classNames';

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
    <div className="border-subtle w-full overflow-hidden rounded-md border">
      <table className="divide-subtle bg-canvasBase text-basis w-full table-fixed divide-y text-sm">
        <thead className="text-muted h-full text-left">
          <tr>
            {columns.map((column, idx) => (
              <th key={idx} className={cn('p-3 font-semibold', column.className)} scope="col">
                {column.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-subtle h-full divide-y">
          {data.length ? (
            data.map((row, idx) => (
              <tr className="truncate" key={`tr-${idx}`}>
                {columns.map((column, idx) => (
                  <td className={cn('truncate p-3', column.className)} key={`td-${idx}`}>
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
