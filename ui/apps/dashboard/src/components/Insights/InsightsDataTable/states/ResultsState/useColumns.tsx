'use client';

import { useMemo } from 'react';
import { TextCell, TimeCell } from '@inngest/components/Table';
import type { ColumnDef } from '@tanstack/react-table';

import type { InsightsFetchResult } from '@/components/Insights/InsightsStateMachineContext/types';
import { CellPadding } from './CellPadding';
import { JSONAwareTextCell } from './JSONAwareTextCell';

type InsightsEntry = InsightsFetchResult['rows'][number];
type InsightsColumnValue = InsightsEntry['values'][string];
type Column = ColumnDef<InsightsEntry, InsightsColumnValue>;

export function useColumns(data?: InsightsFetchResult): { columns: Column[] } {
  const columns = useMemo(() => {
    const cols = data?.columns ?? [];
    if (cols.length === 0) return [];

    return cols.map(
      (col): ColumnDef<InsightsEntry, InsightsColumnValue> => ({
        accessorKey: `values.${col.name}`,
        cell: ({ getValue }) => {
          const value = getValue();

          if (value == null)
            return (
              <CellPadding>
                <TextCell />
              </CellPadding>
            );

          switch (col.type) {
            case 'date':
              return (
                <CellPadding>
                  <TimeCell date={new Date(value)} />
                </CellPadding>
              );
            case 'string':
              return <JSONAwareTextCell>{String(value)}</JSONAwareTextCell>;
            case 'number':
            default:
              return (
                <CellPadding>
                  <TextCell>{String(value)}</TextCell>
                </CellPadding>
              );
          }
        },
        header: col.name,
        id: col.name,
      })
    );
  }, [data?.columns]);

  return { columns };
}
