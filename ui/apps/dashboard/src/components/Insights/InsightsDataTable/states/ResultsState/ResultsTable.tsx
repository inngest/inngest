'use client';

import { Table } from '@inngest/components/Table';

import { useInsightsStateMachineContext } from '@/components/Insights/InsightsStateMachineContext/InsightsStateMachineContext';
import { ResultsTableFooter, assertData } from './ResultsTableFooter';
import { useColumns } from './useColumns';

export function ResultsTable() {
  const { data } = useInsightsStateMachineContext();

  const { columns } = useColumns(data);

  if (!assertData(data)) return null;

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <div className="flex-1 overflow-auto" id="insights-table-container">
        <Table
          columns={columns}
          data={data.rows}
          isLoading={false}
          cellClassName="[&:not(:first-child)]:border-l [&:not(:first-child)]:border-light box-border"
        />
      </div>
      <ResultsTableFooter />
    </div>
  );
}
