import { type ColumnDef } from '@tanstack/react-table';

import { type ColumnType, type InsightsColumn, type InsightsEntry } from '../types';

export function formatCellValue(value: Date | string | number | null, type: ColumnType) {
  if (value == null) return <span className="text-muted">-</span>;

  switch (type) {
    case 'number':
      return <span className="font-mono">{(value as number).toLocaleString()}</span>;
    case 'Date':
      const date = new Date(value as string);
      return <span className="font-mono">{date.toLocaleString()}</span>;
    case 'string':
    default:
      return <span>{String(value)}</span>;
  }
}

export function getTableColumns(columns: InsightsColumn[]): ColumnDef<InsightsEntry, any>[] {
  if (!columns.length) return [];

  return columns.map(
    (column): ColumnDef<InsightsEntry, any> => ({
      accessorKey: `values.${column.name}`,
      cell: ({ getValue }) => {
        const value = getValue();
        return formatCellValue(value, column.type);
      },
      header: column.name,
      id: column.name,
    })
  );
}

const SCROLL_THRESHOLD = 200;

export function handleScroll(
  event: React.UIEvent<HTMLDivElement>,
  options: {
    hasEntries: boolean;
    hasNextPage: boolean;
    state: string;
    fetchMore: () => void;
  }
) {
  const { hasEntries, hasNextPage, state, fetchMore } = options;

  if (hasEntries && hasNextPage && state === 'success') {
    const { scrollHeight, scrollTop, clientHeight } = event.target as HTMLDivElement;

    const reachedBottom = scrollHeight - scrollTop - clientHeight < SCROLL_THRESHOLD;
    if (reachedBottom) fetchMore();
  }
}
