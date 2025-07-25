'use client';

import { useMemo, type ReactElement } from 'react';
import { NumberCell, TextCell, TimeCell } from '@inngest/components/Table';
import NewTable from '@inngest/components/Table/NewTable';
import { createColumnHelper, type ColumnDef } from '@tanstack/react-table';

import { EmptyState } from './EmptyState';
import type { InsightTableRow } from './types';

// Extract the data type from our interface for type safety
type DataType = InsightTableRow['properties'][string]['type'];

const MOCK_SEE_EXAMPLES = () => {
  alert('TODO');
};

type DataTableProps = {
  data: InsightTableRow[];
  isLoading?: boolean;
};

export function DataTable({ data, isLoading = false }: DataTableProps) {
  const columnHelper = createColumnHelper<InsightTableRow>();
  const columns = useMemo(() => generateColumns(data, columnHelper), [data, columnHelper]);

  return (
    <div className="border-subtle flex min-h-0 flex-1 flex-col border">
      <div className="border-subtle flex h-12 shrink-0 items-center justify-between border-b px-4">
        <div className="flex items-center gap-2">
          <h3 className="text-basis text-sm font-medium">Results</h3>
        </div>
      </div>

      <div className="bg-canvasBase flex min-h-0 flex-1 flex-col overflow-y-auto">
        <NewTable
          blankState={<EmptyState onSeeExamples={MOCK_SEE_EXAMPLES} />}
          columns={columns}
          data={data}
          isLoading={isLoading}
        />
      </div>
    </div>
  );
}

function formatColumnHeader(key: string): string {
  return key
    .split('_')
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ');
}

function RowNumberCell({ children }: { children: React.ReactNode }) {
  return (
    <div className="bg-tableHeader text-muted -mx-4 flex h-[42px] w-8 items-center justify-center text-xs font-medium">
      {children}
    </div>
  );
}

function isValidDate(value: unknown): value is string | Date {
  if (typeof value === 'string') return !isNaN(Date.parse(value));
  return value instanceof Date;
}

function isValidNumber(value: unknown): value is number {
  return typeof value === 'number' && !isNaN(value);
}

function renderCellWithTypeCheck(value: unknown, dataType: DataType): ReactElement {
  switch (dataType) {
    case 'date': {
      if (isValidDate(value)) return <TimeCell date={value} />;
      return <TextCell>{String(value)}</TextCell>;
    }
    case 'number': {
      if (isValidNumber(value)) return <NumberCell value={value} />;
      return <TextCell>{String(value)}</TextCell>;
    }
    case 'string':
    default:
      return <TextCell>{String(value)}</TextCell>;
  }
}

function generateColumns(
  data: InsightTableRow[],
  columnHelper: ReturnType<typeof createColumnHelper<InsightTableRow>>
): ColumnDef<InsightTableRow, any>[] {
  const sampleRow = data[0];
  if (sampleRow === undefined) return [];

  // Include the row column first
  const rowColumn = columnHelper.accessor('row', {
    cell: (info) => <RowNumberCell>{info.getValue()}</RowNumberCell>,
    enableSorting: false,
    header: '#',
    size: 32,
  });

  // Then add all the property columns
  const propertyKeys = Object.keys(sampleRow.properties);
  const propertyColumns = propertyKeys.map((key) => {
    const propertyInfo = sampleRow.properties[key];
    const dataType: DataType = propertyInfo?.type ?? 'string';

    return columnHelper.accessor(`properties.${key}.value`, {
      cell: (info) => renderCellWithTypeCheck(info.getValue(), dataType),
      enableSorting: false,
      header: formatColumnHeader(key),
    });
  });

  return [rowColumn, ...propertyColumns];
}
