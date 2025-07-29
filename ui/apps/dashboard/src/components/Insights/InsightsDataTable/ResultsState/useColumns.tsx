import { useMemo } from 'react';
import { TextCell, TimeCell } from '@inngest/components/Table';
import type { ColumnDef } from '@tanstack/react-table';

import { type InsightsEntry, type InsightsResult } from '../types';

export function useColumns(data?: InsightsResult): { columns: ColumnDef<InsightsEntry, any>[] } {
  const columns = useMemo(() => {
    const columns = data?.columns || [];
    if (columns.length === 0) return [];

    return columns.map(
      (column): ColumnDef<InsightsEntry, any> => ({
        accessorKey: `values.${column.name}`,
        cell: ({ getValue }) => {
          const value = getValue();

          if (value == null) return <TextCell />;

          switch (column.type) {
            case 'date':
              return <TimeCell date={new Date(value)} />;
            case 'number':
            case 'string':
              return <TextCell>{String(value)}</TextCell>;
          }
        },
        header: column.name,
        id: column.name,
      })
    );
  }, [data?.columns]);

  return { columns };
}
