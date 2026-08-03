import { useMemo, useState } from 'react';
import { Table } from '@inngest/components/Table';
import type { ColumnDef, OnChangeFn, SortingState } from '@tanstack/react-table';

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
  // 'default' or 'subtle' (no filled header background) — see Table's own
  // headerStyle doc. Passed straight through.
  headerStyle?: 'default' | 'subtle';
  // 'default' or 'compact' (shorter rows) — see Table's own density doc.
  // Passed straight through.
  density?: 'default' | 'compact';
  // Lets the user click any column header to sort rows by it (identifier,
  // function, session key, and every value column) — off by default, since
  // Table's headers otherwise show a clickable/sort-icon affordance with no
  // sorting actually wired up.
  sortable?: boolean;
  // Controlled sort state — when given (with `onSortingChange`), RankedTable
  // renders `items` as-is (assumed already sorted the requested way, e.g. by
  // a server request keyed off this same state) instead of sorting them
  // itself. Omit both for a self-contained client-side sort of the given
  // `items`, tracked in local state.
  sorting?: SortingState;
  onSortingChange?: (sorting: SortingState) => void;
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
  headerStyle,
  density,
  sortable = false,
  sorting: controlledSorting,
  onSortingChange: controlledOnSortingChange,
  className,
}: Props) {
  // Controlled mode (both given): the caller owns sort state, e.g. to
  // re-issue a server request for the new order — `items` is rendered as-is,
  // assumed already sorted. Uncontrolled (neither given): RankedTable sorts
  // the given `items` itself, client-side, tracked in local state.
  const isControlled = controlledSorting !== undefined && controlledOnSortingChange !== undefined;
  const [uncontrolledSorting, setUncontrolledSorting] = useState<SortingState>([]);
  const sorting = isControlled ? controlledSorting : uncontrolledSorting;
  // Table (via tanstack) calls this with either a new value or an updater
  // function — resolve that here so both the controlled callback and the
  // uncontrolled setState only ever see a plain next-value.
  const setSorting: OnChangeFn<SortingState> = (updater) => {
    const next = typeof updater === 'function' ? updater(sorting) : updater;
    if (isControlled) {
      controlledOnSortingChange(next);
    } else {
      setUncontrolledSorting(next);
    }
  };

  const rows = useMemo<Row[]>(
    () => (items ?? []).map((item) => ({ ...item, id: item.identifier })),
    [items],
  );

  // getSortValue mirrors each column's own cell value — used to sort rows
  // client-side (uncontrolled mode only), since Table's `manualSorting:
  // true` means it only tracks which column/direction is active and expects
  // the caller to reorder `data` to match, rather than sorting internally.
  const getSortValue = useMemo(
    () =>
      (row: Row, columnId: string): string | number => {
        if (columnId === 'identifier') return row.identifier;
        if (columnId === 'functionId') return row.functionId ?? '';
        if (columnId === 'sessionKey') return row.sessionKey ?? '';
        const col = columns.find(
          (c) => (typeof c.valueName === 'string' ? c.valueName : c.label) === columnId,
        );
        if (!col) return '';
        const value = typeof col.valueName === 'function' ? col.valueName(row) : valuesToMap(row.values).get(col.valueName);
        // Undefined sorts to the end regardless of direction — toggleable
        // sort direction still only applies to rows that actually have a
        // value for this column.
        return value ?? -Infinity;
      },
    [columns],
  );

  const sortedRows = useMemo<Row[]>(() => {
    if (isControlled) return rows;
    const [sort] = sorting;
    if (!sort) return rows;
    return [...rows].sort((a, b) => {
      const av = getSortValue(a, sort.id);
      const bv = getSortValue(b, sort.id);
      const cmp = typeof av === 'number' && typeof bv === 'number' ? av - bv : String(av).localeCompare(String(bv));
      return sort.desc ? -cmp : cmp;
    });
  }, [rows, sorting, getSortValue, isControlled]);

  const tableColumns = useMemo<ColumnDef<Row, unknown>[]>(() => {
    // tanstack's own getCanSort() requires a truthy `accessorFn` on the
    // column regardless of `enableSorting` (see RowSorting.js) — a plain
    // `{ id, cell }` "display column" (every column here besides
    // identifier, which gets one implicitly via accessorKey) can never be
    // marked sortable without one. manualSorting means tanstack never
    // actually calls this to compute a value itself, so reusing
    // getSortValue here purely satisfies that requirement.
    const identifierColumn: ColumnDef<Row, unknown> = {
      accessorKey: 'identifier',
      header: identifierLabel,
      enableSorting: sortable,
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
            enableSorting: sortable,
            accessorFn: (row) => getSortValue(row, 'functionId'),
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
            enableSorting: sortable,
            accessorFn: (row) => getSortValue(row, 'sessionKey'),
            cell: ({ row }) =>
              row.original.sessionKey ? sessionKeyColumn.render(row.original.sessionKey) : '—',
          },
        ]
      : [];

    const valueColumns: ColumnDef<Row, unknown>[] = columns.map((col) => {
      // Labels are unique within one table's column set, so this is a
      // stable id even when valueName is a function rather than a string.
      const id = typeof col.valueName === 'string' ? col.valueName : col.label;
      return {
        id,
        header: col.label,
        enableSorting: sortable,
        accessorFn: (row) => getSortValue(row, id),
        cell: ({ row }) => {
          const value =
            typeof col.valueName === 'function'
              ? col.valueName(row.original)
              : valuesToMap(row.original.values).get(col.valueName);
          return value === undefined ? '—' : (col.format ?? defaultFormat)(value);
        },
      };
    });

    return [identifierColumn, ...functionColumnDef, ...sessionKeyColumnDef, ...valueColumns];
  }, [columns, identifierLabel, renderIdentifier, functionColumn, sessionKeyColumn, sortable, getSortValue]);

  // Wide identifier column, narrower value/function columns — a rough match
  // for a real row's shape rather than an exact one.
  const columnWidths = useMemo(
    () => tableColumns.map((_, i) => (i === 0 ? 'w-2/5' : 'w-1/6')),
    [tableColumns],
  );

  return (
    <div className={className}>
      <Table
        data={sortedRows}
        columns={tableColumns}
        isLoading={isLoading}
        headerStyle={headerStyle}
        density={density}
        sorting={sortable ? sorting : undefined}
        setSorting={sortable ? setSorting : undefined}
        cellClassName="text-xs"
        blankState={<TableRowsSkeleton columnWidths={columnWidths} />}
      />
    </div>
  );
}
