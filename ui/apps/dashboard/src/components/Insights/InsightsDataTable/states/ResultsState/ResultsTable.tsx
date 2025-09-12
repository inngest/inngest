'use client';

import { memo, useState } from 'react';
import { Table } from '@inngest/components/Table';
import type { ColumnDef } from '@tanstack/react-table';

import { useInsightsStateMachineContext } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import type { InsightsFetchResult } from '@/components/Insights/InsightsStateMachineContext/types';
import { ResultsTableFooter, assertData } from './ResultsTableFooter';
import { mockData } from './mockData';
import { useColumns } from './useColumns';

type InsightsEntry = InsightsFetchResult['rows'][number];
type InsightsColumnValue = InsightsEntry['values'][string];
type InsightsTableProps = {
  cellClassName: string;
  columns: ColumnDef<InsightsEntry, InsightsColumnValue>[];
  data: InsightsEntry[];
};

function InsightsTable({ columns, data, cellClassName }: InsightsTableProps) {
  return <Table<InsightsEntry> cellClassName={cellClassName} columns={columns} data={data} />;
}

const MemoizedInsightsTable = memo(InsightsTable);

export function ResultsTable() {
  const { data } = useInsightsStateMachineContext();
  const [useMockData, setUseMockData] = useState(true);

  const dataToUse = useMockData ? mockData : data;
  const { columns } = useColumns(dataToUse);

  if (!assertData(dataToUse)) return null;

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <div className="flex-1 overflow-auto" id="insights-table-container">
        <MemoizedInsightsTable
          columns={columns}
          data={dataToUse.rows}
          cellClassName="[&:not(:first-child)]:border-l [&:not(:first-child)]:border-light box-border align-top p-2.5"
        />
      </div>
      <ResultsTableFooter />
    </div>
  );
}
