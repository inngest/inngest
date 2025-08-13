'use client';

import { useMemo } from 'react';
import { Skeleton } from '@inngest/components/Skeleton';
import { TextCell, TimeCell } from '@inngest/components/Table';
import type { ColumnDef } from '@tanstack/react-table';

import type { InsightsFetchResult } from '@/components/Insights/InsightsStateMachineContext/types';

type InsightsEntry = InsightsFetchResult['entries'][number];
type InsightsColumnValue = InsightsEntry['values'][string];
type Column = ColumnDef<InsightsEntry, InsightsColumnValue>;

export function useColumns(data?: InsightsFetchResult): { columns: Column[] } {
  const columns = useMemo(() => {
    const cols = data?.columns ?? [];
    if (cols.length === 0) return [];

    return cols.map(
      (col): ColumnDef<InsightsEntry, InsightsColumnValue> => ({
        accessorKey: `values.${col.name}`,
        cell: ({ getValue, row }) => {
          if (row.original.isLoadingRow) {
            return <Skeleton className="my-2 block h-3" />;
          }

          const value = getValue();

          if (value == null) return <TextCell />;

          switch (col.type) {
            case 'date':
              return <TimeCell date={new Date(value)} />;
            case 'number':
            case 'string':
              return <TextCell>{String(value)}</TextCell>;
          }
        },
        header: col.name,
        id: col.name,
      })
    );
  }, [data?.columns]);

  return { columns };
}
