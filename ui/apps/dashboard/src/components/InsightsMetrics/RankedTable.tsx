import { useMemo } from 'react';
import { Table } from '@inngest/components/Table';
import type { ColumnDef } from '@tanstack/react-table';

import { TableRowsSkeleton } from './ChartSkeleton';
import { valuesToMap, type InsightsMetricItem } from './types';

export type RankedTableColumn = {
  // Which NamedValue.name to read, or a function computing the cell's value
  // from the whole row — for anything beyond a single named field (e.g.
  // summing input_tokens + output_tokens into one "tokens used" column).
  // Returns undefined to render '—', same as a missing named value would.
  valueName: string | ((item: InsightsMetricItem) => number | undefined);
  label: string;
  format?: (value: number) => string;
};

type Row = InsightsMetricItem & { id: string };

type Props = {
  items: InsightsMetricItem[] | undefined;
  identifierLabel: string;
  columns: RankedTableColumn[];
  // Renders the identifier cell — defaults to the raw identifier string.
  // Callers use this to resolve a function slug to a display name/link, or
  // a model name to a badge, without RankedTable knowing about either. The
  // full item is also passed so a renderer can pull in other row fields
  // (e.g. sessionKey, needed alongside the session id to link a session).
  renderIdentifier?: (identifier: string, item: InsightsMetricItem) => React.ReactNode;
  // Adds a "Function" column reading each item's `functionId` (present only
  // when the underlying query also selects function_id — e.g. rankings
  // whose rows are runs/steps that each still belong to one function).
  // Omitted entirely if not given, rather than rendering an empty column.
  functionColumn?: {
    label: string;
    render: (functionId: string) => React.ReactNode;
  };
  // Adds a "Session key" column reading each item's `sessionKey` (present
  // only when the underlying query also selects session_key — e.g.
  // most-expensive-sessions, whose `identifier` is a session id that's only
  // unique within its key). Omitted entirely if not given.
  sessionKeyColumn?: {
    label: string;
    render: (sessionKey: string) => React.ReactNode;
  };
  isLoading?: boolean;
  className?: string;
};

const defaultFormat = (value: number) => value.toLocaleString();

// RankedTable renders an InsightsListMetricResult as an ordered top-N table
// (e.g. top functions by cost). Generic over which values it shows as
// columns; the caller supplies `columns`, so this component has no
// AI-specific knowledge. Row order is preserved as given — the data is
// already ranked server-side.
export function RankedTable({
  items,
  identifierLabel,
  columns,
  renderIdentifier,
  functionColumn,
  sessionKeyColumn,
  isLoading = false,
  className,
}: Props) {
  const rows = useMemo<Row[]>(
    () => (items ?? []).map((item) => ({ ...item, id: item.identifier })),
    [items],
  );

  const tableColumns = useMemo<ColumnDef<Row, unknown>[]>(() => {
    const identifierColumn: ColumnDef<Row, unknown> = {
      accessorKey: 'identifier',
      header: identifierLabel,
      cell: ({ row }) =>
        renderIdentifier
          ? renderIdentifier(row.original.identifier, row.original)
          : row.original.identifier,
    };

    const functionColumnDef: ColumnDef<Row, unknown>[] = functionColumn
      ? [
          {
            id: 'functionId',
            header: functionColumn.label,
            cell: ({ row }) =>
              row.original.functionId ? functionColumn.render(row.original.functionId) : '—',
          },
        ]
      : [];

    const sessionKeyColumnDef: ColumnDef<Row, unknown>[] = sessionKeyColumn
      ? [
          {
            id: 'sessionKey',
            header: sessionKeyColumn.label,
            cell: ({ row }) =>
              row.original.sessionKey ? sessionKeyColumn.render(row.original.sessionKey) : '—',
          },
        ]
      : [];

    const valueColumns: ColumnDef<Row, unknown>[] = columns.map((col) => ({
      // Labels are unique within one table's column set, so this is a
      // stable id even when valueName is a function rather than a string.
      id: typeof col.valueName === 'string' ? col.valueName : col.label,
      header: col.label,
      cell: ({ row }) => {
        const value =
          typeof col.valueName === 'function'
            ? col.valueName(row.original)
            : valuesToMap(row.original.values).get(col.valueName);
        return value === undefined ? '—' : (col.format ?? defaultFormat)(value);
      },
    }));

    return [identifierColumn, ...functionColumnDef, ...sessionKeyColumnDef, ...valueColumns];
  }, [columns, identifierLabel, renderIdentifier, functionColumn, sessionKeyColumn]);

  // Wide identifier column, narrower value/function columns — a rough match
  // for a real row's shape rather than an exact one.
  const columnWidths = useMemo(
    () => tableColumns.map((_, i) => (i === 0 ? 'w-2/5' : 'w-1/6')),
    [tableColumns],
  );

  return (
    <div className={className}>
      <Table
        data={rows}
        columns={tableColumns}
        isLoading={isLoading}
        blankState={<TableRowsSkeleton columnWidths={columnWidths} />}
      />
    </div>
  );
}
