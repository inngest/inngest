'use client';

import { useMemo } from 'react';
import { TimeCell } from '@inngest/components/Table';
import type { ColumnDef } from '@tanstack/react-table';

import type { InsightsFetchResult } from '@/components/Insights/InsightsStateMachineContext/types';
import { FormattedDataCell } from './FormattedDataCell';

type InsightsEntry = InsightsFetchResult['rows'][number];
type InsightsColumnValue = InsightsEntry['values'][string];
type Column = ColumnDef<InsightsEntry, InsightsColumnValue>;

export function useColumns(data?: InsightsFetchResult): { columns: Column[] } {
  const columns = useMemo(() => {
    const cols = data?.columns ?? [];
    if (cols.length === 0) return [];

    const jsonObjectColumns = computeJsonColumnsFromFirstNonNullValue(data);

    return cols.map(
      (col): ColumnDef<InsightsEntry, InsightsColumnValue> => ({
        accessorKey: `values.${col.name}`,
        cell: ({ getValue }) => {
          const value = getValue();

          if (value == null) return <FormattedDataCell type="string" value="" />;

          switch (col.type) {
            case 'date':
              return <TimeCell date={new Date(value)} />;
            case 'number':
              return <FormattedDataCell type="number" value={String(value)} />;
            case 'string':
              return (
                <FormattedDataCell
                  type={jsonObjectColumns.has(col.name) ? 'json' : 'string'}
                  value={String(value)}
                />
              );
          }
        },
        header: col.name,
        id: col.name,
      })
    );
  }, [data?.columns, data?.rows]);

  function computeJsonColumnsFromFirstNonNullValue(
    d: InsightsFetchResult | undefined
  ): Set<string> {
    const result = new Set<string>();
    const rows = d?.rows ?? [];
    const columns = d?.columns ?? [];

    for (const col of columns) {
      if (col.type !== 'string') continue;

      const firstValue = getFirstNonNull(rows.map((r) => r.values[col.name]));
      if (typeof firstValue !== 'string') continue;

      if (isJSONObject(String(firstValue))) result.add(col.name);
    }
    return result;
  }

  function getFirstNonNull<T>(values: (T | null | undefined)[]): T | undefined {
    return values.find((v) => v != null) as T | undefined;
  }

  function isJSONObject(text: string): boolean {
    try {
      const parsed = JSON.parse(text);
      return parsed !== null && typeof parsed === 'object';
    } catch {
      return false;
    }
  }

  return { columns };
}
