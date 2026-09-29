import { useMemo } from 'react';
import { LinkCell, TextCell, TimeCell } from '@inngest/components/Table';
import type { ColumnDef } from '@tanstack/react-table';

import type { InsightsFetchResult } from '@/components/Insights/InsightsStateMachineContext/types';
import { InsightsColumnRole } from '@/gql/graphql';
import { usePathCreator } from '@/utils/usePathCreator';

type InsightsEntry = InsightsFetchResult['rows'][number];
type InsightsColumnValue = InsightsEntry['values'][string];
type Column = ColumnDef<InsightsEntry, InsightsColumnValue>;

const COLUMN_SIZE_BY_TYPE: Record<string, number> = {
  date: 200,
  number: 150,
  string: 320,
};

const DEFAULT_COLUMN_SIZE = 340;

// TODO: Support 'json' column type when BE supports it.
export function useColumns(data?: InsightsFetchResult): { columns: Column[] } {
  const pathCreator = usePathCreator();
  const columns = useMemo(() => {
    const cols = data?.columns ?? [];
    if (cols.length === 0) return [];

    return cols.map(
      (col): ColumnDef<InsightsEntry, InsightsColumnValue> => ({
        accessorFn: (row) => row.values[col.name],
        cell: ({ getValue }) => {
          const value = getValue();

          if (value == null) return <TextCell />;

          switch (col.type) {
            case 'date':
              return <TimeCell date={new Date(value)} />;
            case 'string': {
              const id = String(value);
              switch (col.role) {
                case InsightsColumnRole.RunId:
                  return (
                    <LinkCell href={pathCreator.runPopout({ runID: id })}>
                      {id}
                    </LinkCell>
                  );
                case InsightsColumnRole.EventId:
                  return (
                    <LinkCell href={pathCreator.eventPopout({ eventID: id })}>
                      {id}
                    </LinkCell>
                  );
              }

              return <TextCell>{id}</TextCell>;
            }
            case 'number':
            default:
              return <TextCell>{String(value)}</TextCell>;
          }
        },
        header: col.name,
        id: col.name,
        minSize: COLUMN_SIZE_BY_TYPE[col.type] ?? DEFAULT_COLUMN_SIZE,
      }),
    );
  }, [data?.columns, pathCreator]);

  return { columns };
}
